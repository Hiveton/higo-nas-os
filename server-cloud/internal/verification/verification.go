// Package verification issues and checks the one-time codes behind phone-SMS and
// email flows. The code store is an in-memory TTL cache (swap for Redis in
// production); the Sender is pluggable, defaulting to a dev stub that logs the
// code instead of dialling a real SMS/email gateway.
package verification

import (
	"crypto/rand"
	"log/slog"
	"math/big"
	"strings"
	"sync"
	"time"
)

// Channel distinguishes the delivery medium.
type Channel string

const (
	ChannelSMS   Channel = "sms"
	ChannelEmail Channel = "email"
)

// Sender delivers a code to a target (phone number or email address).
type Sender interface {
	Send(channel Channel, target, code string) error
}

// LogSender is the dev stub: it logs the code so a developer can read it from
// the server output. It never contacts an external gateway.
type LogSender struct{ Logger *slog.Logger }

func (s LogSender) Send(channel Channel, target, code string) error {
	if s.Logger != nil {
		s.Logger.Info("verification code issued (dev stub)",
			slog.String("channel", string(channel)),
			slog.String("target", target),
			slog.String("code", code))
	}
	return nil
}

type entry struct {
	code     string
	expires  time.Time
	attempts int
}

// Service holds active codes and enforces TTL + a small attempt cap.
type Service struct {
	mu     sync.Mutex
	codes  map[string]entry
	ttl    time.Duration
	sender Sender
	debug  bool // echo codes back to the API caller (dev only)
}

// NewService builds a verification service.
func NewService(ttl time.Duration, sender Sender, debug bool) *Service {
	return &Service{codes: map[string]entry{}, ttl: ttl, sender: sender, debug: debug}
}

func key(channel Channel, target string) string {
	return string(channel) + "|" + strings.TrimSpace(strings.ToLower(target))
}

// Issue generates a 6-digit code, stores it under the TTL, and hands it to the
// Sender. When debug is on it also returns the code so dev clients can complete
// the flow without reading logs; otherwise it returns "".
func (s *Service) Issue(channel Channel, target string) (debugCode string, err error) {
	code := sixDigits()
	s.mu.Lock()
	s.codes[key(channel, target)] = entry{code: code, expires: time.Now().Add(s.ttl)}
	s.mu.Unlock()
	if err := s.sender.Send(channel, target, code); err != nil {
		return "", err
	}
	if s.debug {
		return code, nil
	}
	return "", nil
}

// Verify reports whether code matches the active, unexpired code for the target.
// A correct or exhausted entry is consumed so codes are single-use.
func (s *Service) Verify(channel Channel, target, code string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(channel, target)
	e, ok := s.codes[k]
	if !ok || time.Now().After(e.expires) {
		delete(s.codes, k)
		return false
	}
	e.attempts++
	if e.code == code {
		delete(s.codes, k)
		return true
	}
	if e.attempts >= 5 {
		delete(s.codes, k) // too many tries — burn it
	} else {
		s.codes[k] = e
	}
	return false
}

func sixDigits() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "000000"
	}
	s := n.String()
	for len(s) < 6 {
		s = "0" + s
	}
	return s
}
