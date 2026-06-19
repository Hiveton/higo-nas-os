// Package oauth resolves third-party login assertions (Sign in with Apple,
// WeChat) into a stable principal — Apple's `sub` or WeChat's `unionid` — that
// the account service treats like any other login identity.
//
// Both providers have a real path and a dev stub. The real Apple path verifies
// the identity token's RS256 signature against Apple's published JWKS and checks
// audience + issuer; the real WeChat path exchanges the auth code via
// sns/oauth2/access_token. With DevStub=true the assertion is decoded locally so
// the end-to-end flow is exercisable offline — never enable DevStub in production.
package oauth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ErrUnverified is returned when an assertion cannot be trusted.
var ErrUnverified = errors.New("oauth assertion could not be verified")

const (
	appleIssuer  = "https://appleid.apple.com"
	appleJWKSURL = "https://appleid.apple.com/auth/keys"
)

// Apple resolves an Apple identity token to its subject.
type Apple struct {
	ClientID string // expected audience (bundle id)
	DevStub  bool   // when true, decode without signature verification (dev only)

	// Keys supplies RSA public keys by `kid`. Nil uses a cached fetch of Apple's
	// JWKS; tests inject a static source.
	Keys KeySource
	// HTTPClient is used for JWKS fetches. Nil uses a 10s-timeout default.
	HTTPClient *http.Client
}

// KeySource returns the RSA public key for a JWKS key id.
type KeySource interface {
	Key(ctx context.Context, kid string) (*rsa.PublicKey, error)
}

// Verify returns the Apple `sub` for identityToken after verifying its signature,
// audience and issuer. The returned principal is namespaced ("apple:<sub>").
func (a Apple) Verify(identityToken string) (string, error) {
	if a.DevStub {
		claims, err := decodeJWTPayload(identityToken)
		if err != nil || claims.Sub == "" {
			return "", ErrUnverified
		}
		return "apple:" + claims.Sub, nil
	}

	header, claims, signing, sig, err := splitJWT(identityToken)
	if err != nil {
		return "", err
	}
	if header.Alg != "RS256" {
		return "", ErrUnverified
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	source := a.Keys
	if source == nil {
		source = &appleJWKS{client: a.httpClient()}
	}
	pub, err := source.Key(ctx, header.Kid)
	if err != nil {
		return "", ErrUnverified
	}
	hashed := sha256.Sum256([]byte(signing))
	if rsa.VerifyPKCS1v15(pub, crypto.SHA256, hashed[:], sig) != nil {
		return "", ErrUnverified
	}
	if claims.Iss != appleIssuer || claims.Sub == "" {
		return "", ErrUnverified
	}
	if a.ClientID != "" && claims.Aud != a.ClientID {
		return "", ErrUnverified
	}
	if claims.Exp != 0 && time.Now().Unix() >= claims.Exp {
		return "", ErrUnverified
	}
	return "apple:" + claims.Sub, nil
}

func (a Apple) httpClient() *http.Client {
	if a.HTTPClient != nil {
		return a.HTTPClient
	}
	return &http.Client{Timeout: 10 * time.Second}
}

// WeChat resolves a WeChat auth code to a stable unionid (falling back to openid).
type WeChat struct {
	AppID   string
	Secret  string
	DevStub bool

	// BaseURL overrides the WeChat API host (tests point it at httptest).
	BaseURL    string
	HTTPClient *http.Client
}

// Exchange swaps a WeChat auth code for the user's unionid via
// sns/oauth2/access_token. The returned principal is namespaced ("wechat:<id>").
func (w WeChat) Exchange(code string) (string, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", ErrUnverified
	}
	if w.DevStub {
		return "wechat:" + code, nil
	}

	base := w.BaseURL
	if base == "" {
		base = "https://api.weixin.qq.com"
	}
	q := url.Values{}
	q.Set("appid", w.AppID)
	q.Set("secret", w.Secret)
	q.Set("code", code)
	q.Set("grant_type", "authorization_code")

	client := w.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/sns/oauth2/access_token?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var payload struct {
		OpenID  string `json:"openid"`
		UnionID string `json:"unionid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.ErrCode != 0 {
		return "", ErrUnverified
	}
	id := payload.UnionID
	if id == "" {
		id = payload.OpenID
	}
	if id == "" {
		return "", ErrUnverified
	}
	return "wechat:" + id, nil
}

// --- JWKS fetch + cache ------------------------------------------------------

type appleJWKS struct {
	client *http.Client
	mu     sync.Mutex
	keys   map[string]*rsa.PublicKey
	expiry time.Time
}

func (j *appleJWKS) Key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if time.Now().Before(j.expiry) {
		if k, ok := j.keys[kid]; ok {
			return k, nil
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appleJWKSURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := j.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var set struct {
		Keys []struct {
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return nil, err
	}
	j.keys = map[string]*rsa.PublicKey{}
	for _, k := range set.Keys {
		if pub, err := rsaKey(k.N, k.E); err == nil {
			j.keys[k.Kid] = pub
		}
	}
	j.expiry = time.Now().Add(time.Hour)
	if k, ok := j.keys[kid]; ok {
		return k, nil
	}
	return nil, ErrUnverified
}

// StaticKeySource is a KeySource backed by an in-memory map — used in tests.
type StaticKeySource map[string]*rsa.PublicKey

func (s StaticKeySource) Key(_ context.Context, kid string) (*rsa.PublicKey, error) {
	if k, ok := s[kid]; ok {
		return k, nil
	}
	return nil, ErrUnverified
}

// rsaKey builds an RSA public key from base64url modulus/exponent (JWK form).
func rsaKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, err
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(new(big.Int).SetBytes(eBytes).Int64()),
	}, nil
}

// --- JWT helpers -------------------------------------------------------------

type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}

type jwtClaims struct {
	Sub string `json:"sub"`
	Aud string `json:"aud"`
	Iss string `json:"iss"`
	Exp int64  `json:"exp"`
}

func splitJWT(tok string) (jwtHeader, jwtClaims, string, []byte, error) {
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		return jwtHeader{}, jwtClaims{}, "", nil, ErrUnverified
	}
	hRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return jwtHeader{}, jwtClaims{}, "", nil, ErrUnverified
	}
	var h jwtHeader
	if err := json.Unmarshal(hRaw, &h); err != nil {
		return jwtHeader{}, jwtClaims{}, "", nil, ErrUnverified
	}
	c, err := decodeJWTPayload(tok)
	if err != nil {
		return jwtHeader{}, jwtClaims{}, "", nil, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return jwtHeader{}, jwtClaims{}, "", nil, ErrUnverified
	}
	return h, c, parts[0] + "." + parts[1], sig, nil
}

func decodeJWTPayload(tok string) (jwtClaims, error) {
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		return jwtClaims{}, ErrUnverified
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return jwtClaims{}, ErrUnverified
	}
	var c jwtClaims
	if err := json.Unmarshal(payload, &c); err != nil {
		return jwtClaims{}, ErrUnverified
	}
	return c, nil
}
