// Package foldersync is the device/folder synchronization domain: configurable
// sync pairs that perform real incremental file synchronization (one-way mirror
// or additive two-way merge with conflict detection) through the central task
// runtime, with selective-path filters and an audit trail. The Go package is
// named foldersync to avoid colliding with the standard library's sync package;
// its REST surface lives under /api/v1/sync.
package foldersync

import "time"

// Direction is how a pair synchronizes.
type Direction string

const (
	// DirectionMirror is a one-way incremental mirror: source overwrites target.
	DirectionMirror Direction = "mirror"
	// DirectionTwoWay is an additive two-way merge: each side gets the other's
	// newer files; same-path different-content with equal mtime is a conflict.
	DirectionTwoWay Direction = "two-way"
)

// ConflictPolicy decides how two-way conflicts are auto-resolved.
type ConflictPolicy string

const (
	ConflictNewer  ConflictPolicy = "newer"  // keep the file with the newer mtime
	ConflictSource ConflictPolicy = "source" // source side wins
	ConflictTarget ConflictPolicy = "target" // target side wins
	ConflictManual ConflictPolicy = "manual" // leave both, record for the user
)

// SyncPair is one configured synchronization relationship.
type SyncPair struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Source         string         `json:"source"`
	Target         string         `json:"target"`
	Direction      Direction      `json:"direction"`
	ConflictPolicy ConflictPolicy `json:"conflictPolicy"`
	// Includes are optional glob patterns (slash paths, relative to source). When
	// non-empty only matching files sync (selective / on-demand sync).
	Includes []string `json:"includes,omitempty"`
	// BandwidthLimit is an advisory cap shown in the UI, e.g. "10 MB/s". Real
	// throughput throttling is not yet enforced (documented).
	BandwidthLimit string `json:"bandwidthLimit,omitempty"`
	Enabled        bool   `json:"enabled"`
	IntervalHours  int    `json:"intervalHours,omitempty"` // 0 = manual only

	// Live/last-run fields.
	State     string     `json:"state"` // 空闲 / 同步中 / 已完成 / 失败 / 有冲突
	Progress  int        `json:"progress"`
	LastRun   *time.Time `json:"lastRun,omitempty"`
	LastStats *RunStats  `json:"lastStats,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	CreatedBy string     `json:"createdBy,omitempty"`
}

// RunStats summarises one sync run.
type RunStats struct {
	Copied    int   `json:"copied"`
	Skipped   int   `json:"skipped"`
	Bytes     int64 `json:"bytes"`
	Conflicts int   `json:"conflicts"`
}

// Conflict is a two-way collision left for the user (or auto-resolution record).
type Conflict struct {
	ID         string    `json:"id"`
	PairID     string    `json:"pairId"`
	RelPath    string    `json:"relPath"`
	Detail     string    `json:"detail"`
	Resolved   bool      `json:"resolved"`
	Resolution string    `json:"resolution,omitempty"`
	DetectedAt time.Time `json:"detectedAt"`
}

// AuditEntry is the append-only run/config log (newest first).
type AuditEntry struct {
	ID     string    `json:"id"`
	Event  string    `json:"event"`
	Actor  string    `json:"actor,omitempty"`
	PairID string    `json:"pairId,omitempty"`
	Result string    `json:"result"` // ok / blocked / conflict
	Time   time.Time `json:"time"`
}

// CreatePairRequest is the body for creating a sync pair.
type CreatePairRequest struct {
	Name           string         `json:"name"`
	Source         string         `json:"source"`
	Target         string         `json:"target"`
	Direction      Direction      `json:"direction"`
	ConflictPolicy ConflictPolicy `json:"conflictPolicy"`
	Includes       []string       `json:"includes,omitempty"`
	BandwidthLimit string         `json:"bandwidthLimit,omitempty"`
	IntervalHours  int            `json:"intervalHours,omitempty"`
	Actor          string         `json:"actor,omitempty"`
}

// UpdatePairRequest patches a pair's config (pointer fields = leave unchanged).
type UpdatePairRequest struct {
	Name           *string         `json:"name,omitempty"`
	Direction      *Direction      `json:"direction,omitempty"`
	ConflictPolicy *ConflictPolicy `json:"conflictPolicy,omitempty"`
	Includes       *[]string       `json:"includes,omitempty"`
	BandwidthLimit *string         `json:"bandwidthLimit,omitempty"`
	Enabled        *bool           `json:"enabled,omitempty"`
	IntervalHours  *int            `json:"intervalHours,omitempty"`
	Actor          string          `json:"actor,omitempty"`
}

func clonePair(p SyncPair) SyncPair {
	if p.Includes != nil {
		p.Includes = append([]string(nil), p.Includes...)
	}
	if p.LastRun != nil {
		t := *p.LastRun
		p.LastRun = &t
	}
	if p.LastStats != nil {
		s := *p.LastStats
		p.LastStats = &s
	}
	return p
}

func clonePairs(in []SyncPair) []SyncPair {
	out := make([]SyncPair, 0, len(in))
	for _, p := range in {
		out = append(out, clonePair(p))
	}
	return out
}

func validDirection(d Direction) bool {
	return d == DirectionMirror || d == DirectionTwoWay
}

func validConflictPolicy(p ConflictPolicy) bool {
	switch p {
	case ConflictNewer, ConflictSource, ConflictTarget, ConflictManual:
		return true
	}
	return false
}
