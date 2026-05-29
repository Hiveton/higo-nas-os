package music

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

var audioExtensions = map[string]string{
	".mp3":  "MP3",
	".flac": "FLAC",
	".m4a":  "AAC/ALAC",
	".aac":  "AAC",
	".wav":  "WAV",
	".ogg":  "OGG",
	".opus": "Opus",
	".aiff": "AIFF",
	".alac": "ALAC",
}

type Service struct {
	mu        sync.RWMutex
	settings  LibrarySettings
	tracks    []Track
	statePath string
}

type snapshot struct {
	Settings LibrarySettings `json:"settings"`
	Tracks   []Track         `json:"tracks"`
}

type readSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type coverAsset struct {
	reader      readSeekCloser
	contentType string
	name        string
	modTime     time.Time
}

type nopReadSeekCloser struct {
	*bytes.Reader
}

func (nopReadSeekCloser) Close() error {
	return nil
}

func NewService() *Service {
	return &Service{
		settings: LibrarySettings{
			Paths:    []string{},
			AutoScan: true,
			Status:   "未设置媒体库",
		},
	}
}

func NewServiceWithStateDir(stateDir string) (*Service, error) {
	service := NewService()
	if stateDir == "" {
		return service, nil
	}
	service.statePath = filepath.Join(stateDir, "music.json")
	var persisted snapshot
	if err := state.LoadJSON(service.statePath, &persisted); err != nil {
		return nil, err
	}
	if len(persisted.Settings.Paths) > 0 || len(persisted.Tracks) > 0 {
		service.settings = persisted.Settings
		service.tracks = cloneTracks(persisted.Tracks)
		service.settings.TrackCount = len(service.tracks)
		if service.settings.Status == "" {
			service.settings.Status = "媒体库已加载"
		}
	}
	return service, nil
}

func (s *Service) Settings(ctx context.Context) (LibrarySettings, error) {
	if err := ctx.Err(); err != nil {
		return LibrarySettings{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	settings := s.settings
	settings.Paths = append([]string{}, settings.Paths...)
	settings.TrackCount = len(s.tracks)
	return settings, nil
}

func (s *Service) UpdateSettings(ctx context.Context, request LibraryUpdateRequest) (LibrarySettings, error) {
	if err := ctx.Err(); err != nil {
		return LibrarySettings{}, err
	}
	paths, err := normalizeLibraryPaths(request.Paths)
	if err != nil {
		return LibrarySettings{}, err
	}
	autoScan := true
	if request.AutoScan != nil {
		autoScan = *request.AutoScan
	}

	s.mu.Lock()
	s.settings.Paths = paths
	s.settings.AutoScan = autoScan
	s.settings.Status = "媒体库已设置，等待扫描"
	s.settings.TrackCount = len(s.tracks)
	settings := s.settings
	settings.Paths = append([]string{}, settings.Paths...)
	err = s.saveLocked()
	s.mu.Unlock()
	if err != nil {
		return LibrarySettings{}, err
	}
	if autoScan {
		if _, err := s.Scan(ctx); err != nil {
			return LibrarySettings{}, err
		}
		return s.Settings(ctx)
	}
	return settings, nil
}

func (s *Service) Scan(ctx context.Context) (ScanResult, error) {
	if err := ctx.Err(); err != nil {
		return ScanResult{}, err
	}
	s.mu.RLock()
	paths := append([]string(nil), s.settings.Paths...)
	s.mu.RUnlock()
	if len(paths) == 0 {
		return ScanResult{}, fmt.Errorf("music library path is required")
	}

	now := time.Now().UTC()
	nextTracks := make([]Track, 0)
	for _, root := range paths {
		if err := ctx.Err(); err != nil {
			return ScanResult{}, err
		}
		info, err := os.Stat(root)
		if err != nil {
			return ScanResult{}, fmt.Errorf("stat library path %s: %w", root, err)
		}
		if !info.IsDir() {
			return ScanResult{}, fmt.Errorf("library path is not a directory: %s", root)
		}
		err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			name := entry.Name()
			if entry.IsDir() {
				if strings.HasPrefix(name, ".") && path != root {
					return filepath.SkipDir
				}
				return nil
			}
			ext := strings.ToLower(filepath.Ext(name))
			codec, ok := audioExtensions[ext]
			if !ok {
				return nil
			}
			track, err := trackFromFile(root, path, codec, now)
			if err != nil {
				return nil
			}
			nextTracks = append(nextTracks, track)
			return nil
		})
		if err != nil {
			return ScanResult{}, err
		}
	}
	sort.Slice(nextTracks, func(i, j int) bool {
		if nextTracks[i].Album == nextTracks[j].Album {
			if nextTracks[i].TrackNumber != nextTracks[j].TrackNumber {
				return nextTracks[i].TrackNumber < nextTracks[j].TrackNumber
			}
			return nextTracks[i].Title < nextTracks[j].Title
		}
		return nextTracks[i].Album < nextTracks[j].Album
	})

	s.mu.Lock()
	s.tracks = nextTracks
	s.settings.TrackCount = len(nextTracks)
	s.settings.LastScanAt = now.Format(time.RFC3339)
	s.settings.Status = "扫描完成"
	err := s.saveLocked()
	s.mu.Unlock()
	if err != nil {
		return ScanResult{}, err
	}
	return ScanResult{
		ID:         "music-scan-" + now.Format("20060102150405"),
		State:      "已完成",
		Message:    fmt.Sprintf("已扫描 %d 首音乐。", len(nextTracks)),
		TrackCount: len(nextTracks),
		ScannedAt:  now.Format(time.RFC3339),
	}, nil
}

