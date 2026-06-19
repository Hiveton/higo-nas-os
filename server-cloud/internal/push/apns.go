package push

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// Sender delivers a notification to one APNs device token.
type Sender interface {
	Send(ctx context.Context, deviceToken, title, body string) error
}

// APNsConfig is the token-based (.p8) APNs auth material.
type APNsConfig struct {
	KeyPath  string // path to the AuthKey_XXXX.p8
	KeyID    string
	TeamID   string
	BundleID string // apns-topic
	// Host overrides the APNs endpoint (production default). Tests point it at a
	// local server; use "https://api.sandbox.push.apple.com" for the sandbox.
	Host string
}

// APNsSender signs a per-connection provider JWT (ES256) and POSTs alerts over
// HTTP/2. The provider token is cached and rotated every ~40 minutes per Apple's
// guidance.
type APNsSender struct {
	cfg    APNsConfig
	key    *ecdsa.PrivateKey
	client *http.Client

	mu       sync.Mutex
	jwt      string
	jwtAt    time.Time
}

// NewAPNsSender loads the .p8 key and returns a ready sender.
func NewAPNsSender(cfg APNsConfig) (*APNsSender, error) {
	if cfg.KeyPath == "" || cfg.KeyID == "" || cfg.TeamID == "" || cfg.BundleID == "" {
		return nil, errors.New("APNs requires key path, key id, team id and bundle id")
	}
	if cfg.Host == "" {
		cfg.Host = "https://api.push.apple.com"
	}
	pemBytes, err := os.ReadFile(cfg.KeyPath)
	if err != nil {
		return nil, err
	}
	key, err := parseP8(pemBytes)
	if err != nil {
		return nil, err
	}
	return &APNsSender{cfg: cfg, key: key, client: &http.Client{Timeout: 10 * time.Second}}, nil
}

// Send delivers an alert notification to one device token.
func (s *APNsSender) Send(ctx context.Context, deviceToken, title, body string) error {
	jwt, err := s.providerToken()
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{
		"aps": map[string]any{
			"alert": map[string]string{"title": title, "body": body},
			"sound": "default",
		},
	})
	url := s.cfg.Host + "/3/device/" + deviceToken
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("authorization", "bearer "+jwt)
	req.Header.Set("apns-topic", s.cfg.BundleID)
	req.Header.Set("apns-push-type", "alert")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("apns status %d", resp.StatusCode)
	}
	return nil
}

func (s *APNsSender) providerToken() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.jwt != "" && time.Since(s.jwtAt) < 40*time.Minute {
		return s.jwt, nil
	}
	enc := base64.RawURLEncoding
	header, _ := json.Marshal(map[string]string{"alg": "ES256", "kid": s.cfg.KeyID})
	claims, _ := json.Marshal(map[string]any{"iss": s.cfg.TeamID, "iat": time.Now().Unix()})
	signing := enc.EncodeToString(header) + "." + enc.EncodeToString(claims)
	sig, err := signES256(s.key, signing)
	if err != nil {
		return "", err
	}
	s.jwt = signing + "." + enc.EncodeToString(sig)
	s.jwtAt = time.Now()
	return s.jwt, nil
}

// signES256 returns the JWS ECDSA signature (r||s, each left-padded to 32 bytes).
func signES256(key *ecdsa.PrivateKey, signing string) ([]byte, error) {
	hashed := sha256.Sum256([]byte(signing))
	r, sVal, err := ecdsa.Sign(rand.Reader, key, hashed[:])
	if err != nil {
		return nil, err
	}
	out := make([]byte, 64)
	r.FillBytes(out[:32])
	sVal.FillBytes(out[32:])
	return out, nil
}

func parseP8(pemBytes []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("invalid .p8 PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New(".p8 is not an ECDSA key")
	}
	return key, nil
}
