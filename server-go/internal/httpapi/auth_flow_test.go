package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"higoos/server-go/internal/accounts"
	"higoos/server-go/internal/httpapi"
	"higoos/server-go/internal/platform"
)

// buildAuthRouter returns a router whose accounts service has a known admin
// password and which enforces auth + CSRF (Environment "prod").
func buildAuthRouter(t *testing.T) http.Handler {
	t.Helper()
	acct := accounts.NewService()
	if _, err := acct.BootstrapAdmin(context.Background(), "Admin1234"); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	return httpapi.NewRouter(httpapi.Dependencies{
		Config:   platform.Config{Environment: "prod", Version: "test"},
		Accounts: acct,
	})
}

func cookieValue(resp *http.Response, name string) string {
	for _, c := range resp.Cookies() {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

func TestAuthMeRequiresSession(t *testing.T) {
	router := buildAuthRouter(t)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for /auth/me without session, got %d", rec.Code)
	}
}

func TestAuthLoginFlow(t *testing.T) {
	router := buildAuthRouter(t)

	// Wrong password is rejected.
	badRec := httptest.NewRecorder()
	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin","password":"nope"}`))
	router.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password, got %d", badRec.Code)
	}

	// Correct password issues a session + CSRF cookie.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin","password":"Admin1234"}`))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 login, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := rec.Result()
	session := cookieValue(resp, "higo_session")
	csrf := cookieValue(resp, "higo_csrf")
	if session == "" || csrf == "" {
		t.Fatalf("expected session and csrf cookies, got session=%q csrf=%q", session, csrf)
	}

	// /auth/me with the cookie returns the admin identity.
	meRec := httptest.NewRecorder()
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReq.AddCookie(&http.Cookie{Name: "higo_session", Value: session})
	router.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("expected 200 /auth/me, got %d", meRec.Code)
	}
	var me struct {
		Data struct {
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"data"`
	}
	if err := json.Unmarshal(meRec.Body.Bytes(), &me); err != nil {
		t.Fatalf("decode me: %v", err)
	}
	if me.Data.Username != "admin" || me.Data.Role != "admin" {
		t.Fatalf("unexpected me payload: %+v", me.Data)
	}

	// A write without the CSRF header is rejected.
	noCSRF := httptest.NewRecorder()
	wReq := httptest.NewRequest(http.MethodPost, "/api/v1/accounts/users", bytes.NewBufferString(`{"username":"x","password":"Passw0rd1"}`))
	wReq.AddCookie(&http.Cookie{Name: "higo_session", Value: session})
	router.ServeHTTP(noCSRF, wReq)
	if noCSRF.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without CSRF, got %d", noCSRF.Code)
	}

	// With the CSRF header it succeeds (admin creating a user).
	okRec := httptest.NewRecorder()
	okReq := httptest.NewRequest(http.MethodPost, "/api/v1/accounts/users", bytes.NewBufferString(`{"username":"lin","displayName":"Lin","password":"Passw0rd1","role":"user"}`))
	okReq.AddCookie(&http.Cookie{Name: "higo_session", Value: session})
	okReq.AddCookie(&http.Cookie{Name: "higo_csrf", Value: csrf})
	okReq.Header.Set("X-CSRF-Token", csrf)
	router.ServeHTTP(okRec, okReq)
	if okRec.Code != http.StatusOK {
		t.Fatalf("expected 200 create user, got %d: %s", okRec.Code, okRec.Body.String())
	}
}

func TestNonAdminBlockedFromAdminRoutes(t *testing.T) {
	router := buildAuthRouter(t)

	// Log in as admin and create a non-admin user.
	adminSession, adminCSRF := login(t, router, "admin", "Admin1234")
	createUser(t, router, adminSession, adminCSRF, `{"username":"bob","displayName":"Bob","password":"Passw0rd1","role":"user"}`)

	// Log in as bob and attempt an admin-only write.
	bobSession, bobCSRF := login(t, router, "bob", "Passw0rd1")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts/users", bytes.NewBufferString(`{"username":"eve","password":"Passw0rd1"}`))
	req.AddCookie(&http.Cookie{Name: "higo_session", Value: bobSession})
	req.AddCookie(&http.Cookie{Name: "higo_csrf", Value: bobCSRF})
	req.Header.Set("X-CSRF-Token", bobCSRF)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin creating a user, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthAuditAdminOnly(t *testing.T) {
	router := buildAuthRouter(t)
	adminSession, adminCSRF := login(t, router, "admin", "Admin1234")

	// Admin can read the audit trail.
	adminRec := httptest.NewRecorder()
	adminReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/audit", nil)
	adminReq.AddCookie(&http.Cookie{Name: "higo_session", Value: adminSession})
	router.ServeHTTP(adminRec, adminReq)
	if adminRec.Code != http.StatusOK {
		t.Fatalf("expected admin 200 on /auth/audit, got %d", adminRec.Code)
	}

	// A non-admin is forbidden.
	createUser(t, router, adminSession, adminCSRF, `{"username":"viewer","password":"Passw0rd1","role":"user"}`)
	viewerSession, _ := login(t, router, "viewer", "Passw0rd1")
	viewerRec := httptest.NewRecorder()
	viewerReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/audit", nil)
	viewerReq.AddCookie(&http.Cookie{Name: "higo_session", Value: viewerSession})
	router.ServeHTTP(viewerRec, viewerReq)
	if viewerRec.Code != http.StatusForbidden {
		t.Fatalf("expected non-admin 403 on /auth/audit, got %d", viewerRec.Code)
	}
}

func login(t *testing.T, router http.Handler, username, password string) (session, csrf string) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"`+username+`","password":"`+password+`"}`))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login %s failed: %d %s", username, rec.Code, rec.Body.String())
	}
	resp := rec.Result()
	return cookieValue(resp, "higo_session"), cookieValue(resp, "higo_csrf")
}

func createUser(t *testing.T, router http.Handler, session, csrf, body string) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts/users", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "higo_session", Value: session})
	req.AddCookie(&http.Cookie{Name: "higo_csrf", Value: csrf})
	req.Header.Set("X-CSRF-Token", csrf)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create user failed: %d %s", rec.Code, rec.Body.String())
	}
}
