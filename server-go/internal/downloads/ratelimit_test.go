package downloads

import (
	"testing"
	"time"
)

func TestParseSpeedLimit(t *testing.T) {
	cases := map[string]int64{
		"不限速":      0,
		"":         0,
		"18 MB/s":  18 * 1024 * 1024,
		"500KB":    500 * 1024,
		"500 kb/s": 500 * 1024,
		"1 GB/s":   1024 * 1024 * 1024,
		"2":        2 * 1024 * 1024, // bare number → MB (NAS UX default)
		"1024 b":   1024,
	}
	for in, want := range cases {
		if got := parseSpeedLimit(in); got != want {
			t.Errorf("parseSpeedLimit(%q) = %d, want %d", in, got, want)
		}
	}
}

// TestRateLimiterThrottles verifies that pushing N bytes through a limiter set
// to L bytes/sec takes at least ~N/L seconds.
func TestRateLimiterThrottles(t *testing.T) {
	const limit = 100 * 1024 // 100 KB/s
	lim := newRateLimiter(limit)
	// Drain the initial 1s burst allowance first so timing reflects the cap.
	lim.wait(limit)

	start := time.Now()
	const chunk = 10 * 1024
	const chunks = 20 // 200 KB → expect ~2s at 100 KB/s
	for i := 0; i < chunks; i++ {
		lim.wait(chunk)
	}
	elapsed := time.Since(start)
	wantAtLeast := 1500 * time.Millisecond // 200KB/100KBps = 2s; allow slack
	if elapsed < wantAtLeast {
		t.Errorf("expected throttle >= %v for 200KB at 100KB/s, got %v", wantAtLeast, elapsed)
	}
}

func TestRateLimiterUnlimitedIsNoop(t *testing.T) {
	lim := newRateLimiter(0)
	start := time.Now()
	for i := 0; i < 100; i++ {
		lim.wait(1024 * 1024)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Errorf("unlimited limiter should not block, took %v", elapsed)
	}
}

func TestUpdateQueueConfigPumpsAndPersists(t *testing.T) {
	s := NewService()
	if got := s.QueueConfig(nil).MaxConcurrentDownloads; got != defaultMaxConcurrent {
		t.Fatalf("default max concurrent = %d, want %d", got, defaultMaxConcurrent)
	}
	cfg, err := s.UpdateQueueConfig(nil, QueueConfig{MaxConcurrentDownloads: 7})
	if err != nil {
		t.Fatalf("UpdateQueueConfig: %v", err)
	}
	if cfg.MaxConcurrentDownloads != 7 {
		t.Fatalf("updated max concurrent = %d, want 7", cfg.MaxConcurrentDownloads)
	}
	if _, err := s.UpdateQueueConfig(nil, QueueConfig{MaxConcurrentDownloads: -1}); err == nil {
		t.Fatal("expected error for negative max concurrent")
	}
}
