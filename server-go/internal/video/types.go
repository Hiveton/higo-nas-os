package video

type LibraryType string

const (
	LibraryMovie  LibraryType = "movie"
	LibrarySeries LibraryType = "series"
	LibraryMixed  LibraryType = "mixed"
)

type JobStatus string

const (
	JobQueued  JobStatus = "queued"
	JobRunning JobStatus = "running"
	JobDone    JobStatus = "done"
	JobFailed  JobStatus = "failed"
)

type LibrarySettings struct {
	Libraries []Library `json:"libraries"`
	Status    string    `json:"status"`
	ItemCount int       `json:"itemCount"`
	LastScan  string    `json:"lastScan,omitempty"`
}

type Library struct {
	ID                string      `json:"id"`
	Name              string      `json:"name"`
	Type              LibraryType `json:"type"`
	Paths             []string    `json:"paths"`
	MetadataLanguage  string      `json:"metadataLanguage"`
	AllowAdultContent bool        `json:"allowAdultContent"`
	AutoSubtitles     bool        `json:"autoSubtitles"`
	SubtitleLanguage  string      `json:"subtitleLanguage"`
	Count             int         `json:"count"`
	Status            string      `json:"status"`
}

type Item struct {
	ID                string       `json:"id"`
	LibraryID         string       `json:"libraryId"`
	LibraryName       string       `json:"libraryName"`
	Title             string       `json:"title"`
	OriginalTitle     string       `json:"originalTitle,omitempty"`
	SeriesTitle       string       `json:"seriesTitle,omitempty"`
	EpisodeTitle      string       `json:"episodeTitle,omitempty"`
	Kind              string       `json:"kind"`
	Year              string       `json:"year,omitempty"`
	Season            int          `json:"season,omitempty"`
	Episode           int          `json:"episode,omitempty"`
	Container         string       `json:"container"`
	Codec             string       `json:"codec"`
	Resolution        string       `json:"resolution"`
	DurationSeconds   int          `json:"durationSeconds,omitempty"`
	SizeBytes         int64        `json:"sizeBytes"`
	Size              string       `json:"size"`
	ModifiedAt        string       `json:"modifiedAt"`
	DiscoveredAt      string       `json:"discoveredAt"`
	FileName          string       `json:"fileName"`
	Path              string       `json:"path,omitempty"`
	PosterURL         string       `json:"posterUrl"`
	PosterPath        string       `json:"-"`
	PosterRemoteURL   string       `json:"posterRemoteUrl,omitempty"`
	BackdropURL       string       `json:"backdropUrl,omitempty"`
	BackdropRemoteURL string       `json:"backdropRemoteUrl,omitempty"`
	StreamURL         string       `json:"streamUrl"`
	SubtitleURL       string       `json:"subtitleUrl,omitempty"`
	Overview          string       `json:"overview"`
	AIOverview        string       `json:"aiOverview,omitempty"`
	AITranscript      string       `json:"aiTranscript,omitempty"`
	Tagline           string       `json:"tagline,omitempty"`
	ContentRating     string       `json:"contentRating,omitempty"`
	ReleaseDate       string       `json:"releaseDate,omitempty"`
	Genres            []string     `json:"genres"`
	Tags              []string     `json:"tags,omitempty"`
	Directors         []string     `json:"directors,omitempty"`
	Writers           []string     `json:"writers,omitempty"`
	Actors            []string     `json:"actors,omitempty"`
	Studios           []string     `json:"studios,omitempty"`
	Countries         []string     `json:"countries,omitempty"`
	VideoTracks       []MediaTrack `json:"videoTracks,omitempty"`
	AudioTracks       []MediaTrack `json:"audioTracks,omitempty"`
	SubtitleTracks    []MediaTrack `json:"subtitleTracks,omitempty"`
	Rating            string       `json:"rating"`
	MetadataSource    string       `json:"metadataSource,omitempty"`
	ProviderID        string       `json:"providerId,omitempty"`
	ScrapedAt         string       `json:"scrapedAt,omitempty"`
	Progress          int          `json:"progress"`
	Status            string       `json:"status"`
}

type MediaTrack struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Language string `json:"language,omitempty"`
	Codec    string `json:"codec,omitempty"`
	Channels string `json:"channels,omitempty"`
	Default  bool   `json:"default,omitempty"`
}

type ScanResult struct {
	ID        string `json:"id"`
	State     string `json:"state"`
	Message   string `json:"message"`
	ItemCount int    `json:"itemCount"`
	ScannedAt string `json:"scannedAt"`
}

type Task struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	ItemID    string    `json:"itemId,omitempty"`
	Title     string    `json:"title"`
	Status    JobStatus `json:"status"`
	Message   string    `json:"message"`
	Progress  int       `json:"progress"`
	Profile   string    `json:"profile,omitempty"`
	CreatedAt string    `json:"createdAt"`
}

type LiveSource struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	UserAgent    string `json:"userAgent,omitempty"`
	StreamLimit  int    `json:"streamLimit,omitempty"`
	ChannelCount int    `json:"channelCount"`
	Status       string `json:"status"`
}

