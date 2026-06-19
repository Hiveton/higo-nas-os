// Package account is the cloud identity domain: it turns a proven login
// (phone+SMS, email+password, Apple, WeChat) into a HiGoOS cloud account and a
// token pair. One account aggregates many login identities, so a user who signed
// up by phone can later add Apple and sign in either way.
package account

import (
	"context"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"higoos/server-cloud/internal/auth"
	"higoos/server-cloud/internal/oauth"
	"higoos/server-cloud/internal/platform"
	"higoos/server-cloud/internal/store"
	"higoos/server-cloud/internal/verification"
)

// Common domain errors. Handlers map these to HTTP codes.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidCode        = errors.New("invalid or expired code")
	ErrEmailTaken         = errors.New("email already registered")
	ErrWeakInput          = errors.New("invalid input")
	ErrRefreshRejected    = errors.New("refresh token rejected")
)

// Session is the result of any successful authentication.
type Session struct {
	User          store.User `json:"user"`
	AccessToken   string     `json:"accessToken"`
	AccessExpires time.Time  `json:"accessExpires"`
	RefreshToken  string     `json:"refreshToken"`
}

// Service wires the store, token manager, verification and oauth providers.
type Service struct {
	store    store.Store
	tokens   *auth.Manager
	verify   *verification.Service
	apple    oauth.Apple
	wechat   oauth.WeChat
}

// NewService builds the account service.
func NewService(s store.Store, tokens *auth.Manager, v *verification.Service, apple oauth.Apple, wechat oauth.WeChat) *Service {
	return &Service{store: s, tokens: tokens, verify: v, apple: apple, wechat: wechat}
}

// --- Phone + SMS -------------------------------------------------------------

// StartSMS issues a verification code to the phone number. The returned string
// is the dev-debug code (empty outside dev).
func (s *Service) StartSMS(_ context.Context, phone string) (string, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return "", ErrWeakInput
	}
	return s.verify.Issue(verification.ChannelSMS, phone)
}

// VerifySMS checks the code and logs the caller in (auto-registering on first
// use), returning a fresh session.
func (s *Service) VerifySMS(ctx context.Context, phone, code string) (Session, error) {
	phone = strings.TrimSpace(phone)
	if !s.verify.Verify(verification.ChannelSMS, phone, code) {
		return Session{}, ErrInvalidCode
	}
	user, err := s.findOrCreateByIdentity(ctx, store.IdentityPhone, phone, "", maskPhone(phone))
	if err != nil {
		return Session{}, err
	}
	return s.issueSession(ctx, user)
}

// --- Email + password --------------------------------------------------------

// RegisterEmail creates an account with an email+password identity.
func (s *Service) RegisterEmail(ctx context.Context, email, password string) (Session, error) {
	email = normalizeEmail(email)
	if email == "" || len(password) < 8 {
		return Session{}, ErrWeakInput
	}
	if _, err := s.store.GetIdentity(ctx, store.IdentityEmail, email); err == nil {
		return Session{}, ErrEmailTaken
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Session{}, err
	}
	user, err := s.createUserWithIdentity(ctx, store.IdentityEmail, email, string(hash), emailLocalPart(email))
	if err != nil {
		return Session{}, err
	}
	// A real deployment would send a verification email here
	// (s.verify.Issue(ChannelEmail, email)); the account is usable immediately.
	return s.issueSession(ctx, user)
}

// LoginEmail authenticates an email+password identity.
func (s *Service) LoginEmail(ctx context.Context, email, password string) (Session, error) {
	email = normalizeEmail(email)
	id, err := s.store.GetIdentity(ctx, store.IdentityEmail, email)
	if err != nil {
		return Session{}, ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(id.SecretHash), []byte(password)) != nil {
		return Session{}, ErrInvalidCredentials
	}
	user, err := s.store.GetUser(ctx, id.UserID)
	if err != nil {
		return Session{}, err
	}
	return s.issueSession(ctx, user)
}

// --- Apple / WeChat ----------------------------------------------------------

// LoginApple verifies an Apple identity token and logs in (auto-registering).
func (s *Service) LoginApple(ctx context.Context, identityToken string) (Session, error) {
	sub, err := s.apple.Verify(identityToken)
	if err != nil {
		return Session{}, ErrInvalidCredentials
	}
	user, err := s.findOrCreateByIdentity(ctx, store.IdentityApple, sub, "", "Apple 用户")
	if err != nil {
		return Session{}, err
	}
	return s.issueSession(ctx, user)
}

// LoginWeChat exchanges a WeChat code and logs in (auto-registering).
func (s *Service) LoginWeChat(ctx context.Context, code string) (Session, error) {
	unionid, err := s.wechat.Exchange(code)
	if err != nil {
		return Session{}, ErrInvalidCredentials
	}
	user, err := s.findOrCreateByIdentity(ctx, store.IdentityWeChat, unionid, "", "微信用户")
	if err != nil {
		return Session{}, err
	}
	return s.issueSession(ctx, user)
}

