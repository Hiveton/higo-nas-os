package push_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"higoos/server-cloud/internal/push"
)

// writeTestP8 generates a P-256 key and writes it as a PKCS8 .p8 file.
func writeTestP8(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	path := filepath.Join(t.TempDir(), "AuthKey_TEST.p8")
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestAPNsSend(t *testing.T) {
	var gotAuth, gotTopic, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("authorization")
		gotTopic = r.Header.Get("apns-topic")
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sender, err := push.NewAPNsSender(push.APNsConfig{
		KeyPath:  writeTestP8(t),
		KeyID:    "ABC123KEYID",
		TeamID:   "TEAM123456",
		BundleID: "com.hiveton.higoos",
		Host:     srv.URL,
	})
	if err != nil {
		t.Fatalf("NewAPNsSender: %v", err)
	}

	if err := sender.Send(context.Background(), "devicetoken123", "标题", "正文"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.HasPrefix(gotAuth, "bearer ") || strings.Count(gotAuth, ".") != 2 {
		t.Fatalf("expected a bearer JWT, got %q", gotAuth)
	}
	if gotTopic != "com.hiveton.higoos" {
		t.Fatalf("apns-topic = %q", gotTopic)
	}
	if gotPath != "/3/device/devicetoken123" {
		t.Fatalf("path = %q", gotPath)
	}
}

func TestAPNsRequiresConfig(t *testing.T) {
	if _, err := push.NewAPNsSender(push.APNsConfig{KeyID: "x"}); err == nil {
		t.Fatal("expected error for incomplete APNs config")
	}
}