func (s *Service) Tracks(ctx context.Context, query string) ([]Track, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	query = strings.TrimSpace(strings.ToLower(query))
	s.mu.RLock()
	defer s.mu.RUnlock()
	if query == "" {
		return cloneTracks(s.tracks), nil
	}
	matches := make([]Track, 0, len(s.tracks))
	for _, track := range s.tracks {
		haystack := strings.ToLower(track.Title + " " + track.Artist + " " + track.Album + " " + track.FileName)
		if strings.Contains(haystack, query) {
			matches = append(matches, track)
		}
	}
	return cloneTracks(matches), nil
}

func (s *Service) Albums(ctx context.Context) ([]Album, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	byKey := map[string]Album{}
	for _, track := range s.tracks {
		key := strings.ToLower(track.Album + "\x00" + track.Artist)
		album := byKey[key]
		if album.ID == "" {
			album = Album{
				ID:       stableID(track.Album + "\x00" + track.Artist),
				Name:     track.Album,
				Artist:   track.Artist,
				CoverURL: track.CoverURL,
			}
		}
		album.Count++
		if album.CoverURL == "" {
			album.CoverURL = track.CoverURL
		}
		byKey[key] = album
	}
	albums := make([]Album, 0, len(byKey))
	for _, album := range byKey {
		albums = append(albums, album)
	}
	sort.Slice(albums, func(i, j int) bool { return albums[i].Name < albums[j].Name })
	return albums, nil
}

