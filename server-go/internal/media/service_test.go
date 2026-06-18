package media

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"higoos/server-go/internal/tasks"
)

func TestTranscodeAndSubtitleJobsAreDrivenToReadyByRunner(t *testing.T) {
	root, _ := createMediaFixture(t)
	service, err := NewServiceWithRoots("", []string{root})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	mgr, err := tasks.NewManager("", tasks.WithWorkers(2), tasks.WithDispatchInterval(20*time.Millisecond))
	if err != nil {
		t.Fatalf("new task manager: %v", err)
	}
	service.AttachTaskRunner(mgr)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mgr.Start(ctx)
	defer mgr.Stop()

	items, err := service.Items(ctx, ItemFilter{})
	if err != nil || len(items) == 0 {
		t.Fatalf("need at least one media item: err=%v len=%d", err, len(items))
	}
	itemID := items[0].ID

	transcode, err := service.CreateTranscodeJob(ctx, CreateMediaJobRequest{ItemID: itemID})
	if err != nil {
		t.Fatalf("create transcode job: %v", err)
	}
	subtitle, err := service.CreateSubtitleJob(ctx, CreateMediaJobRequest{ItemID: itemID})
	if err != nil {
		t.Fatalf("create subtitle job: %v", err)
	}

	transcodeStatus := func() JobStatus {
		service.mu.RLock()
		defer service.mu.RUnlock()
		for _, j := range service.transcodeJobs {
			if j.ID == transcode.ID {
				return j.Status
			}
		}
		return ""
	}
	subtitleStatus := func() JobStatus {
		service.mu.RLock()
		defer service.mu.RUnlock()
		for _, j := range service.subtitleJobs {
			if j.ID == subtitle.ID {
				return j.Status
			}
		}
		return ""
	}
	waitReady := func(kind string, status func() JobStatus) {
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if status() == JobStatusReady {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatalf("%s job did not reach ready, last status=%q", kind, status())
	}
	waitReady("transcode", transcodeStatus)
	waitReady("subtitle", subtitleStatus)
}

func TestNewServiceWithRootsScansActualMediaFiles(t *testing.T) {
	root, timeline := createMediaFixture(t)
	service, err := NewServiceWithRoots("", []string{root})
	if err != nil {
		t.Fatalf("new service with roots: %v", err)
	}
	ctx := context.Background()

	items, err := service.Items(ctx, ItemFilter{})
	if err != nil {
		t.Fatalf("items: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("scanned items = %d, want 2: %#v", len(items), items)
	}
	if items[0].Title != "family-photo" || items[0].Timeline != timeline || items[0].People != "待 AI 识别" {
		t.Fatalf("first scanned item mismatch: %#v", items[0])
	}
	if strings.Contains(items[0].Title, "春节") || strings.Contains(items[0].Timeline, "春节") {
		t.Fatalf("scanned media should not contain demo labels: %#v", items[0])
	}

	albums, err := service.Albums(ctx)
	if err != nil {
		t.Fatalf("albums: %v", err)
	}
	if len(albums) != 2 {
		t.Fatalf("albums = %d, want 2: %#v", len(albums), albums)
	}
	if albums[0].Name != "photos" || albums[0].Count != 1 {
		t.Fatalf("directory album mismatch: %#v", albums[0])
	}

	people := service.People(ctx)
	if len(people) != 0 {
		t.Fatalf("pending AI people should not create fake people facets: %#v", people)
	}
}

func TestNewServiceWithRootsExcludesMusicAndVideoLibraries(t *testing.T) {
	root, _ := createMediaFixture(t)
	service, err := NewServiceWithRootsAndExcludes("", []string{root}, []string{filepath.Join(root, "videos")})
	if err != nil {
		t.Fatalf("new service with excludes: %v", err)
	}

	items, err := service.Items(context.Background(), ItemFilter{})
	if err != nil {
		t.Fatalf("items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items after excludes = %d, want 1: %#v", len(items), items)
	}
	if items[0].Title != "family-photo" || items[0].Kind != MediaKindPhoto {
		t.Fatalf("expected only photo library item, got %#v", items)
	}
}

func TestNewServiceWithStateDirReadsMusicAndVideoLibraryExcludes(t *testing.T) {
	root, _ := createMediaFixture(t)
	stateDir := t.TempDir()
	t.Setenv("HIGO_MEDIA_ROOTS", root)
	musicState := `{"settings":{"paths":["` + filepath.ToSlash(filepath.Join(root, "photos")) + `"]}}`
	videoState := `{"settings":{"libraries":[{"paths":["` + filepath.ToSlash(filepath.Join(root, "videos")) + `"]}]}}`
	if err := os.WriteFile(filepath.Join(stateDir, "music.json"), []byte(musicState), 0o600); err != nil {
		t.Fatalf("write music state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "video.json"), []byte(videoState), 0o600); err != nil {
		t.Fatalf("write video state: %v", err)
	}

	service, err := NewServiceWithStateDir(stateDir)
	if err != nil {
		t.Fatalf("new service with state dir: %v", err)
	}
	items, err := service.Items(context.Background(), ItemFilter{})
	if err != nil {
		t.Fatalf("items: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("photo media should exclude music/video library roots, got %#v", items)
	}
}

func TestItemsFiltersByScannedDimensions(t *testing.T) {
	root, timeline := createMediaFixture(t)
	service, err := NewServiceWithRoots("", []string{root})
	if err != nil {
		t.Fatalf("new service with roots: %v", err)
	}
	ctx := context.Background()

	tests := []struct {
		name      string
		filter    ItemFilter
		wantTitle string
	}{
		{name: "timeline", filter: ItemFilter{Dimension: DimensionTimeline, Facet: timeline}, wantTitle: "family-photo"},
		{name: "people pending ai", filter: ItemFilter{Dimension: DimensionPeople, Facet: "待 AI 识别"}, wantTitle: "family-photo"},
		{name: "place pending ai", filter: ItemFilter{Dimension: DimensionPlaces, Facet: "待 AI 识别"}, wantTitle: "family-photo"},
		{name: "device pending ai", filter: ItemFilter{Dimension: DimensionDevices, Facet: "待 AI 识别"}, wantTitle: "family-photo"},
		{name: "album", filter: ItemFilter{Dimension: DimensionAlbums, Facet: "videos"}, wantTitle: "trip-video"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := service.Items(ctx, tt.filter)
			if err != nil {
				t.Fatalf("items: %v", err)
			}
			if len(items) == 0 {
				t.Fatalf("filter returned no items: %#v", tt.filter)
			}
			if items[0].Title != tt.wantTitle {
				t.Fatalf("first title = %q, want %q; items=%#v", items[0].Title, tt.wantTitle, items)
			}
		})
	}
}

func TestCreateAlbumPrependsSharedAlbumAndLinksSelectedItems(t *testing.T) {
	root, _ := createMediaFixture(t)
	service, err := NewServiceWithRoots("", []string{root})
	if err != nil {
		t.Fatalf("new service with roots: %v", err)
	}
	ctx := context.Background()

	album, err := service.CreateAlbum(ctx, CreateAlbumRequest{
		Name:    "外婆精选",
		Type:    AlbumTypeShared,
		ItemIDs: []int{1, 2},
		Privacy: "亲友可见",
	})
	if err != nil {
		t.Fatalf("create album: %v", err)
	}
	if album.ID == 0 || album.Name != "外婆精选" || album.Type != AlbumTypeShared || album.Count != 2 {
		t.Fatalf("created album mismatch: %#v", album)
	}

	albums, _ := service.Albums(ctx)
	if albums[0].ID != album.ID {
		t.Fatalf("new album should be first: %#v", albums)
	}

	items, err := service.Items(ctx, ItemFilter{Dimension: DimensionAlbums, Facet: "外婆精选"})
	if err != nil {
		t.Fatalf("items by new album: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("linked items = %d, want 2: %#v", len(items), items)
	}
}

func TestCreateMemoryIncrementsRunCountAndCreatesSmartAlbum(t *testing.T) {
	root, timeline := createMediaFixture(t)
	service, err := NewServiceWithRoots("", []string{root})
	if err != nil {
		t.Fatalf("new service with roots: %v", err)
	}
	ctx := context.Background()

	first, err := service.CreateMemory(ctx, CreateMemoryRequest{Dimension: DimensionTimeline, Facet: timeline})
	if err != nil {
		t.Fatalf("create first memory: %v", err)
	}
	second, err := service.CreateMemory(ctx, CreateMemoryRequest{Dimension: DimensionAlbums, Facet: "videos"})
	if err != nil {
		t.Fatalf("create second memory: %v", err)
	}

	if first.RunCount != 1 || first.Album.Name != "AI 回忆 1" {
		t.Fatalf("first memory mismatch: %#v", first)
	}
	if second.RunCount != 2 || second.Album.Name != "AI 回忆 2" {
		t.Fatalf("second memory mismatch: %#v", second)
	}
	if first.Status != JobStatusQueued || !strings.Contains(first.Message, "素材来自 "+timeline) {
		t.Fatalf("memory job response mismatch: %#v", first)
	}
}

func TestMergePeopleKeepsRollbackNoticeAndUpdatesMatchingItems(t *testing.T) {
	root, _ := createMediaFixture(t)
	service, err := NewServiceWithRoots("", []string{root})
	if err != nil {
		t.Fatalf("new service with roots: %v", err)
	}
	ctx := context.Background()

	result, err := service.MergePeople(ctx, MergePeopleRequest{
		SourceNames: []string{"待 AI 识别"},
		TargetName:  "家庭成员",
	})
	if err != nil {
		t.Fatalf("merge people: %v", err)
	}
	if result.Status != JobStatusReady || result.ID == "" {
		t.Fatalf("merge result missing task state: %#v", result)
	}
	if !strings.Contains(result.Notice, "30 天可回滚") {
		t.Fatalf("merge notice should mention rollback: %#v", result)
	}

	items, err := service.Items(ctx, ItemFilter{Dimension: DimensionPeople, Facet: "家庭成员"})
	if err != nil {
		t.Fatalf("items by merged people: %v", err)
	}
	if len(items) == 0 || items[0].Status != "人物合并待确认" {
		t.Fatalf("merged items mismatch: %#v", items)
	}
}

func TestCreateSubtitleTranscodeAndShareJobs(t *testing.T) {
	root, _ := createMediaFixture(t)
	service, err := NewServiceWithRoots("", []string{root})
	if err != nil {
		t.Fatalf("new service with roots: %v", err)
	}
	ctx := context.Background()

	subtitle, err := service.CreateSubtitleJob(ctx, CreateMediaJobRequest{ItemID: 2})
	if err != nil {
		t.Fatalf("subtitle job: %v", err)
	}
	if subtitle.ID == "" || subtitle.Status != JobStatusQueued || !strings.Contains(subtitle.Message, "字幕匹配任务") {
		t.Fatalf("subtitle job mismatch: %#v", subtitle)
	}

	transcode, err := service.CreateTranscodeJob(ctx, CreateMediaJobRequest{ItemID: 2, Profile: "1080p 家庭共享版本"})
	if err != nil {
		t.Fatalf("transcode job: %v", err)
	}
	if transcode.ID == "" || transcode.Status != JobStatusRunning || transcode.Profile != "1080p 家庭共享版本" {
		t.Fatalf("transcode job mismatch: %#v", transcode)
	}

	share, err := service.CreateShare(ctx, CreateShareRequest{AlbumID: 1, ExpiresInDays: 7})
	if err != nil {
		t.Fatalf("share: %v", err)
	}
	if share.ID == "" || share.Status != JobStatusReady || share.ExpiresInDays != 7 {
		t.Fatalf("share result mismatch: %#v", share)
	}
	if !strings.Contains(share.Message, "共享链接开启") || !strings.Contains(share.AuditNotice, "ACL") {
		t.Fatalf("share should include ACL/risk message: %#v", share)
	}
}

func createMediaFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	photos := filepath.Join(root, "photos")
	videos := filepath.Join(root, "videos")
	if err := os.MkdirAll(photos, 0o755); err != nil {
		t.Fatalf("mkdir photos: %v", err)
	}
	if err := os.MkdirAll(videos, 0o755); err != nil {
		t.Fatalf("mkdir videos: %v", err)
	}
	baseTime := time.Date(2026, 6, 3, 12, 0, 0, 0, time.Local)
	files := []string{
		filepath.Join(photos, "family-photo.jpg"),
		filepath.Join(photos, "song.flac"),
		filepath.Join(videos, "trip-video.mp4"),
	}
	for idx, path := range files {
		if err := os.WriteFile(path, []byte(strings.Repeat("x", 1024*(idx+1))), 0o644); err != nil {
			t.Fatalf("write fixture media: %v", err)
		}
		mod := baseTime.Add(time.Duration(-idx) * time.Minute)
		if err := os.Chtimes(path, mod, mod); err != nil {
			t.Fatalf("chtimes fixture media: %v", err)
		}
	}
	return root, baseTime.Format("2006-01")
}
