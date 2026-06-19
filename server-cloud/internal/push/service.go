// Package push registers APNs device tokens and (eventually) sends notifications
// to a cloud account's devices. The send path is stubbed for now — token-based
// APNs (.p8) wiring is an M5 follow-up — but token registration is real so the
// App can call it today.
package push

import (
	"context"
	"log/slog"
	"time"

	"higoos/server-cloud/internal/platform"
	"higoos/server-cloud/internal/store"
)

// Service manages push tokens and sends notifications. When sender is nil it logs
// the intent instead of dialling APNs (dev), so the control flow is observable
// without credentials.
type Service struct {
	store  store.Store
	logger *slog.Logger
	sender Sender
}

func NewService(s store.Store, logger *slog.Logger) *Service {
	return &Service{store: s, logger: logger}
}

// WithSender attaches a real notification sender (e.g. APNsSender).
func (s *Service) WithSender(sender Sender) *Service {
	s.sender = sender
	return s
}

// Register upserts an APNs token for a cloud account.
func (s *Service) Register(ctx context.Context, userID, token, platformName string) error {
	if platformName == "" {
		platformName = "ios"
	}
	return s.store.UpsertPushToken(ctx, store.PushToken{
		ID:        platform.NewID("pt"),
		UserID:    userID,
		Token:     token,
		Platform:  platformName,
		CreatedAt: time.Now().UTC(),
	})
}

// Notify sends a notification to every device registered for userID. The actual
// APNs HTTP/2 send is an M5 TODO; for now it logs the intent so the control flow
// is observable end-to-end.
func (s *Service) Notify(ctx context.Context, userID, title, body string) error {
	tokens, err := s.store.ListPushTokensByUser(ctx, userID)
	if err != nil {
		return err
	}
	if s.sender == nil {
		if s.logger != nil {
			s.logger.Info("push notify (stub — no APNs configured)",
				slog.String("userId", userID),
				slog.Int("tokens", len(tokens)),
				slog.String("title", title))
		}
		return nil
	}
	for _, t := range tokens {
		if err := s.sender.Send(ctx, t.Token, title, body); err != nil && s.logger != nil {
			s.logger.Warn("apns send failed", slog.String("userId", userID), slog.Any("error", err))
		}
	}
	return nil
}
