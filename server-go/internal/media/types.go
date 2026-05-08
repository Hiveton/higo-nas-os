package media

type DimensionKey string

const (
	DimensionTimeline DimensionKey = "timeline"
	DimensionPeople   DimensionKey = "people"
	DimensionPlaces   DimensionKey = "places"
	DimensionDevices  DimensionKey = "devices"
	DimensionAlbums   DimensionKey = "albums"
)

type MediaKind string

const (
	MediaKindPhoto MediaKind = "照片"
	MediaKindVideo MediaKind = "视频"
	MediaKindMusic MediaKind = "音乐"
)

type AlbumType string

const (
	AlbumTypeFamily AlbumType = "家庭相册"
	AlbumTypeShared AlbumType = "共享相册"
	AlbumTypeMemory AlbumType = "智能回忆"
)

type JobStatus string

const (
	JobStatusQueued  JobStatus = "排队中"
	JobStatusRunning JobStatus = "运行中"
	JobStatusReady   JobStatus = "已就绪"
)

type MediaItem struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Kind        MediaKind `json:"kind"`
	Timeline    string    `json:"timeline"`
	People      string    `json:"people"`
	Place       string    `json:"place"`
	Device      string    `json:"device"`
	Album       string    `json:"album"`
	Meta        string    `json:"meta"`
	Status      string    `json:"status"`
	Accent      string    `json:"accent"`
	HasSubtitle bool      `json:"hasSubtitle"`
	Transcoded  bool      `json:"transcoded"`
}

type Album struct {
	ID      int       `json:"id"`
	Name    string    `json:"name"`
	Type    AlbumType `json:"type"`
	Count   int       `json:"count"`
	Privacy string    `json:"privacy"`
}

type Person struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Cluster       string `json:"cluster"`
	Count         int    `json:"count"`
	RollbackUntil string `json:"rollbackUntil,omitempty"`
}

type MemoryRun struct {
	ID        string       `json:"id"`
	Status    JobStatus    `json:"status"`
	Message   string       `json:"message"`
	RunCount  int          `json:"runCount"`
	Dimension DimensionKey `json:"dimension"`
	Facet     string       `json:"facet"`
	Album     Album        `json:"album"`
	ItemIDs   []int        `json:"itemIds"`
}

type SubtitleJob struct {
	ID      string    `json:"id"`
	ItemID  int       `json:"itemId"`
	Title   string    `json:"title"`
	Status  JobStatus `json:"status"`
	Message string    `json:"message"`
}

type TranscodeJob struct {
	ID      string    `json:"id"`
	ItemID  int       `json:"itemId"`
	Title   string    `json:"title"`
	Profile string    `json:"profile"`
	Status  JobStatus `json:"status"`
	Message string    `json:"message"`
}

type ShareResult struct {
	ID            string    `json:"id"`
	AlbumID       int       `json:"albumId"`
	AlbumName     string    `json:"albumName"`
	Status        JobStatus `json:"status"`
	Message       string    `json:"message"`
	AuditNotice   string    `json:"auditNotice"`
	ExpiresInDays int       `json:"expiresInDays"`
}

type ItemFilter struct {
	Dimension DimensionKey `json:"dimension"`
	Facet     string       `json:"facet"`
}

type CreateAlbumRequest struct {
	Name    string    `json:"name"`
	Type    AlbumType `json:"type"`
	ItemIDs []int     `json:"itemIds"`
	Privacy string    `json:"privacy"`
}

type CreateMemoryRequest struct {
	Dimension DimensionKey `json:"dimension"`
	Facet     string       `json:"facet"`
}

type MergePeopleRequest struct {
	SourceNames []string `json:"sourceNames"`
	TargetName  string   `json:"targetName"`
}

type MergePeopleResult struct {
	ID      string    `json:"id"`
	Status  JobStatus `json:"status"`
	Message string    `json:"message"`
	Notice  string    `json:"notice"`
}

type CreateMediaJobRequest struct {
	ItemID  int    `json:"itemId"`
	Profile string `json:"profile"`
}

type CreateShareRequest struct {
	AlbumID       int `json:"albumId"`
	ExpiresInDays int `json:"expiresInDays"`
}