type LiveChannel struct {
	ID        string `json:"id"`
	SourceID  string `json:"sourceId"`
	GuideID   string `json:"guideId,omitempty"`
	GuideName string `json:"guideName,omitempty"`
	Name      string `json:"name"`
	Group     string `json:"group"`
	Logo      string `json:"logo,omitempty"`
	URL       string `json:"url"`
}

type LiveGuideSource struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	UserAgent    string `json:"userAgent,omitempty"`
	ProgramCount int    `json:"programCount"`
	Status       string `json:"status"`
	LastRefresh  string `json:"lastRefresh,omitempty"`
}

type LiveProgram struct {
	ID          string   `json:"id"`
	GuideID     string   `json:"guideId"`
	ChannelID   string   `json:"channelId"`
	ChannelName string   `json:"channelName"`
	Title       string   `json:"title"`
	Overview    string   `json:"overview,omitempty"`
	Categories  []string `json:"categories,omitempty"`
	StartAt     string   `json:"startAt"`
	EndAt       string   `json:"endAt"`
	Duration    int      `json:"durationSeconds"`
}

type LiveProgramQuery struct {
	SourceID  string
	ChannelID string
	From      string
	To        string
}

type DVRSettings struct {
	RecordingPath       string `json:"recordingPath"`
	MovieRecordingPath  string `json:"movieRecordingPath,omitempty"`
	SeriesRecordingPath string `json:"seriesRecordingPath,omitempty"`
	PrePaddingSeconds   int    `json:"prePaddingSeconds"`
	PostPaddingSeconds  int    `json:"postPaddingSeconds"`
	MaxConcurrentRecord int    `json:"maxConcurrentRecord"`
	SaveNFO             bool   `json:"saveNfo"`
	SaveImages          bool   `json:"saveImages"`
	PostProcessCommand  string `json:"postProcessCommand,omitempty"`
}

type RecordingStatus string

const (
	RecordingScheduled RecordingStatus = "scheduled"
	RecordingCancelled RecordingStatus = "cancelled"
	RecordingRunning   RecordingStatus = "recording"
	RecordingCompleted RecordingStatus = "completed"
	RecordingFailed    RecordingStatus = "failed"
)

type RecordingTimer struct {
	ID                 string          `json:"id"`
	ProgramID          string          `json:"programId,omitempty"`
	ChannelID          string          `json:"channelId"`
	ChannelName        string          `json:"channelName"`
	Name               string          `json:"name"`
	Overview           string          `json:"overview,omitempty"`
	StartAt            string          `json:"startAt"`
	EndAt              string          `json:"endAt"`
	PrePaddingSeconds  int             `json:"prePaddingSeconds"`
	PostPaddingSeconds int             `json:"postPaddingSeconds"`
	Priority           int             `json:"priority"`
	Status             RecordingStatus `json:"status"`
	TargetPath         string          `json:"targetPath,omitempty"`
	CreatedAt          string          `json:"createdAt"`
}

type RecordingItem struct {
	ID        string          `json:"id"`
	TimerID   string          `json:"timerId"`
	ProgramID string          `json:"programId,omitempty"`
	Title     string          `json:"title"`
	ChannelID string          `json:"channelId"`
	Status    RecordingStatus `json:"status"`
	Path      string          `json:"path,omitempty"`
	StartedAt string          `json:"startedAt,omitempty"`
	EndedAt   string          `json:"endedAt,omitempty"`
	Message   string          `json:"message,omitempty"`
}

type LibraryUpdateRequest struct {
	Libraries []Library `json:"libraries"`
}

type CreateLibraryRequest struct {
	Name              string      `json:"name"`
	Type              LibraryType `json:"type"`
	Paths             []string    `json:"paths"`
	MetadataLanguage  string      `json:"metadataLanguage"`
	AllowAdultContent bool        `json:"allowAdultContent"`
	AutoSubtitles     bool        `json:"autoSubtitles"`
	SubtitleLanguage  string      `json:"subtitleLanguage"`
}

type DeleteLibraryResult struct {
	ID           string          `json:"id"`
	RemovedItems int             `json:"removedItems"`
	RemovedTasks int             `json:"removedTasks"`
	Settings     LibrarySettings `json:"settings"`
}

type CreateTaskRequest struct {
	ItemID  string `json:"itemId"`
	Profile string `json:"profile"`
}

type CreateLiveSourceRequest struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	UserAgent   string `json:"userAgent,omitempty"`
	StreamLimit int    `json:"streamLimit,omitempty"`
}

type CreateGuideSourceRequest struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	UserAgent string `json:"userAgent,omitempty"`
}

type CreateRecordingTimerRequest struct {
	ProgramID  string `json:"programId,omitempty"`
	ChannelID  string `json:"channelId,omitempty"`
	Name       string `json:"name,omitempty"`
	Overview   string `json:"overview,omitempty"`
	StartAt    string `json:"startAt,omitempty"`
	EndAt      string `json:"endAt,omitempty"`
	Priority   int    `json:"priority,omitempty"`
	TargetPath string `json:"targetPath,omitempty"`
}
