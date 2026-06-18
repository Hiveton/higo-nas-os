package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDoUnwrapsEnvelopeData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/system/info" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("space"); got != "home" {
			t.Errorf("query space = %q, want home", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{"version":"1.2.3"},"requestId":"r1"}`))
	}))
	defer srv.Close()

	c := NewRemote(srv.URL, Auth{}, nil)
	q := map[string][]string{"space": {"home"}}
	data, err := c.Do(context.Background(), http.MethodGet, "/api/v1/system/info", q, nil)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	var out struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if out.Version != "1.2.3" {
		t.Fatalf("version = %q, want 1.2.3", out.Version)
	}
}

func TestDoReturnsTypedErrorFromEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"files_unavailable","message":"down"}}`))
	}))
	defer srv.Close()

	c := NewRemote(srv.URL, Auth{}, nil)
	_, err := c.Do(context.Background(), http.MethodGet, "/api/v1/files/tree", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error type = %T, want *Error", err)
	}
	if apiErr.Status != http.StatusServiceUnavailable || apiErr.Code != "files_unavailable" {
		t.Fatalf("got status=%d code=%q", apiErr.Status, apiErr.Code)
	}
}

func TestDoForwardsAuth(t *testing.T) {
	var gotAuth, gotCookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if ck, err := r.Cookie("higo_session"); err == nil {
			gotCookie = ck.Value
		}
		_, _ = w.Write([]byte(`{"ok":true,"data":null}`))
	}))
	defer srv.Close()

	c := NewRemote(srv.URL, Auth{}, nil).WithAuth(Auth{Authorization: "Bearer xyz", SessionCookie: "sess1"})
	if _, err := c.Do(context.Background(), http.MethodGet, "/api/v1/system/info", nil, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if gotAuth != "Bearer xyz" {
		t.Errorf("Authorization = %q, want Bearer xyz", gotAuth)
	}
	if gotCookie != "sess1" {
		t.Errorf("higo_session cookie = %q, want sess1", gotCookie)
	}
}

func TestInProcessClientRoundTrips(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{"path":"` + r.URL.Path + `"}}`))
	})
	c := NewInProcess(handler, Auth{})
	data, err := c.Do(context.Background(), http.MethodGet, "/api/v1/desktop/apps", nil, nil)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if string(data) != `{"path":"/api/v1/desktop/apps"}` {
		t.Fatalf("unexpected data: %s", data)
	}
}
