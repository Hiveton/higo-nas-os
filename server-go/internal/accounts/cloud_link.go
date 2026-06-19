package accounts

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// LinkCloudAccount provisions (or returns the existing) NAS-local user that backs
// a HiGoOS cloud account. The cloud calls this over the relay tunnel during the
// account-binding flow; the returned User.ID is the localUserId the cloud records
// in the binding and embeds in every device access token.
//
// The local user has a random password it never uses — cloud-bound users
// authenticate by presenting a cloud-signed device access token, not a password.
// role is the cloud-side role ("admin" for the first binder, else "user").
func (s *Service) LinkCloudAccount(ctx context.Context, cloudUserID, role string) (User, error) {
	username := cloudUsername(cloudUserID)
	if existing, ok := s.GetUserByUsername(ctx, username); ok {
		return existing, nil
	}
	password, err := randomSecret()
	if err != nil {
		return User{}, err
	}
	r := RoleUser
	if role == string(RoleAdmin) {
		r = RoleAdmin
	}
	return s.CreateUser(ctx, CreateUserRequest{
		Username:    username,
		DisplayName: "云账号 " + shortCloudID(cloudUserID),
		Password:    password,
		Role:        r,
	})
}

// UnlinkCloudAccount removes the local user backing a cloud account (best-effort).
func (s *Service) UnlinkCloudAccount(ctx context.Context, cloudUserID string) error {
	user, ok := s.GetUserByUsername(ctx, cloudUsername(cloudUserID))
	if !ok {
		return nil
	}
	return s.DeleteUser(ctx, user.ID)
}

// cloudUsername maps a cloud user id to a deterministic, system-safe username.
func cloudUsername(cloudUserID string) string {
	id := strings.ToLower(cloudUserID)
	id = strings.ReplaceAll(id, "cu-", "")
	if len(id) > 12 {
		id = id[:12]
	}
	return "cloud-" + id
}

func shortCloudID(cloudUserID string) string {
	id := strings.TrimPrefix(strings.ToLower(cloudUserID), "cu-")
	if len(id) > 6 {
		return id[:6]
	}
	return id
}

func randomSecret() (string, error) {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
