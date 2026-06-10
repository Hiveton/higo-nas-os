package music

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestScanLibraryFindsAudioLyricsAndCover(t *testing.T) {
	dir := t.TempDir()
	albumDir := filepath.Join(dir, "专辑A")
	if err := os.MkdirAll(albumDir, 0o755); err != nil {
		t.Fatal(err)
	}
	audioPath := filepath.Join(albumDir, "01 Artist - Song.mp3")
	if err := os.WriteFile(audioPath, []byte("fake audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(albumDir, "01 Artist - Song.lrc"), []byte("[00:01.00]歌词"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(albumDir, "cover.jpg"), []byte("cover"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := NewService()
	autoScan := false
	if _, err := service.UpdateSettings(context.Background(), LibraryUpdateRequest{Paths: []string{dir}, AutoScan: &autoScan}); err != nil {
		t.Fatal(err)
	}
	result, err := service.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.TrackCount != 1 {
		t.Fatalf("expected 1 track, got %d", result.TrackCount)
	}
	tracks, err := service.Tracks(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(tracks))
	}
	track := tracks[0]
	if track.Title != "Song" || track.Artist != "Artist" || track.Album != "专辑A" {
		t.Fatalf("unexpected track metadata: %#v", track)
	}
	if track.CoverURL == "" || track.LyricsURL == "" || track.StreamURL == "" {
		t.Fatalf("expected media URLs, got %#v", track)
	}
	lyrics, err := service.Lyrics(context.Background(), track.ID)
	if err != nil {
		t.Fatal(err)
	}
	if lyrics != "[00:01.00]歌词" {
		t.Fatalf("unexpected lyrics: %q", lyrics)
	}
}

func TestScanLibraryFindsEmbeddedMP3Cover(t *testing.T) {
	dir := t.TempDir()
	image := []byte{0xff, 0xd8, 0xff, 0xdb, 0x00, 0x43, 0x00, 0xff, 0xd9}
	audioPath := filepath.Join(dir, "Artist - Embedded.mp3")
	if err := os.WriteFile(audioPath, testMP3WithAPIC(image), 0o644); err != nil {
		t.Fatal(err)
	}

	service := NewService()
	autoScan := false
	if _, err := service.UpdateSettings(context.Background(), LibraryUpdateRequest{Paths: []string{dir}, AutoScan: &autoScan}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	tracks, err := service.Tracks(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 || tracks[0].CoverURL == "" {
		t.Fatalf("expected embedded cover URL, got %#v", tracks)
	}
	cover, contentType, _, _, err := service.OpenCover(context.Background(), tracks[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	defer cover.Close()
	if contentType != "image/jpeg" {
		t.Fatalf("expected jpeg cover, got %s", contentType)
	}
	data, err := io.ReadAll(cover)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, image) {
		t.Fatalf("unexpected cover bytes: %x", data)
	}
}

func TestScanLibraryFindsEmbeddedFLACCover(t *testing.T) {
	dir := t.TempDir()
	image := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	audioPath := filepath.Join(dir, "Artist - Embedded.flac")
	if err := os.WriteFile(audioPath, testFLACWithPicture(image), 0o644); err != nil {
		t.Fatal(err)
	}

	service := NewService()
	autoScan := false
	if _, err := service.UpdateSettings(context.Background(), LibraryUpdateRequest{Paths: []string{dir}, AutoScan: &autoScan}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	tracks, err := service.Tracks(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 || tracks[0].CoverURL == "" {
		t.Fatalf("expected embedded cover URL, got %#v", tracks)
	}
	cover, contentType, _, _, err := service.OpenCover(context.Background(), tracks[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	defer cover.Close()
	if contentType != "image/png" {
		t.Fatalf("expected png cover, got %s", contentType)
	}
	data, err := io.ReadAll(cover)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, image) {
		t.Fatalf("unexpected cover bytes: %x", data)
	}
}

func TestScanLibraryReadsFLACDuration(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "Artist - Duration.flac")
	if err := os.WriteFile(audioPath, testFLACWithStreamInfo(48000, 48000*125), 0o644); err != nil {
		t.Fatal(err)
	}

	service := NewService()
	autoScan := false
	if _, err := service.UpdateSettings(context.Background(), LibraryUpdateRequest{Paths: []string{dir}, AutoScan: &autoScan}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	tracks, err := service.Tracks(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(tracks))
	}
	if tracks[0].DurationSeconds != 125 {
		t.Fatalf("expected 125 seconds, got %d", tracks[0].DurationSeconds)
	}
}

func testMP3WithAPIC(image []byte) []byte {
	framePayload := append([]byte{0, 'i', 'm', 'a', 'g', 'e', '/', 'j', 'p', 'e', 'g', 0, 3, 0}, image...)
	frame := append([]byte{'A', 'P', 'I', 'C'}, uint32Bytes(uint32(len(framePayload)))...)
	frame = append(frame, 0, 0)
	frame = append(frame, framePayload...)
	header := []byte{'I', 'D', '3', 3, 0, 0}
	header = append(header, syncsafeBytes(len(frame))...)
	return append(append(header, frame...), []byte("audio")...)
}

func testFLACWithStreamInfo(sampleRate int, totalSamples uint64) []byte {
	block := make([]byte, 34)
	block[10] = byte(sampleRate >> 12)
	block[11] = byte(sampleRate >> 4)
	block[12] = byte((sampleRate & 0x0f) << 4)
	block[13] = byte(totalSamples>>32) & 0x0f
	block[14] = byte(totalSamples >> 24)
	block[15] = byte(totalSamples >> 16)
	block[16] = byte(totalSamples >> 8)
	block[17] = byte(totalSamples)
	return append([]byte{'f', 'L', 'a', 'C', 0x80, 0, 0, byte(len(block))}, block...)
}

func testFLACWithPicture(image []byte) []byte {
	var picture bytes.Buffer
	_ = binary.Write(&picture, binary.BigEndian, uint32(3))
	_ = binary.Write(&picture, binary.BigEndian, uint32(len("image/png")))
	picture.WriteString("image/png")
	_ = binary.Write(&picture, binary.BigEndian, uint32(0))
	_ = binary.Write(&picture, binary.BigEndian, uint32(1))
	_ = binary.Write(&picture, binary.BigEndian, uint32(1))
	_ = binary.Write(&picture, binary.BigEndian, uint32(24))
	_ = binary.Write(&picture, binary.BigEndian, uint32(0))
	_ = binary.Write(&picture, binary.BigEndian, uint32(len(image)))
	picture.Write(image)
	block := picture.Bytes()
	return append([]byte{'f', 'L', 'a', 'C', 0x86, byte(len(block) >> 16), byte(len(block) >> 8), byte(len(block))}, block...)
}

func uint32Bytes(value uint32) []byte {
	return []byte{byte(value >> 24), byte(value >> 16), byte(value >> 8), byte(value)}
}

func syncsafeBytes(value int) []byte {
	return []byte{byte(value >> 21), byte(value >> 14), byte(value >> 7), byte(value)}
}
