package music

type LibrarySettings struct {
	Paths      []string `json:"paths"`
	AutoScan   bool     `json:"autoScan"`
	LastScanAt string   `json:"lastScanAt,omitempty"`
	TrackCount int      `json:"trackCount"`
	Status     string   `json:"status"`
}

type Track struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Artist          string `json:"artist"`
	Album           string `json:"album"`
	TrackNumber     int    `json:"trackNumber,omitempty"`
	Year            string `json:"year,omitempty"`
	Codec           string `json:"codec"`
	Format          string `json:"format"`
	DurationSeconds int    `json:"durationSeconds,omitempty"`
	SizeBytes       int64  `json:"sizeBytes"`
	Size            string `json:"size"`
	ModifiedAt      string `json:"modifiedAt"`
	DiscoveredAt    string `json:"discoveredAt"`
	FileName        string `json:"fileName"`
	Path            string `json:"path,omitempty"`
	StreamURL       string `json:"streamUrl"`
	CoverURL        string `json:"coverUrl,omitempty"`
	LyricsURL       string `json:"lyricsUrl,omitempty"`
	Lyrics          string `json:"lyrics,omitempty"`
	Status          string `json:"status"`
}

type Album struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Artist   string `json:"artist"`
	Count    int    `json:"count"`
	CoverURL string `json:"coverUrl,omitempty"`
}

type ScanResult struct {
	ID         string `json:"id"`
	State      string `json:"state"`
	Message    string `json:"message"`
	TrackCount int    `json:"trackCount"`
	ScannedAt  string `json:"scannedAt"`
}

type LibraryUpdateRequest struct {
	Paths    []string `json:"paths"`
	AutoScan *bool    `json:"autoScan,omitempty"`
}
