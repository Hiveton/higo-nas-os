package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"higoos/server-cloud/internal/httpapi"
	"higoos/server-cloud/internal/platform"
)

// testServer spins up the full router against the memory store in test mode.
func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	cfg := platform.Config{Environment: "test", VerificationDebug: true, AuthRequired: false}
	handler, err := httpapi.NewRouter(httpapi.Dependencies{Config: cfg})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// envelope mirrors platform.Envelope for decoding in tests.
type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func do(t *testing.T, srv *httptest.Server, method, path, bearer string, body any, headers map[string]string) (int, envelope) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, srv.URL+path, rdr)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	var env envelope
	_ = json.NewDecoder(resp.Body).Decode(&env)
	return resp.StatusCode, env
}

func TestHealth(t *testing.T) {
	srv := testServer(t)
	status, env := do(t, srv, http.MethodGet, "/healthz", "", nil, nil)
	if status != http.StatusOK || !env.OK {
		t.Fatalf("health: status=%d ok=%v", status, env.OK)
	}
}

// TestSMSLoginFlow covers issue -> verify -> session -> refresh.
func TestSMSLoginFlow(t *testing.T) {
	srv := testServer(t)

	status, env := do(t, srv, http.MethodPost, "/v1/auth/sms/start", "", map[string]string{"phone": "13800001111"}, nil)
	if status != http.StatusOK {
		t.Fatalf("sms start status=%d", status)
	}
	var start struct {
		DebugCode string `json:"debugCode"`
	}
	mustData(t, env, &start)
	if start.DebugCode == "" {
		t.Fatal("expected debugCode in test mode")
	}

	status, env = do(t, srv, http.MethodPost, "/v1/auth/sms/verify", "",
		map[string]string{"phone": "13800001111", "code": start.DebugCode}, nil)
	if status != http.StatusOK {
		t.Fatalf("sms verify status=%d err=%v", status, env.Error)
	}
	var sess struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	mustData(t, env, &sess)
	if sess.AccessToken == "" || sess.RefreshToken == "" || !strings.HasPrefix(sess.User.ID, "cu-") {
		t.Fatalf("bad session: %+v", sess)
	}

	// /v1/account/me with the access token returns this user.
	status, env = do(t, srv, http.MethodGet, "/v1/account/me", sess.AccessToken, nil, nil)
	if status != http.StatusOK {
		t.Fatalf("account/me status=%d", status)
	}

	// Refresh rotates tokens.
	status, env = do(t, srv, http.MethodPost, "/v1/auth/refresh", "",
		map[string]string{"refreshToken": sess.RefreshToken}, nil)
	if status != http.StatusOK {
		t.Fatalf("refresh status=%d err=%v", status, env.Error)
	}
	// Old refresh token is now revoked.
	status, _ = do(t, srv, http.MethodPost, "/v1/auth/refresh", "",
		map[string]string{"refreshToken": sess.RefreshToken}, nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("reused refresh should be rejected, got %d", status)
	}
}

func TestEmailRegisterAndLogin(t *testing.T) {
	srv := testServer(t)
	creds := map[string]string{"email": "Nas@Example.com", "password": "supersecret"}

	status, env := do(t, srv, http.MethodPost, "/v1/auth/email/register", "", creds, nil)
	if status != http.StatusCreated {
		t.Fatalf("register status=%d err=%v", status, env.Error)
	}
	// Duplicate registration conflicts.
	status, _ = do(t, srv, http.MethodPost, "/v1/auth/email/register", "", creds, nil)
	if status != http.StatusConflict {
		t.Fatalf("dup register should conflict, got %d", status)
	}
	// Login works (and email is case-insensitive).
	status, env = do(t, srv, http.MethodPost, "/v1/auth/email/login", "",
		map[string]string{"email": "nas@example.com", "password": "supersecret"}, nil)
	if status != http.StatusOK {
		t.Fatalf("login status=%d err=%v", status, env.Error)
	}
	// Wrong password is rejected.
	status, _ = do(t, srv, http.MethodPost, "/v1/auth/email/login", "",
		map[string]string{"email": "nas@example.com", "password": "wrong"}, nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("bad password should be 401, got %d", status)
	}
}

