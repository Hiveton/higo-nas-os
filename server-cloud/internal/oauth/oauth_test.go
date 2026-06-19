package oauth_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"higoos/server-cloud/internal/oauth"
)

// signRS256 builds a signed Apple-style identity token for testing.
func signRS256(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	enc := base64.RawURLEncoding
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": kid, "typ": "JWT"})
	payload, _ := json.Marshal(claims)
	signing := enc.EncodeToString(header) + "." + enc.EncodeToString(payload)
	hashed := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashed[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signing + "." + enc.EncodeToString(sig)
}

func TestAppleVerifyReal(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	source := oauth.StaticKeySource{"key-1": &key.PublicKey}
	apple := oauth.Apple{ClientID: "com.hiveton.higoos", Keys: source}

	valid := signRS256(t, key, "key-1", map[string]any{
		"sub": "001234.abcd", "aud": "com.hiveton.higoos",
		"iss": "https://appleid.apple.com", "exp": time.Now().Add(time.Hour).Unix(),
	})
	sub, err := apple.Verify(valid)
	if err != nil || sub != "apple:001234.abcd" {
		t.Fatalf("valid token: sub=%q err=%v", sub, err)
	}

	// Wrong audience.
	badAud := signRS256(t, key, "key-1", map[string]any{
		"sub": "x", "aud": "com.someone.else",
		"iss": "https://appleid.apple.com", "exp": time.Now().Add(time.Hour).Unix(),
	})
	if _, err := apple.Verify(badAud); err == nil {
		t.Fatal("wrong audience should fail")
	}

	// Expired.
	expired := signRS256(t, key, "key-1", map[string]any{
		"sub": "x", "aud": "com.hiveton.higoos",
		"iss": "https://appleid.apple.com", "exp": time.Now().Add(-time.Hour).Unix(),
	})
	if _, err := apple.Verify(expired); err == nil {
		t.Fatal("expired token should fail")
	}

	// Unknown kid.
	badKid := signRS256(t, key, "key-unknown", map[string]any{
		"sub": "x", "aud": "com.hiveton.higoos",
		"iss": "https://appleid.apple.com", "exp": time.Now().Add(time.Hour).Unix(),
	})
	if _, err := apple.Verify(badKid); err == nil {
		t.Fatal("unknown kid should fail")
	}

	// Tampered signature (flip the audience after signing).
	tampered := strings.Replace(valid, valid[strings.Index(valid, ".")+1:], "tampered", 1)
	if _, err := apple.Verify(tampered); err == nil {
		t.Fatal("tampered token should fail")
	}
}

func TestWeChatExchangeReal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sns/oauth2/access_token" || r.URL.Query().Get("code") != "good-code" {
			_ = json.NewEncoder(w).Encode(map[string]any{"errcode": 40029, "errmsg": "invalid code"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"openid": "o123", "unionid": "u456", "access_token": "tok"})
	}))
	defer srv.Close()

	wechat := oauth.WeChat{AppID: "app", Secret: "sec", BaseURL: srv.URL}
	id, err := wechat.Exchange("good-code")
	if err != nil || id != "wechat:u456" {
		t.Fatalf("exchange: id=%q err=%v", id, err)
	}
	if _, err := wechat.Exchange("bad-code"); err == nil {
		t.Fatal("bad code should fail")
	}
}