func (s *Service) Track(ctx context.Context, id string) (Track, error) {
	if err := ctx.Err(); err != nil {
		return Track{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, track := range s.tracks {
		if track.ID == id {
			return cloneTrack(track), nil
		}
	}
	return Track{}, fmt.Errorf("music track not found: %s", id)
}

func (s *Service) OpenTrack(ctx context.Context, id string) (*os.File, Track, string, error) {
	track, err := s.Track(ctx, id)
	if err != nil {
		return nil, Track{}, "", err
	}
	file, err := os.Open(track.Path)
	if err != nil {
		return nil, Track{}, "", err
	}
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(track.Path)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return file, track, contentType, nil
}

func (s *Service) OpenCover(ctx context.Context, id string) (readSeekCloser, string, string, time.Time, error) {
	track, err := s.Track(ctx, id)
	if err != nil {
		return nil, "", "", time.Time{}, err
	}
	path := coverPathForTrack(track.Path)
	if path != "" {
		file, err := os.Open(path)
		if err != nil {
			return nil, "", "", time.Time{}, err
		}
		info, err := file.Stat()
		if err != nil {
			_ = file.Close()
			return nil, "", "", time.Time{}, err
		}
		contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
		if contentType == "" {
			contentType = "image/jpeg"
		}
		return file, contentType, filepath.Base(path), info.ModTime(), nil
	}
	embedded, err := embeddedCoverForTrack(track.Path)
	if err != nil {
		return nil, "", "", time.Time{}, err
	}
	return embedded.reader, embedded.contentType, embedded.name, embedded.modTime, nil
}

func (s *Service) Lyrics(ctx context.Context, id string) (string, error) {
	track, err := s.Track(ctx, id)
	if err != nil {
		return "", err
	}
	path := lyricsPathForTrack(track.Path)
	if path == "" {
		return "", fmt.Errorf("music lyrics not found")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, snapshot{
		Settings: s.settings,
		Tracks:   cloneTracks(s.tracks),
	})
}

func normalizeLibraryPaths(paths []string) ([]string, error) {
	seen := map[string]bool{}
	normalized := make([]string, 0, len(paths))
	for _, raw := range paths {
		path := strings.TrimSpace(raw)
		if path == "" {
			continue
		}
		if strings.HasPrefix(path, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
			}
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		abs = filepath.Clean(abs)
		if seen[abs] {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf("stat library path %s: %w", abs, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("library path is not a directory: %s", abs)
		}
		seen[abs] = true
		normalized = append(normalized, abs)
	}
	return normalized, nil
}

func trackFromFile(root, path, codec string, now time.Time) (Track, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Track{}, err
	}
	title, artist, album, trackNo := inferTrackInfo(root, path)
	id := stableID(path)
	track := Track{
		ID:           id,
		Title:        title,
		Artist:       artist,
		Album:        album,
		TrackNumber:  trackNo,
		Codec:        codec,
		Format:       strings.TrimPrefix(strings.ToUpper(filepath.Ext(path)), "."),
		SizeBytes:    info.Size(),
		Size:         formatBytes(info.Size()),
		ModifiedAt:   info.ModTime().UTC().Format(time.RFC3339),
		DiscoveredAt: now.Format(time.RFC3339),
		FileName:     filepath.Base(path),
		Path:         path,
		StreamURL:    "/api/v1/music/tracks/" + id + "/stream",
		Status:       "已索引 · 浏览器原生解码",
	}
	if coverPathForTrack(path) != "" || hasEmbeddedCover(path) {
		track.CoverURL = "/api/v1/music/tracks/" + id + "/cover"
	}
	if lyricsPathForTrack(path) != "" {
		track.LyricsURL = "/api/v1/music/tracks/" + id + "/lyrics"
	}
	return track, nil
}

func inferTrackInfo(root, path string) (title, artist, album string, trackNo int) {
	fileBase := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	cleanBase := strings.TrimSpace(fileBase)
	fields := strings.Fields(cleanBase)
	if len(fields) > 1 {
		if no, err := strconv.Atoi(strings.Trim(fields[0], ".-_")); err == nil {
			trackNo = no
			cleanBase = strings.TrimSpace(strings.TrimPrefix(cleanBase, fields[0]))
			cleanBase = strings.TrimLeft(cleanBase, " .-_")
		}
	}
	album = filepath.Base(filepath.Dir(path))
	if album == "." || album == string(filepath.Separator) || samePath(filepath.Dir(path), root) {
		album = "未归类专辑"
	}
	artist = "未知艺术家"
	title = cleanBase
	parts := strings.Split(cleanBase, " - ")
	if len(parts) >= 2 {
		artist = strings.TrimSpace(parts[0])
		title = strings.TrimSpace(strings.Join(parts[1:], " - "))
	}
	if title == "" {
		title = fileBase
	}
	return title, artist, album, trackNo
}

func coverPathForTrack(trackPath string) string {
	dir := filepath.Dir(trackPath)
	base := strings.TrimSuffix(trackPath, filepath.Ext(trackPath))
	candidates := []string{
		base + ".jpg", base + ".jpeg", base + ".png", base + ".webp",
		filepath.Join(dir, "cover.jpg"), filepath.Join(dir, "cover.png"),
		filepath.Join(dir, "folder.jpg"), filepath.Join(dir, "folder.png"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func hasEmbeddedCover(trackPath string) bool {
	_, err := embeddedCoverForTrack(trackPath)
	return err == nil
}

func embeddedCoverForTrack(trackPath string) (coverAsset, error) {
	ext := strings.ToLower(filepath.Ext(trackPath))
	switch ext {
	case ".mp3":
		return embeddedMP3Cover(trackPath)
	case ".flac":
		return embeddedFLACCover(trackPath)
	default:
		return coverAsset{}, fmt.Errorf("embedded cover unsupported for %s", ext)
	}
}

func embeddedMP3Cover(trackPath string) (coverAsset, error) {
	data, err := os.ReadFile(trackPath)
	if err != nil {
		return coverAsset{}, err
	}
	if len(data) < 10 || string(data[:3]) != "ID3" {
		return coverAsset{}, fmt.Errorf("id3 tag not found")
	}
	version := data[3]
	tagSize := syncsafeInt(data[6:10])
	end := min(len(data), 10+tagSize)
	offset := 10
	for offset+10 <= end {
		frameID := string(data[offset : offset+4])
		if strings.Trim(frameID, "\x00") == "" {
			break
		}
		sizeBytes := data[offset+4 : offset+8]
		frameSize := int(binary.BigEndian.Uint32(sizeBytes))
		if version == 4 {
			frameSize = syncsafeInt(sizeBytes)
		}
		contentStart := offset + 10
		contentEnd := contentStart + frameSize
		if frameSize <= 0 || contentEnd > end {
			break
		}
		if frameID == "APIC" {
			return parseAPICFrame(data[contentStart:contentEnd], trackPath)
		}
		offset = contentEnd
	}
	return coverAsset{}, fmt.Errorf("apic frame not found")
}

func parseAPICFrame(frame []byte, trackPath string) (coverAsset, error) {
	if len(frame) < 4 {
		return coverAsset{}, fmt.Errorf("invalid apic frame")
	}
	encoding := frame[0]
	mimeEnd := bytes.IndexByte(frame[1:], 0)
	if mimeEnd < 0 {
		return coverAsset{}, fmt.Errorf("invalid apic mime")
	}
	mimeType := string(frame[1 : 1+mimeEnd])
	offset := 1 + mimeEnd + 1
	if offset >= len(frame) {
		return coverAsset{}, fmt.Errorf("invalid apic payload")
	}
	offset++ // picture type
	descEnd := -1
	if encoding == 1 || encoding == 2 {
		for i := offset; i+1 < len(frame); i += 2 {
			if frame[i] == 0 && frame[i+1] == 0 {
				descEnd = i + 2
				break
			}
		}
	} else if i := bytes.IndexByte(frame[offset:], 0); i >= 0 {
		descEnd = offset + i + 1
	}
	if descEnd < 0 || descEnd >= len(frame) {
		return coverAsset{}, fmt.Errorf("invalid apic description")
	}
	image := append([]byte(nil), frame[descEnd:]...)
	if len(image) == 0 {
		return coverAsset{}, fmt.Errorf("empty apic image")
	}
	if mimeType == "" {
		mimeType = httpContentTypeFromImage(image)
	}
	return coverAssetFromBytes(image, mimeType, trackPath, modTime(trackPath)), nil
}

func embeddedFLACCover(trackPath string) (coverAsset, error) {
	data, err := os.ReadFile(trackPath)
	if err != nil {
		return coverAsset{}, err
	}
	if len(data) < 8 || string(data[:4]) != "fLaC" {
		return coverAsset{}, fmt.Errorf("flac marker not found")
	}
	offset := 4
	for offset+4 <= len(data) {
		header := data[offset]
		blockType := header & 0x7f
		blockLen := int(data[offset+1])<<16 | int(data[offset+2])<<8 | int(data[offset+3])
		offset += 4
		if blockLen < 0 || offset+blockLen > len(data) {
			break
		}
		block := data[offset : offset+blockLen]
		if blockType == 6 {
			return parseFLACPicture(block, trackPath)
		}
		offset += blockLen
		if header&0x80 != 0 {
			break
		}
	}
	return coverAsset{}, fmt.Errorf("flac picture block not found")
}

func parseFLACPicture(block []byte, trackPath string) (coverAsset, error) {
	offset := 0
	readUint32 := func() (uint32, bool) {
		if offset+4 > len(block) {
			return 0, false
		}
		value := binary.BigEndian.Uint32(block[offset : offset+4])
		offset += 4
		return value, true
	}
	if _, ok := readUint32(); !ok {
		return coverAsset{}, fmt.Errorf("invalid flac picture")
	}
	mimeLen, ok := readUint32()
	if !ok || offset+int(mimeLen) > len(block) {
		return coverAsset{}, fmt.Errorf("invalid flac picture mime")
	}
	mimeType := string(block[offset : offset+int(mimeLen)])
	offset += int(mimeLen)
	descLen, ok := readUint32()
	if !ok || offset+int(descLen) > len(block) {
		return coverAsset{}, fmt.Errorf("invalid flac picture description")
	}
	offset += int(descLen)
	for i := 0; i < 4; i++ {
		if _, ok := readUint32(); !ok {
			return coverAsset{}, fmt.Errorf("invalid flac picture metadata")
		}
	}
	imageLen, ok := readUint32()
	if !ok || offset+int(imageLen) > len(block) {
		return coverAsset{}, fmt.Errorf("invalid flac picture data")
	}
	image := append([]byte(nil), block[offset:offset+int(imageLen)]...)
	if mimeType == "" {
		mimeType = httpContentTypeFromImage(image)
	}
	return coverAssetFromBytes(image, mimeType, trackPath, modTime(trackPath)), nil
}

func coverAssetFromBytes(data []byte, contentType, trackPath string, modTime time.Time) coverAsset {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return coverAsset{
		reader:      nopReadSeekCloser{bytes.NewReader(data)},
		contentType: contentType,
		name:        strings.TrimSuffix(filepath.Base(trackPath), filepath.Ext(trackPath)) + coverExtension(contentType),
		modTime:     modTime,
	}
}

func coverExtension(contentType string) string {
	if ext, err := mime.ExtensionsByType(contentType); err == nil && len(ext) > 0 {
		return ext[0]
	}
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".cover"
	}
}

func httpContentTypeFromImage(data []byte) string {
	if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
		return "image/jpeg"
	}
	if len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		return "image/png"
	}
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	return "application/octet-stream"
}

func syncsafeInt(data []byte) int {
	if len(data) < 4 {
		return 0
	}
	return int(data[0]&0x7f)<<21 | int(data[1]&0x7f)<<14 | int(data[2]&0x7f)<<7 | int(data[3]&0x7f)
}

func modTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Now()
	}
	return info.ModTime()
}

func lyricsPathForTrack(trackPath string) string {
	candidate := strings.TrimSuffix(trackPath, filepath.Ext(trackPath)) + ".lrc"
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		return candidate
	}
	return ""
}

func stableID(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])[:16]
}

func formatBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func samePath(left, right string) bool {
	leftAbs, _ := filepath.Abs(left)
	rightAbs, _ := filepath.Abs(right)
	return filepath.Clean(leftAbs) == filepath.Clean(rightAbs)
}

func cloneTracks(tracks []Track) []Track {
	out := make([]Track, len(tracks))
	for i, track := range tracks {
		out[i] = cloneTrack(track)
	}
	return out
}

func cloneTrack(track Track) Track {
	return track
}