// TestPairingBindFlow covers device register -> pairing issue (device-authed) ->
// claim (cloud-authed) -> binding + device token -> access ticket.
func TestPairingBindFlow(t *testing.T) {
	srv := testServer(t)
	access := loginViaSMS(t, srv, "13900002222")

	// NAS self-registers.
	status, env := do(t, srv, http.MethodPost, "/v1/devices/register", "",
		map[string]string{"deviceId": "hg-test01", "model": "HiGoOS Mini", "version": "1.2.0"}, nil)
	if status != http.StatusOK {
		t.Fatalf("device register status=%d", status)
	}
	var reg struct {
		DeviceSecret   string `json:"deviceSecret"`
		Serial         string `json:"serial"`
		CloudPublicKey string `json:"cloudPublicKey"`
	}
	mustData(t, env, &reg)
	if reg.DeviceSecret == "" || reg.Serial == "" || len(reg.CloudPublicKey) != 64 {
		t.Fatalf("bad registration: %+v", reg)
	}

	// NAS publishes a pairing code (device-authenticated).
	status, env = do(t, srv, http.MethodPost, "/v1/bindings/pairing/issue", "",
		map[string]string{"kind": "code"},
		map[string]string{"X-Device-Id": "hg-test01", "X-Device-Secret": reg.DeviceSecret})
	if status != http.StatusOK {
		t.Fatalf("pairing issue status=%d err=%v", status, env.Error)
	}
	var pairing struct {
		Secret string `json:"secret"`
	}
	mustData(t, env, &pairing)

	// App claims the code (cloud-authenticated).
	status, env = do(t, srv, http.MethodPost, "/v1/bindings/pairing/claim", access,
		map[string]string{"code": pairing.Secret}, nil)
	if status != http.StatusCreated {
		t.Fatalf("pairing claim status=%d err=%v", status, env.Error)
	}
	var bound struct {
		DeviceToken string `json:"deviceToken"`
		Binding     struct {
			Role string `json:"role"`
		} `json:"binding"`
	}
	mustData(t, env, &bound)
	if bound.DeviceToken == "" || bound.Binding.Role != "admin" {
		t.Fatalf("expected admin binding + device token, got %+v", bound)
	}

	// The code is single-use.
	status, _ = do(t, srv, http.MethodPost, "/v1/bindings/pairing/claim", access,
		map[string]string{"code": pairing.Secret}, nil)
	if status == http.StatusCreated {
		t.Fatal("pairing code should be single-use")
	}

	// Bindings list shows the device.
	status, env = do(t, srv, http.MethodGet, "/v1/bindings", access, nil, nil)
	if status != http.StatusOK {
		t.Fatalf("bindings list status=%d", status)
	}

	// Access ticket re-mints a device token.
	status, env = do(t, srv, http.MethodPost, "/v1/devices/hg-test01/access-ticket", access, nil, nil)
	if status != http.StatusOK {
		t.Fatalf("access-ticket status=%d err=%v", status, env.Error)
	}

	// Relay forward returns 503 while the device is offline.
	status, _ = do(t, srv, http.MethodGet, "/d/hg-test01/api/v1/system/health", access, nil, nil)
	if status != http.StatusServiceUnavailable {
		t.Fatalf("offline relay should be 503, got %d", status)
	}
}

func loginViaSMS(t *testing.T, srv *httptest.Server, phone string) (accessToken string) {
	t.Helper()
	_, env := do(t, srv, http.MethodPost, "/v1/auth/sms/start", "", map[string]string{"phone": phone}, nil)
	var start struct {
		DebugCode string `json:"debugCode"`
	}
	mustData(t, env, &start)
	_, env = do(t, srv, http.MethodPost, "/v1/auth/sms/verify", "",
		map[string]string{"phone": phone, "code": start.DebugCode}, nil)
	var sess struct {
		AccessToken string `json:"accessToken"`
	}
	mustData(t, env, &sess)
	if sess.AccessToken == "" {
		t.Fatal("login failed")
	}
	return sess.AccessToken
}

func mustData(t *testing.T, env envelope, dst any) {
	t.Helper()
	if !env.OK {
		t.Fatalf("envelope not ok: %+v", env.Error)
	}
	if err := json.Unmarshal(env.Data, dst); err != nil {
		t.Fatalf("decode data: %v", err)
	}
}
