package httpapi

import (
	"errors"
	"net/http"

	"higoos/server-cloud/internal/account"
	"higoos/server-cloud/internal/platform"
)

// sessionDTO is the wire shape every login/refresh returns.
type sessionDTO struct {
	User          any    `json:"user"`
	AccessToken   string `json:"accessToken"`
	AccessExpires string `json:"accessExpires"`
	RefreshToken  string `json:"refreshToken"`
}

func toSessionDTO(s account.Session) sessionDTO {
	return sessionDTO{
		User:          s.User,
		AccessToken:   s.AccessToken,
		AccessExpires: s.AccessExpires.UTC().Format(http.TimeFormat),
		RefreshToken:  s.RefreshToken,
	}
}

// POST /v1/auth/sms/start { phone }
func (a *API) smsStart(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body struct {
		Phone string `json:"phone"`
	}
	if !platform.DecodeJSON(w, r, &body) {
		return
	}
	debugCode, err := a.account.StartSMS(r.Context(), body.Phone)
	if err != nil {
		writeAccountErr(w, r, err)
		return
	}
	resp := map[string]any{"sent": true}
	if debugCode != "" {
		resp["debugCode"] = debugCode // dev-only convenience
	}
	platform.WriteJSON(w, r, http.StatusOK, resp)
}

// POST /v1/auth/sms/verify { phone, code }
func (a *API) smsVerify(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if !platform.DecodeJSON(w, r, &body) {
		return
	}
	sess, err := a.account.VerifySMS(r.Context(), body.Phone, body.Code)
	if err != nil {
		writeAccountErr(w, r, err)
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, toSessionDTO(sess))
}

// POST /v1/auth/email/register { email, password }
func (a *API) emailRegister(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !platform.DecodeJSON(w, r, &body) {
		return
	}
	sess, err := a.account.RegisterEmail(r.Context(), body.Email, body.Password)
	if err != nil {
		writeAccountErr(w, r, err)
		return
	}
	platform.WriteJSON(w, r, http.StatusCreated, toSessionDTO(sess))
}

// POST /v1/auth/email/login { email, password }
func (a *API) emailLogin(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !platform.DecodeJSON(w, r, &body) {
		return
	}
	sess, err := a.account.LoginEmail(r.Context(), body.Email, body.Password)
	if err != nil {
		writeAccountErr(w, r, err)
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, toSessionDTO(sess))
}

// POST /v1/auth/apple { identityToken }
func (a *API) appleLogin(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body struct {
		IdentityToken string `json:"identityToken"`
	}
	if !platform.DecodeJSON(w, r, &body) {
		return
	}
	sess, err := a.account.LoginApple(r.Context(), body.IdentityToken)
	if err != nil {
		writeAccountErr(w, r, err)
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, toSessionDTO(sess))
}

// POST /v1/auth/wechat { code }
func (a *API) wechatLogin(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body struct {
		Code string `json:"code"`
	}
	if !platform.DecodeJSON(w, r, &body) {
		return
	}
	sess, err := a.account.LoginWeChat(r.Context(), body.Code)
	if err != nil {
		writeAccountErr(w, r, err)
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, toSessionDTO(sess))
}

// POST /v1/auth/refresh { refreshToken }
func (a *API) authRefresh(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body struct {
		RefreshToken string `json:"refreshToken"`
	}
	if !platform.DecodeJSON(w, r, &body) {
		return
	}
	sess, err := a.account.Refresh(r.Context(), body.RefreshToken)
	if err != nil {
		writeAccountErr(w, r, err)
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, toSessionDTO(sess))
}

// POST /v1/auth/logout { refreshToken }
func (a *API) authLogout(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body struct {
		RefreshToken string `json:"refreshToken"`
	}
	if !platform.DecodeJSON(w, r, &body) {
		return
	}
	_ = a.account.Logout(r.Context(), body.RefreshToken)
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{"ok": true})
}

// writeAccountErr maps domain errors to HTTP status + stable code.
func writeAccountErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, account.ErrInvalidCredentials):
		platform.WriteError(w, r, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
	case errors.Is(err, account.ErrInvalidCode):
		platform.WriteError(w, r, http.StatusUnauthorized, "invalid_code", "invalid or expired verification code")
	case errors.Is(err, account.ErrEmailTaken):
		platform.WriteError(w, r, http.StatusConflict, "email_taken", "email already registered")
	case errors.Is(err, account.ErrWeakInput):
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_input", "invalid input")
	case errors.Is(err, account.ErrRefreshRejected):
		platform.WriteError(w, r, http.StatusUnauthorized, "refresh_rejected", "refresh token rejected")
	default:
		platform.WriteError(w, r, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
