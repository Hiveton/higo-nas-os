package files

import "time"

type FileNode struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Path       string     `json:"path"`
	RealPath   string     `json:"-"`
	Type       string     `json:"type"`
	Space      string     `json:"space"`
	Size       string     `json:"size"`
	SizeBytes  int64      `json:"sizeBytes"`
	Modified   time.Time  `json:"modifiedAt"`
	Tags       []string   `json:"tags"`
	Permission string     `json:"permission"`
	Summary    string     `json:"aiSummary"`
	IsDir      bool       `json:"isDir"`
	Children   []FileNode `json:"children,omitempty"`
}

type FileRow struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Path       string   `json:"path"`
	Type       string   `json:"type"`
	Space      string   `json:"space"`
	Size       string   `json:"size"`
	Modified   string   `json:"modified"`
	Tags       []string `json:"tags"`
	Permission string   `json:"permission"`
	AISummary  string   `json:"aiSummary"`
}

type SearchQuery struct {
	Query string   `json:"q"`
	Space string   `json:"space"`
	Type  string   `json:"type"`
	Tags  []string `json:"tags"`
	Limit int      `json:"limit"`
}

type ShareLink struct {
	ID            string    `json:"id"`
	FileID        string    `json:"fileId"`
	URL           string    `json:"url"`
	Password      string    `json:"password,omitempty"`
	ExpiresAt     time.Time `json:"expiresAt"`
	DownloadLimit int       `json:"downloadLimit"`
	Audit         string    `json:"audit"`
	Revoked       bool      `json:"revoked"`
}

type TagMutation struct {
	FileID string   `json:"fileId"`
	Tags   []string `json:"tags"`
	Actor  string   `json:"actor"`
	Source string   `json:"source"`
	Audit  string   `json:"audit"`
}

type BatchOperation struct {
	Type        string            `json:"type"`
	FileIDs     []string          `json:"fileIds"`
	Destination string            `json:"destination"`
	Rename      map[string]string `json:"rename"`
	Actor       string            `json:"actor"`
}

type Preview struct {
	FileID    string `json:"fileId"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Supported bool   `json:"supported"`
	Kind      string `json:"kind"`
	Summary   string `json:"summary"`
	Text      string `json:"text,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type Task struct {
	ID                   string          `json:"id"`
	Type                 string          `json:"type"`
	Status               string          `json:"status"`
	Actor                string          `json:"actor"`
	FileIDs              []string        `json:"fileIds"`
	PlannedActions       []PlannedAction `json:"plannedActions"`
	RollbackPlan         RollbackPlan    `json:"rollbackPlan"`
	RequiresConfirmation bool            `json:"requiresConfirmation"`
	CreatedAt            time.Time       `json:"createdAt"`
}

type PlannedAction struct {
	FileID string `json:"fileId"`
	From   string `json:"from"`
	To     string `json:"to"`
	Action string `json:"action"`
}

type RollbackPlan struct {
	ID        string         `json:"id"`
	Steps     []RollbackStep `json:"steps"`
	CreatedAt time.Time      `json:"createdAt"`
}

type RollbackStep struct {
	FileID string `json:"fileId"`
	From   string `json:"from"`
	To     string `json:"to"`
	Action string `json:"action"`
}
