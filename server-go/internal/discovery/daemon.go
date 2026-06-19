// Package discovery runs a lightweight UDP responder so the HiGoOS desktop
// assistant (tools/higo-assistant) can find a fresh NAS on the LAN and read its
// fingerprint without prior configuration.
//
// Protocol HIGOOS/1 (see docs/desktop-scanner.md §4): a client broadcasts a
// JSON "discover" datagram; the daemon replies, unicast, with an "announce"
// carrying the identity fingerprint and echoing the client's nonce. The daemon
// is READ-ONLY — it never accepts configuration over UDP; all writes go through
// the authenticated HTTP API.
package discovery

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"sync"
	"time"

	"higoos/server-go/internal/identity"
)

const (
	magic       = "HIGOOS/1"
	typeProbe   = "discover"
	typeAnswer  = "announce"
	maxDatagram = 2048
	// minReplyGap throttles replies to a single source to blunt UDP-amplification
	// abuse. The answer is barely larger than the probe, so risk is low regardless.
	minReplyGap = 200 * time.Millisecond
)

type probe struct {
	Magic     string `json:"magic"`
	Type      string `json:"type"`
	Nonce     string `json:"nonce"`
	ReplyPort int    `json:"replyPort"`
}

type announce struct {
	Magic string `json:"magic"`
	Type  string `json:"type"`
	Nonce string `json:"nonce"`
	identity.Identity
}

// Snapshotter supplies the current device fingerprint. *identity.Provider
// satisfies it.
type Snapshotter interface {
	Snapshot() identity.Identity
}

// Run listens for discovery probes until ctx is cancelled. It returns the
// listener error (or nil on clean shutdown). Intended to be launched in a
// goroutine.
func Run(ctx context.Context, addr string, snap Snapshotter, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	var lc net.ListenConfig
	conn, err := lc.ListenPacket(ctx, "udp4", addr)
	if err != nil {
		logger.Warn("discovery daemon disabled", slog.String("addr", addr), slog.Any("error", err))
		return err
	}
	logger.Info("discovery daemon listening", slog.String("addr", addr))

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	throttle := newThrottle()
	buf := make([]byte, maxDatagram)
	for {
		n, from, err := conn.ReadFrom(buf)
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				logger.Warn("discovery read error", slog.Any("error", err))
				return err
			}
		}
		p, ok := parseProbe(buf[:n])
		if !ok {
			continue
		}
		udpFrom, ok := from.(*net.UDPAddr)
		if !ok || !throttle.allow(udpFrom.IP.String()) {
			continue
		}
		payload, err := buildReply(p, snap)
		if err != nil {
			continue
		}
		dst := udpFrom
		if p.ReplyPort > 0 {
			dst = &net.UDPAddr{IP: udpFrom.IP, Port: p.ReplyPort, Zone: udpFrom.Zone}
		}
		if _, err := conn.WriteTo(payload, dst); err != nil {
			logger.Debug("discovery reply failed", slog.Any("error", err))
		}
	}
}

// parseProbe decodes a datagram and reports whether it is a valid HIGOOS/1
// discovery probe.
func parseProbe(data []byte) (probe, bool) {
	var p probe
	if json.Unmarshal(data, &p) != nil {
		return probe{}, false
	}
	if p.Magic != magic || p.Type != typeProbe {
		return probe{}, false
	}
	return p, true
}

// buildReply marshals an announce for a probe, echoing its nonce.
func buildReply(p probe, snap Snapshotter) ([]byte, error) {
	return json.Marshal(announce{Magic: magic, Type: typeAnswer, Nonce: p.Nonce, Identity: snap.Snapshot()})
}

// throttle enforces a minimum gap between replies to the same source IP.
type throttle struct {
	mu   sync.Mutex
	last map[string]time.Time
}

func newThrottle() *throttle {
	return &throttle{last: make(map[string]time.Time)}
}

func (t *throttle) allow(ip string) bool {
	now := time.Now()
	t.mu.Lock()
	defer t.mu.Unlock()
	if seen, ok := t.last[ip]; ok && now.Sub(seen) < minReplyGap {
		return false
	}
	t.last[ip] = now
	// Opportunistically prune stale entries so the map can't grow unbounded.
	if len(t.last) > 1024 {
		for k, v := range t.last {
			if now.Sub(v) > time.Minute {
				delete(t.last, k)
			}
		}
	}
	return true
}
