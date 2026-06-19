// Package aianalysis is the global background AI analysis engine. It analyses
// the photo (media), file and video domains, persisting per-item analysis state
// to a JSON ledger so progress survives restarts and power loss (resumable from
// the last interrupted item). Depth is governed by a configurable level
// (off/basic/standard/deep) read from settings; the engine runs in-process in
// higo-api (the worker cannot write JSON state) and drives work through the
// shared persistent task runtime.
package aianalysis

import (
	"strings"
	"time"
)

// DomainKind identifies which media domain a record belongs to.
type DomainKind string

const (
	DomainMedia DomainKind = "media"
	DomainFile  DomainKind = "file"
	DomainVideo DomainKind = "video"
)

// domainOrder is the stable presentation order for status/records surfaces.
var domainOrder = []DomainKind{DomainMedia, DomainFile, DomainVideo}

// ItemState is the analysis lifecycle state of one item.
type ItemState string

const (
	StatePending   ItemState = "pending"
	StateAnalyzing ItemState = "analyzing"
	StateDone      ItemState = "done"
	StateFailed    ItemState = "failed"
	StateSkipped   ItemState = "skipped"
)

// Level gates how deep analysis goes. Higher levels are cumulative.
type Level string

const (
	LevelOff      Level = "off"
	LevelBasic    Level = "basic"
	LevelStandard Level = "standard"
	LevelDeep     Level = "deep"
)

// ParseLevel maps a settings string to a Level, defaulting to basic.
func ParseLevel(s string) Level {
	switch Level(strings.ToLower(strings.TrimSpace(s))) {
	case LevelOff:
		return LevelOff
	case LevelStandard:
		return LevelStandard
	case LevelDeep:
		return LevelDeep
	default:
		return LevelBasic
	}
}

func (l Level) rank() int {
	switch l {
	case LevelDeep:
		return 3
	case LevelStandard:
		return 2
	case LevelBasic:
		return 1
	default:
		return 0
	}
}

// atLeast reports whether l is at least as deep as other.
func (l Level) atLeast(other Level) bool { return l.rank() >= other.rank() }

// AnalyzerResult is the merged output of the per-level analyzers for one item.
type AnalyzerResult struct {
	Summary    string         `json:"summary,omitempty"`
	Caption    string         `json:"caption,omitempty"`
	Tags       []string       `json:"tags,omitempty"`
	People     []string       `json:"people,omitempty"`
	Place      string         `json:"place,omitempty"`
	Device     string         `json:"device,omitempty"`
	Transcript string         `json:"transcript,omitempty"`
	TechMeta   map[string]any `json:"techMeta,omitempty"`
	Embedded   bool           `json:"embedded,omitempty"`
}

// Record is the persisted analysis state of one item. It doubles as the
// item-level checkpoint: on restart, records left mid-flight (StateAnalyzing)
// are reset to StatePending and re-driven, and a changed ContentSig forces a
// re-analysis.
type Record struct {
	Key        string         `json:"key"`
	Domain     DomainKind     `json:"domain"`
	Title      string         `json:"title"`
	SourcePath string         `json:"sourcePath,omitempty"`
	Kind       string         `json:"kind,omitempty"`
	ContentSig string         `json:"contentSig"`
	State      ItemState      `json:"state"`
	Level      Level          `json:"level"`
	Progress   int            `json:"progress"`
	Attempts   int            `json:"attempts"`
	Error      string         `json:"error,omitempty"`
	Result     AnalyzerResult `json:"result"`
	AnalyzedAt *time.Time     `json:"analyzedAt,omitempty"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

// ItemRef is a lightweight pointer to a domain item produced by a DomainSource
// during enumeration. Sig is a cheap content signature (mtime+size) used for
// change detection.
type ItemRef struct {
	Key        string
	Domain     DomainKind
	Title      string
	SourcePath string
	Kind       string
	Sig        string
}

// DomainStats summarises one domain's ledger for the status surface.
type DomainStats struct {
	Domain    DomainKind `json:"domain"`
	Total     int        `json:"total"`
	Pending   int        `json:"pending"`
	Analyzing int        `json:"analyzing"`
	Done      int        `json:"done"`
	Failed    int        `json:"failed"`
	Skipped   int        `json:"skipped"`
	Percent   int        `json:"percent"`
}

// EngineStatus is the aggregate status payload returned by the API.
type EngineStatus struct {
	Level           Level         `json:"level"`
	Paused          bool          `json:"paused"`
	TotalPercent    int           `json:"totalPercent"`
	Domains         []DomainStats `json:"domains"`
	HasChat         bool          `json:"hasChat"`
	HasVision       bool          `json:"hasVision"`
	HasEmbedding    bool          `json:"hasEmbedding"`
	HasASR          bool          `json:"hasAsr"`
	IndexEnabled    bool          `json:"indexEnabled"`
	FFmpegAvailable bool          `json:"ffmpegAvailable"`
	UpdatedAt       time.Time     `json:"updatedAt"`
}
