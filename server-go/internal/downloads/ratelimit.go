package downloads

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// rateLimiter is a minimal dependency-free token bucket used to cap a single
// download's throughput. bytesPerSec <= 0 means unlimited (wait is a no-op).
// The bucket holds at most one second of tokens, so a slow start can't bank a
// large burst.
type rateLimiter struct {
	bytesPerSec int64
	allowance   float64
	last        time.Time
}

func newRateLimiter(bytesPerSec int64) *rateLimiter {
	return &rateLimiter{bytesPerSec: bytesPerSec, allowance: float64(bytesPerSec), last: time.Now()}
}

// setRate updates the enforced rate live (e.g. after a profile switch).
func (r *rateLimiter) setRate(bytesPerSec int64) {
	if bytesPerSec == r.bytesPerSec {
		return
	}
	r.bytesPerSec = bytesPerSec
	if r.allowance > float64(bytesPerSec) {
		r.allowance = float64(bytesPerSec)
	}
}

// wait blocks just long enough that sending n bytes does not exceed the limit.
func (r *rateLimiter) wait(n int) {
	if r.bytesPerSec <= 0 {
		return
	}
	now := time.Now()
	r.allowance += now.Sub(r.last).Seconds() * float64(r.bytesPerSec)
	r.last = now
	if cap := float64(r.bytesPerSec); r.allowance > cap {
		r.allowance = cap
	}
	r.allowance -= float64(n)
	if r.allowance < 0 {
		sleep := time.Duration(-r.allowance / float64(r.bytesPerSec) * float64(time.Second))
		time.Sleep(sleep)
		r.allowance = 0
		r.last = time.Now()
	}
}

var limitPattern = regexp.MustCompile(`(?i)([\d.]+)\s*(kb|mb|gb|k|m|g|b)?`)

// parseSpeedLimit converts a human limit string ("18 MB/s", "500KB", "不限速")
// into bytes/sec. Returns 0 for unlimited / unparseable values.
func parseSpeedLimit(s string) int64 {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || strings.Contains(s, "不限") || strings.Contains(s, "unlimited") || s == "0" {
		return 0
	}
	m := limitPattern.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	val, err := strconv.ParseFloat(m[1], 64)
	if err != nil || val <= 0 {
		return 0
	}
	switch m[2] {
	case "gb", "g":
		return int64(val * 1024 * 1024 * 1024)
	case "mb", "m":
		return int64(val * 1024 * 1024)
	case "kb", "k":
		return int64(val * 1024)
	default: // "b" or no unit — assume the string was MB/s if it had no unit but a big-ish slash form; here treat bare number as MB for NAS UX
		if m[2] == "b" {
			return int64(val)
		}
		return int64(val * 1024 * 1024)
	}
}

// formatLimit renders a bytes/sec value back to a compact display string.
func formatLimit(bytesPerSec int64) string {
	if bytesPerSec <= 0 {
		return "不限速"
	}
	switch {
	case bytesPerSec >= 1024*1024*1024:
		return strconv.FormatFloat(float64(bytesPerSec)/(1024*1024*1024), 'f', 1, 64) + " GB/s"
	case bytesPerSec >= 1024*1024:
		return strconv.FormatFloat(float64(bytesPerSec)/(1024*1024), 'f', 1, 64) + " MB/s"
	case bytesPerSec >= 1024:
		return strconv.FormatFloat(float64(bytesPerSec)/1024, 'f', 0, 64) + " KB/s"
	default:
		return strconv.FormatInt(bytesPerSec, 10) + " B/s"
	}
}
