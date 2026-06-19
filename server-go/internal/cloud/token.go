package cloud

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// ErrInvalidToken marks any malformed, mis-signed, wrong-audience or expired
// device access token.
var ErrInvalidToken = errors.New("invalid device access token")

// DeviceClaims is the payload of a cloud-signed device access token. It mirrors
// server-cloud/internal/token.Claims (the cloud is the signer; this is the
// verifier), so the field tags must stay in lock-step.
type DeviceClaims struct {
	Subject     string `json:"sub"` // cloud user id (cu-xxx)
	Audience    string `json:"aud"` // "device:<deviceId>"
	IssuedAt    int64  `json:"iat"`
	ExpiresAt   int64  `json:"exp"`
	DeviceID    string `json:"did"`
	LocalUserID string `json:"luid"`
	Role        string `json:"role"`
}

// VerifyDeviceToken validates a cloud-signed (Ed25519) device access token using
// the cloud public key handed over at registration, checking the signature,
// expiry and that the audience pins the token to this device.
func VerifyDeviceToken(publicKeyHex, token, deviceID string) (DeviceClaims, error) {
	pub, err := hex.DecodeString(publicKeyHex)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return DeviceClaims{}, ErrInvalidToken
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return DeviceClaims{}, ErrInvalidToken
	}
	signing := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !ed25519.Verify(ed25519.PublicKey(pub), []byte(signing), sig) {
		return DeviceClaims{}, ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return DeviceClaims{}, ErrInvalidToken
	}
	var claims DeviceClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return DeviceClaims{}, ErrInvalidToken
	}
	if claims.ExpiresAt != 0 && time.Now().Unix() >= claims.ExpiresAt {
		return DeviceClaims{}, ErrInvalidToken
	}
	if claims.Audience != "device:"+deviceID {
		return DeviceClaims{}, ErrInvalidToken
	}
	return claims, nil
}
