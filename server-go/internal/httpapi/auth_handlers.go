package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/accounts"
	"higoos/server-go/internal/audit"
	"higoos/server-go/internal/platform"
)

const (
	sessionCookieName = "higo_session"
	csrfCookieName    = "higo_csrf"
)

// loginThrottle counts consecutive failed logins per username so the API can
// lock an account after too many attempts.
type loginThrottle struct {
	mu       sync.Mutex
	failures map[string]int
}

func newLoginThrottle() *loginThrottle {
	return &loginThrottle{failures: make(map[string]int)}
}

func (t *loginThrottle) record(username string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	key := strings.ToLower(username)
	t.failures[key]++
	return t.failures[key]
}

func (t *loginThrottle) reset(username string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.failures, strings.ToLower(username))
}

type currentUserDTO struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	Role        string   `json:"role"`
	Status      string   `json:"status"`
	QuotaBytes  int64    `json:"quotaBytes"`
	Groups      []string `json:"groups"`
	Permissions []string `json:"permissions"`
	MFAEnabled  bool     `json:"mfaEnabled"`
	CSRFToken   string   `json:"csrfToken,omitempty"`
}

type authSessionDTO struct {
	ID         string    `json:"id"`
	Device     string    `json:"device"`
	IPAddress  string    `json:"ipAddress"`
	UserAgent  string    `json:"userAgent,omitempty"`
	Current    bool      `json:"current"`
	CreatedAt  time.Time `json:"createdAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
}

func (a *API) authLogin(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body accounts.LoginRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	user, err := a.accounts.VerifyPassword(r.Context(), body.Username, body.Password)
	if err != nil {
		a.recordLoginFailure(r, body.Username)
		a.auditAuth(r, body.Username, "", "login", audit.ResultDenied, audit.RiskMedium)
		if errors.Is(err, accounts.ErrAccountInactive) {
			platform.WriteError(w, r, http.StatusForbidden, "account_inactive", "account is disabled or locked")
			return
		}
		platform.WriteError(w, r, http.StatusUnauthorized, "invalid_credentials", "invalid username or password")
		return
	}
	// Second factor: if the account has TOTP enabled, a valid code is required.
	if a.accounts.MFAEnabled(user.ID) {
		if strings.TrimSpace(body.Code) == "" {
			platform.WriteError(w, r, http.StatusUnauthorized, "mfa_required", "verification code required")
			return
		}
		if !a.accounts.MFAVerify(user.ID, body.Code) {
			a.auditAuth(r, body.Username, "", "login_mfa", audit.ResultDenied, audit.RiskMedium)
			platform.WriteError(w, r, http.StatusUnauthorized, "mfa_invalid", "invalid verification code")
			return
		}
	}
	a.loginThrottle.reset(body.Username)

	session, err := a.auth.Issue(user.ID, "", clientIP(r), r.UserAgent())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "session_issue_failed", err.Error())
		return
	}
	a.setSessionCookie(w, session.ID, int(a.config.SessionTTL.Seconds()))
	a.setCSRFCookie(w, session.CSRFToken, int(a.config.SessionTTL.Seconds()))
	a.auditAuth(r, user.Username, session.ID, "login", audit.ResultAllowed, audit.RiskLow)
	platform.WriteJSON(w, r, http.StatusOK, currentUserResponse(user, session.CSRFToken, a.accounts.MFAEnabled(user.ID)))
}

func (a *API) authLogout(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		a.auth.Revoke(cookie.Value)
		if principal, ok := platform.PrincipalFromContext(r.Context()); ok {
			a.auditAuth(r, principal.Username, cookie.Value, "logout", audit.ResultAllowed, audit.RiskLow)
		}
	}
	a.clearSessionCookie(w)
	platform.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (a *API) authMe(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	principal, ok := platform.PrincipalFromContext(r.Context())
	if !ok {
		platform.WriteError(w, r, http.StatusUnauthorized, "unauthorized", "no active session")
		return
	}
	csrf := ""
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		if session, valid := a.auth.Validate(cookie.Value, time.Now().UTC()); valid {
			csrf = session.CSRFToken
		}
	}
	if user, found := a.accounts.GetUser(r.Context(), principal.UserID); found {
		platform.WriteJSON(w, r, http.StatusOK, currentUserResponse(user, csrf, a.accounts.MFAEnabled(user.ID)))
		return
	}
	// Implicit dev-admin (or a principal without a stored account record).
	platform.WriteJSON(w, r, http.StatusOK, currentUserDTO{
		ID:          principal.UserID,
		Username:    principal.Username,
		DisplayName: principal.Username,
		Role:        principal.Role,
		Status:      string(accounts.StatusActive),
		Groups:      []string{},
		Permissions: derivePermissions(principal.Role),
		CSRFToken:   csrf,
	})
}

func (a *API) authPassword(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	principal, ok := platform.PrincipalFromContext(r.Context())
	if !ok {
		platform.WriteError(w, r, http.StatusUnauthorized, "unauthorized", "no active session")
		return
	}
	var body accounts.ChangePasswordRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if _, err := a.accounts.VerifyPassword(r.Context(), principal.Username, body.CurrentPassword); err != nil {
		platform.WriteError(w, r, http.StatusForbidden, "invalid_credentials", "current password is incorrect")
		return
	}
	if err := a.accounts.SetUserPassword(r.Context(), principal.UserID, body.NewPassword); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "password_change_failed", err.Error())
		return
	}
	// Invalidate other sessions; keep the caller's current one signed in.
	a.auth.RevokeAllForUser(principal.UserID, principal.SessionID)
	a.auditAuth(r, principal.Username, principal.SessionID, "password_change", audit.ResultAllowed, audit.RiskMedium)
	platform.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "password_changed"})
}

func (a *API) authSessions(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	principal, ok := platform.PrincipalFromContext(r.Context())
	if !ok {
		platform.WriteError(w, r, http.StatusUnauthorized, "unauthorized", "no active session")
		return
	}
	targetUser := principal.UserID
	if q := strings.TrimSpace(r.URL.Query().Get("userId")); q != "" && principal.IsAdmin() {
		targetUser = q
	}
	sessions := a.auth.ListByUser(targetUser)
	out := make([]authSessionDTO, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, authSessionDTO{
			ID:         s.ID,
			Device:     s.UserAgent,
			IPAddress:  s.SourceIP,
			UserAgent:  s.UserAgent,
			Current:    s.ID == principal.SessionID,
			CreatedAt:  s.CreatedAt,
			LastSeenAt: s.LastSeenAt,
		})
	}
	platform.WriteJSON(w, r, http.StatusOK, out)
}

func (a *API) authSessionByID(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/auth/sessions/"), "/")
	if id == "" {
		platform.WriteError(w, r, http.StatusNotFound, "auth_session_route_not_found", "session id is required")
		return
	}
	if !allowMethod(w, r, http.MethodDelete) {
		return
	}
	principal, ok := platform.PrincipalFromContext(r.Context())
	if !ok {
		platform.WriteError(w, r, http.StatusUnauthorized, "unauthorized", "no active session")
		return
	}
	// Only the owner (or an admin) may revoke a session.
	owned := false
	for _, s := range a.auth.ListByUser(principal.UserID) {
		if s.ID == id {
			owned = true
			break
		}
	}
	if !owned && !principal.IsAdmin() {
		platform.WriteError(w, r, http.StatusForbidden, "forbidden", "cannot revoke another user's session")
		return
	}
	if !a.auth.Revoke(id) {
		platform.WriteError(w, r, http.StatusNotFound, "auth_session_not_found", "session not found")
		return
	}
	a.auditAuth(r, principal.Username, id, "session_revoke", audit.ResultAllowed, audit.RiskLow)
	platform.WriteJSON(w, r, http.StatusOK, mapTaskResponse(id, "revoked", "session revoked"))
}

type authAuditDTO struct {
	ID       string    `json:"id"`
	Time     time.Time `json:"time"`
	Actor    string    `json:"actor"`
	Action   string    `json:"action"`
	Domain   string    `json:"domain"`
	Result   string    `json:"result"`
	Risk     string    `json:"risk"`
	SourceIP string    `json:"sourceIp"`
}

// authAudit returns the account/auth audit trail (login, password change, MFA,
// lockouts, authorization denials), newest first. Admin only.
func (a *API) authAudit(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	principal, ok := platform.PrincipalFromContext(r.Context())
	if !ok || !principal.IsAdmin() {
		platform.WriteError(w, r, http.StatusForbidden, "forbidden", "admin role required")
		return
	}
	out := make([]authAuditDTO, 0)
	if a.audit != nil {
		events := a.audit.List()
		for i := len(events) - 1; i >= 0; i-- {
			e := events[i]
			out = append(out, authAuditDTO{
				ID:       e.ID,
				Time:     e.Time,
				Actor:    e.ActorID,
				Action:   e.Action,
				Domain:   e.Domain,
				Result:   string(e.Result),
				Risk:     string(e.Risk),
				SourceIP: e.SourceIP,
			})
		}
	}
	platform.WriteJSON(w, r, http.StatusOK, out)
}

// --- helpers ---------------------------------------------------------------

func (a *API) recordLoginFailure(r *http.Request, username string) {
	count := a.loginThrottle.record(username)
	if a.config.LoginMaxFailures <= 0 || count < a.config.LoginMaxFailures {
		return
	}
	if user, ok := a.accounts.GetUserByUsername(r.Context(), username); ok {
		_ = a.accounts.LockUser(r.Context(), user.ID)
		a.auditAuth(r, username, "", "account_locked", audit.ResultBlocked, audit.RiskHigh)
	}
}

func currentUserResponse(user accounts.User, csrf string, mfaEnabled bool) currentUserDTO {
	groups := user.Groups
	if groups == nil {
		groups = []string{}
	}
	return currentUserDTO{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        string(user.Role),
		Status:      string(user.Status),
		QuotaBytes:  user.QuotaBytes,
		Groups:      groups,
		Permissions: derivePermissions(string(user.Role)),
		MFAEnabled:  mfaEnabled,
		CSRFToken:   csrf,
	}
}

func (a *API) sameSite() http.SameSite {
	switch strings.ToLower(a.config.CookieSameSite) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func (a *API) cookieSecure() bool {
	return a.config.Environment != "dev" || a.sameSite() == http.SameSiteNoneMode
}

func (a *API) setSessionCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cookieSecure(),
		SameSite: a.sameSite(),
		MaxAge:   maxAge,
	})
}

func (a *API) setCSRFCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: false, // readable by the frontend to echo as X-CSRF-Token
		Secure:   a.cookieSecure(),
		SameSite: a.sameSite(),
		MaxAge:   maxAge,
	})
}

func (a *API) clearSessionCookie(w http.ResponseWriter) {
	for _, name := range []string{sessionCookieName, csrfCookieName} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: name == sessionCookieName,
			Secure:   a.cookieSecure(),
			SameSite: a.sameSite(),
			MaxAge:   -1,
		})
	}
}

func (a *API) auditAuth(r *http.Request, actor, sessionID, action string, result audit.AuditResult, risk audit.RiskLevel) {
	if a.audit == nil {
		return
	}
	a.audit.Append(audit.AuditEvent{
		RequestID: platform.RequestIDFromContext(r.Context()),
		ActorID:   actor,
		SessionID: sessionID,
		SourceIP:  clientIP(r),
		Domain:    "auth",
		Action:    action,
		Risk:      risk,
		Result:    result,
	})
}

func (a *API) auditAuthz(r *http.Request, principal platform.Principal, action, target string) {
	if a.audit == nil {
		return
	}
	a.audit.Append(audit.AuditEvent{
		RequestID:   platform.RequestIDFromContext(r.Context()),
		ActorID:     principal.Username,
		SessionID:   principal.SessionID,
		SourceIP:    clientIP(r),
		Domain:      "authz",
		Action:      action,
		TargetScope: target,
		Risk:        audit.RiskMedium,
		Result:      audit.ResultDenied,
	})
}

// resolvePrincipal turns a request into an authenticated principal using the
// session cookie. Returns false when there is no valid session.
func (a *API) resolvePrincipal(r *http.Request) (platform.Principal, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return platform.Principal{}, false
	}
	session, ok := a.auth.Validate(cookie.Value, time.Now().UTC())
	if !ok {
		return platform.Principal{}, false
	}
	user, found := a.accounts.GetUser(r.Context(), session.UserID)
	if !found || user.Status != accounts.StatusActive {
		return platform.Principal{}, false
	}
	return platform.Principal{
		UserID:    user.ID,
		Username:  user.Username,
		Role:      string(user.Role),
		SessionID: session.ID,
		DeviceID:  session.DeviceID,
	}, true
}

// devAdminPrincipal is injected when auth is not required and no valid session
// is present, so the desktop keeps working in development.
func (a *API) devAdminPrincipal(r *http.Request) platform.Principal {
	if user, ok := a.accounts.GetUserByUsername(r.Context(), "admin"); ok {
		return platform.Principal{UserID: user.ID, Username: user.Username, Role: string(user.Role), Implicit: true}
	}
	return platform.Principal{UserID: "admin", Username: "admin", Role: "admin", Implicit: true}
}
