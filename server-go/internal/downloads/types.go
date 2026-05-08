package downloads

type SourceType string

const (
	SourceBT     SourceType = "BT"
	SourceHTTP   SourceType = "HTTP"
	SourceMagnet SourceType = "磁力"
	SourceRSS    SourceType = "订阅"
)

type TaskStatus string

const (
	StatusRunning   TaskStatus = "下载中"
	StatusPaused    TaskStatus = "暂停"
	StatusCompleted TaskStatus = "已完成"
)

type ArchiveRule struct {
	Category       string   `json:"category"`
	TargetPath     string   `json:"targetPath"`
	Tags           []string `json:"tags"`
	IndexAfterMove bool     `json:"indexAfterMove"`
	ScrapeMetadata bool     `json:"scrapeMetadata"`
	VerifyChecksum bool     `json:"verifyChecksum"`
}

type SpeedProfile struct {
	Name          string `json:"name"`
	DownloadLimit string `json:"downloadLimit"`
	UploadLimit   string `json:"uploadLimit"`
	Note          string `json:"note"`
	Active        bool   `json:"active"`
}

type DownloadTask struct {
	ID          int         `json:"id"`
	Name        string      `json:"name"`
	Source      SourceType  `json:"source"`
	Link        string      `json:"link"`
	Category    string      `json:"category"`
	Size        string      `json:"size"`
	Progress    int         `json:"progress"`
	Speed       string      `json:"speed"`
	Status      TaskStatus  `json:"status"`
	Handling    string      `json:"handling"`
	Archived    bool        `json:"archived"`
	ArchiveRule ArchiveRule `json:"archiveRule"`
}

type TaskActionResult struct {
	Task     DownloadTask `json:"task"`
	Message  string       `json:"message"`
	FilePath string       `json:"filePath,omitempty"`
}

type CreateTaskRequest struct {
	Source   SourceType `json:"source"`
	Link     string     `json:"link"`
	Name     string     `json:"name"`
	Category string     `json:"category"`
}
