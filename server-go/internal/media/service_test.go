package media

import (
	"context"
	"strings"
	"testing"
)

func TestNewServiceSeedsFrontendMediaLibrary(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	items, err := service.Items(ctx, ItemFilter{})
	if err != nil {
		t.Fatalf("items: %v", err)
	}
	if len(items) != 6 {
		t.Fatalf("seeded items = %d, want 6", len(items))
	}
	if items[0].Title != "春节团圆 4K 合影" || items[0].Timeline != "2026 春节" || items[0].People != "爸爸 / 妈妈" {
		t.Fatalf("first item does not match PhotoMediaWindow seed: %#v", items[0])
	}

	albums, err := service.Albums(ctx)
	if err != nil {
		t.Fatalf("albums: %v", err)
	}
	if len(albums) != 6 {
		t.Fatalf("seeded albums = %d, want 6", len(albums))
	}
	if albums[0].Name != "家庭年度相册" || albums[0].Type != AlbumTypeFamily {
		t.Fatalf("first album does not match frontend seed: %#v", albums[0])
	}

	people := service.People(ctx)
	if len(people) < 3 || people[0].Name != "爸爸" {
		t.Fatalf("seeded people mismatch: %#v", people)
	}
}

func TestItemsFiltersByFrontendDimensions(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	tests := []struct {
		name      string
		filter    ItemFilter
		wantTitle string
	}{
		{name: "timeline", filter: ItemFilter{Dimension: DimensionTimeline, Facet: "2026 春节"}, wantTitle: "春节团圆 4K 合影"},
		{name: "people exact cluster", filter: ItemFilter{Dimension: DimensionPeople, Facet: "小雨"}, wantTitle: "家庭钢琴练习"},
		{name: "place", filter: ItemFilter{Dimension: DimensionPlaces, Facet: "三亚"}, wantTitle: "海边旅行 vlog"},
		{name: "device", filter: ItemFilter{Dimension: DimensionDevices, Facet: "DJI Osmo"}, wantTitle: "露营星空延时"},
		{name: "album", filter: ItemFilter{Dimension: DimensionAlbums, Facet: "旅行视频"}, wantTitle: "海边旅行 vlog"},
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
	service := NewService()
	ctx := context.Background()

	album, err := service.CreateAlbum(ctx, CreateAlbumRequest{
		Name:    "外婆精选",
		Type:    AlbumTypeShared,
		ItemIDs: []int{1, 4},
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
	service := NewService()
	ctx := context.Background()

	first, err := service.CreateMemory(ctx, CreateMemoryRequest{Dimension: DimensionTimeline, Facet: "2026 春节"})
	if err != nil {
		t.Fatalf("create first memory: %v", err)
	}
	second, err := service.CreateMemory(ctx, CreateMemoryRequest{Dimension: DimensionPlaces, Facet: "三亚"})
	if err != nil {
		t.Fatalf("create second memory: %v", err)
	}

	if first.RunCount != 1 || first.Album.Name != "AI 回忆 1" {
		t.Fatalf("first memory mismatch: %#v", first)
	}
	if second.RunCount != 2 || second.Album.Name != "AI 回忆 2" {
		t.Fatalf("second memory mismatch: %#v", second)
	}
	if first.Status != JobStatusQueued || !strings.Contains(first.Message, "素材来自 2026 春节") {
		t.Fatalf("memory job response mismatch: %#v", first)
	}
}

func TestMergePeopleKeepsRollbackNoticeAndUpdatesMatchingItems(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	result, err := service.MergePeople(ctx, MergePeopleRequest{
		SourceNames: []string{"爸爸", "妈妈"},
		TargetName:  "爸爸 / 妈妈 · 已合并",
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

	items, err := service.Items(ctx, ItemFilter{Dimension: DimensionPeople, Facet: "爸爸 / 妈妈 · 已合并"})
	if err != nil {
		t.Fatalf("items by merged people: %v", err)
	}
	if len(items) == 0 || items[0].Status != "人物合并待确认" {
		t.Fatalf("merged items mismatch: %#v", items)
	}
}

func TestCreateSubtitleTranscodeAndShareJobs(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	subtitle, err := service.CreateSubtitleJob(ctx, CreateMediaJobRequest{ItemID: 2})
	if err != nil {
		t.Fatalf("subtitle job: %v", err)
	}
	if subtitle.ID == "" || subtitle.Status != JobStatusQueued || !strings.Contains(subtitle.Message, "字幕匹配任务") {
		t.Fatalf("subtitle job mismatch: %#v", subtitle)
	}

	transcode, err := service.CreateTranscodeJob(ctx, CreateMediaJobRequest{ItemID: 4, Profile: "1080p 家庭共享版本"})
	if err != nil {
		t.Fatalf("transcode job: %v", err)
	}
	if transcode.ID == "" || transcode.Status != JobStatusRunning || transcode.Profile != "1080p 家庭共享版本" {
		t.Fatalf("transcode job mismatch: %#v", transcode)
	}

	share, err := service.CreateShare(ctx, CreateShareRequest{AlbumID: 2, ExpiresInDays: 7})
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
