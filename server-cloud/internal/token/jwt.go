// Package token mints and verifies the two token families server-cloud issues.
//
//   - Cloud access tokens: short-lived JWTs (HS256) proving "this caller is cloud
//     user cu-xxx". Only the cloud signs and verifies them, so a shared secret is
//     enough.
//   - Device access tokens: short-lived JWTs (Ed25519) the App presents to a NAS.
//     The NAS verifies them with the cloud's *public* key (handed over at device
//     registration), so the signing key must be asymmetric.
//
// Rather than pull in a JWT dependency, this implements the compact JWS subset we
// need with the standard library only — keeping the cross-compile pure-Go and
// the trust surface small.
package token

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// ErrInvalid is returned for any malformed, mis-signed, wrong-audience or expired
// token. Callers should treat all of these as "unauthenticated".
var ErrInvalid = errors.New("invalid token")

// Claims is the JWT payload shared by both token families. Unused fields stay
// empty and are omitted from the wire form.
type Claims struct {
	Subject     string `json:"sub"`
	Audience    string `json:"aud"`
	IssuedAt    int64  `json:"iat"`
	ExpiresAt   int64  `json:"exp"`
	Display     string `json:"name,omitempty"`
	DeviceID    string `json:"did,omitempty"`
	LocalUserID string `json:"luid,omitempty"`
	Role        string `json:"role,omitempty"`
}

var b64 = base64.RawURLEncoding

// --- HS256 (cloud access tokens) --------------------------------------------

// SignHS256 produces a compact JWS over claims using the shared secret.
func SignHS256(secret []byte, c Claims) (string, error) {
	signing, err := signingInput("HS256", c)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signing))
	sig := b64.EncodeToString(mac.Sum(nil))
	return signing + "." + sig, nil
}

// VerifyHS256 validates signature, expiry and (if expected != "") audience.
func VerifyHS256(secret []byte, token, expectedAud string) (Claims, error) {
	signing, sig, claims, err := split(token)
	if err != nil {
		return Claims{}, err
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signing))
	if !hmac.Equal([]byte(sig), []byte(b64.EncodeToString(mac.Sum(nil)))) {
		return Claims{}, ErrInvalid
	}
	if err := validate(claims, expectedAud); err != nil {
		return Claims{}, err
	}
	return claims, nil
}

// --- Ed25519 (device access tokens) -----------------------------------------

// SignEd25519 produces a compact JWS over claims using the cloud private key.
func SignEd25519(priv ed25519.PrivateKey, c Claims) (string, error) {
	signing, err := signingInput("EdDSA", c)
	if err != nil {
		return "", err
	}
	sig := ed25519.Sign(priv, []byte(signing))
	return signing + "." + b64.EncodeToString(sig), nil
}

// VerifyEd25519 validates signature, expiry and audience with the cloud public
// key. server-go calls this to trust an App-presented device access token.
func VerifyEd25519(pub ed25519.PublicKey, token, expectedAud string) (Claims, error) {
	signing, sigStr, claims, err := split(token)
	if err != nil {
		return Claims{}, err
	}
	sig, err := b64.DecodeString(sigStr)
	if err != nil || !ed25519.Verify(pub, []byte(signing), sig) {
		return Claims{}, ErrInvalid
	}
	if err := validate(claims, expectedAud); err != nil {
		return Claims{}, err
	}
	return claims, nil
}

// --- shared helpers ----------------------------------------------------------

type header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

func signingInput(alg string, c Claims) (string, error) {
	h, err := json.Marshal(header{Alg: alg, Typ: "JWT"})
	if err != nil {
		return "", err
	}
	p, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return b64.EncodeToString(h) + "." + b64.EncodeToString(p), nil
}

func split(token string) (signing, sig string, claims Claims, err error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", "", Claims{}, ErrInvalid
	}
	payload, derr := b64.DecodeString(parts[1])
	if derr != nil {
		return "", "", Claims{}, ErrInvalid
	}
	if jerr := json.Unmarshal(payload, &claims); jerr != nil {
		return "", "", Claims{}, ErrInvalid
	}
	return parts[0] + "." + parts[1], parts[2], claims, nil
}

func validate(c Claims, expectedAud string) error {
	now := time.Now().Unix()
	if c.ExpiresAt != 0 && now >= c.ExpiresAt {
		return ErrInvalid
	}
	if expectedAud != "" && c.Audience != expectedAud {
		return ErrInvalid
	}
	return nil
}