// --- Refresh / logout --------------------------------------------------------

// Refresh rotates a refresh token: the old one is revoked and a new access +
// refresh pair is issued. Rejects expired, revoked or unknown tokens.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (Session, error) {
	hash := auth.HashToken(refreshToken)
	rt, err := s.store.GetRefreshToken(ctx, hash)
	if err != nil || rt.RevokedAt != nil || time.Now().After(rt.ExpiresAt) {
		return Session{}, ErrRefreshRejected
	}
	if err := s.store.RevokeRefreshToken(ctx, rt.ID, time.Now()); err != nil {
		return Session{}, err
	}
	user, err := s.store.GetUser(ctx, rt.UserID)
	if err != nil {
		return Session{}, err
	}
	return s.issueSession(ctx, user)
}

// Logout revokes a single refresh token (best-effort).
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	rt, err := s.store.GetRefreshToken(ctx, auth.HashToken(refreshToken))
	if err != nil {
		return nil
	}
	return s.store.RevokeRefreshToken(ctx, rt.ID, time.Now())
}

// --- Profile -----------------------------------------------------------------

// Profile returns the account plus its linked login methods (sans secrets).
type Profile struct {
	User       store.User       `json:"user"`
	Identities []ProfileIdentity `json:"identities"`
}

// ProfileIdentity is the safe projection of a login identity.
type ProfileIdentity struct {
	Type      store.IdentityType `json:"type"`
	Principal string             `json:"principal"`
	Verified  bool               `json:"verified"`
}

func (s *Service) Profile(ctx context.Context, userID string) (Profile, error) {
	user, err := s.store.GetUser(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	ids, err := s.store.ListIdentitiesByUser(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	out := Profile{User: user}
	for _, i := range ids {
		out.Identities = append(out.Identities, ProfileIdentity{
			Type:      i.Type,
			Principal: maskPrincipal(i.Type, i.Principal),
			Verified:  i.VerifiedAt != nil,
		})
	}
	return out, nil
}

// --- internals ---------------------------------------------------------------

func (s *Service) findOrCreateByIdentity(ctx context.Context, t store.IdentityType, principal, secretHash, display string) (store.User, error) {
	if id, err := s.store.GetIdentity(ctx, t, principal); err == nil {
		return s.store.GetUser(ctx, id.UserID)
	}
	return s.createUserWithIdentity(ctx, t, principal, secretHash, display)
}

func (s *Service) createUserWithIdentity(ctx context.Context, t store.IdentityType, principal, secretHash, display string) (store.User, error) {
	now := time.Now().UTC()
	user := store.User{
		ID:          platform.NewID("cu"),
		DisplayName: display,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.store.CreateUser(ctx, user); err != nil {
		return store.User{}, err
	}
	verified := now
	if err := s.store.CreateIdentity(ctx, store.Identity{
		ID:         platform.NewID("ci"),
		UserID:     user.ID,
		Type:       t,
		Principal:  principal,
		SecretHash: secretHash,
		VerifiedAt: &verified,
		CreatedAt:  now,
	}); err != nil {
		return store.User{}, err
	}
	return user, nil
}

func (s *Service) issueSession(ctx context.Context, user store.User) (Session, error) {
	access, exp, err := s.tokens.SignAccess(user.ID, user.DisplayName)
	if err != nil {
		return Session{}, err
	}
	raw, hash, err := auth.NewOpaqueToken()
	if err != nil {
		return Session{}, err
	}
	now := time.Now().UTC()
	if err := s.store.CreateRefreshToken(ctx, store.RefreshToken{
		ID:        platform.NewID("rt"),
		TokenHash: hash,
		UserID:    user.ID,
		ExpiresAt: now.Add(s.tokens.RefreshTTL()),
		CreatedAt: now,
	}); err != nil {
		return Session{}, err
	}
	return Session{User: user, AccessToken: access, AccessExpires: exp, RefreshToken: raw}, nil
}

func normalizeEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }

func emailLocalPart(e string) string {
	if at := strings.IndexByte(e, '@'); at > 0 {
		return e[:at]
	}
	return e
}

func maskPhone(p string) string {
	if len(p) >= 7 {
		return p[:3] + "****" + p[len(p)-4:]
	}
	return p
}

func maskPrincipal(t store.IdentityType, principal string) string {
	switch t {
	case store.IdentityPhone:
		return maskPhone(principal)
	case store.IdentityEmail:
		if at := strings.IndexByte(principal, '@'); at > 1 {
			return principal[:1] + "***" + principal[at:]
		}
	}
	return principal
}
