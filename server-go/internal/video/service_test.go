package video

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestServiceScansLocalLibraryAndCreatesTasks(t *testing.T) {
	ctx := context.Background()
	stateDir := t.TempDir()
	mediaDir := t.TempDir()
	videoPath := filepath.Join(mediaDir, "Galaxy.Movie.2024.1080p.mp4")
	if err := os.WriteFile(videoPath, []byte("not a real video, but enough for library indexing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mediaDir, "Galaxy.Movie.2024.1080p.jpg"), []byte{0xff, 0xd8, 0xff, 0xd9}, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mediaDir, "Galaxy.Movie.2024.1080p.srt"), []byte("1\n00:00:00,000 --> 00:00:01,000\n字幕\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	service, err := NewServiceWithStateDir(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	library, err := service.CreateLibrary(ctx, CreateLibraryRequest{
		Name:             "电影",
		Type:             LibraryMovie,
		Paths:            []string{mediaDir},
		MetadataLanguage: "zh-CN",
		AutoSubtitles:    true,
		SubtitleLanguage: "zh-CN",
	})
	if err != nil {
		t.Fatal(err)
	}
	if library.ID == "" {
		t.Fatal("expected library id")
	}

	result, err := service.Scan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.ItemCount != 1 {
		t.Fatalf("expected 1 item, got %d", result.ItemCount)
	}
	items, err := service.Items(ctx, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one scanned item, got %d", len(items))
	}
	item := items[0]
	if item.Title != "Galaxy Movie" || item.Year != "2024" || item.Resolution != "1080p" {
		t.Fatalf("unexpected item metadata: %+v", item)
	}
	if item.SubtitleURL == "" {
		t.Fatal("expected subtitle url")
	}

	poster, contentType, _, err := service.Poster(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(poster) == 0 || contentType != "image/jpeg" {
		t.Fatalf("unexpected poster response: %q len=%d", contentType, len(poster))
	}
	subtitle, contentType, _, err := service.Subtitle(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(subtitle), "字幕") || contentType != "application/x-subrip; charset=utf-8" {
		t.Fatalf("unexpected subtitle response: %q %q", contentType, string(subtitle))
	}

	for _, create := range []func(context.Context, CreateTaskRequest) (Task, error){
		service.CreateScrapeTask,
		service.CreateSubtitleTask,
		service.CreateTranscodeTask,
	} {
		task, err := create(ctx, CreateTaskRequest{ItemID: item.ID, Profile: "720p H.264"})
		if err != nil {
			t.Fatal(err)
		}
		if task.ID == "" || task.ItemID != item.ID {
			t.Fatalf("unexpected task: %+v", task)
		}
	}
	tasks, err := service.Tasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	deleted, err := service.DeleteLibrary(ctx, library.ID)
	if err != nil {
		t.Fatal(err)
	}
	if deleted.ID != library.ID || deleted.RemovedItems != 1 || deleted.RemovedTasks != 3 {
		t.Fatalf("unexpected delete result: %+v", deleted)
	}
	settings, err := service.Settings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(settings.Libraries) != 0 || settings.ItemCount != 0 {
		t.Fatalf("expected empty library settings after delete: %+v", settings)
	}
	items, err = service.Items(ctx, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected scanned items to be removed, got %d", len(items))
	}
	tasks, err = service.Tasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected tasks to be removed, got %d", len(tasks))
	}
}

func TestParseProbeResultIncludesAudioAndSubtitleTracks(t *testing.T) {
	data := []byte(`{
		"streams": [
			{"index":0,"codec_type":"video","codec_name":"h264","width":1920,"height":1080,"tags":{"language":"und","title":"Main Video"},"disposition":{"default":1}},
			{"index":1,"codec_type":"audio","codec_name":"aac","channels":6,"tags":{"language":"eng","title":"English 5.1"},"disposition":{"default":1}},
			{"index":2,"codec_type":"audio","codec_name":"ac3","channels":2,"tags":{"language":"chi"},"disposition":{"default":0}},
			{"index":3,"codec_type":"subtitle","codec_name":"subrip","tags":{"language":"chi","title":"中文"},"disposition":{"default":1}}
		],
		"format": {"duration": "6899.4"}
	}`)
	result := parseProbeResult(data)
	if result.Codec != "H264" || result.Resolution != "1920x1080" || result.DurationSeconds != 6899 {
		t.Fatalf("unexpected video summary: %+v", result)
	}
	if len(result.VideoTracks) != 1 || result.VideoTracks[0].Title != "1920x1080 Main Video" || !result.VideoTracks[0].Default {
		t.Fatalf("unexpected video tracks: %+v", result.VideoTracks)
	}
	if len(result.AudioTracks) != 2 || result.AudioTracks[0].Codec != "AAC" || result.AudioTracks[0].Channels != "5.1" || result.AudioTracks[1].Language != "chi" {
		t.Fatalf("unexpected audio tracks: %+v", result.AudioTracks)
	}
	if len(result.SubtitleTracks) != 1 || result.SubtitleTracks[0].Codec != "SUBRIP" || result.SubtitleTracks[0].Title != "中文" {
		t.Fatalf("unexpected subtitle tracks: %+v", result.SubtitleTracks)
	}
}

func TestServiceScrapeAppliesSidecarMetadata(t *testing.T) {
	ctx := context.Background()
	mediaDir := t.TempDir()
	videoPath := filepath.Join(mediaDir, "Zootopia.2016.1080p.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	metadata := map[string]any{
		"title":          "疯狂动物城",
		"originalTitle":  "Zootopia",
		"year":           "2016",
		"overview":       "兔子警官朱迪和狐狸尼克联手揭开动物城的阴谋。",
		"rating":         "7.8",
		"contentRating":  "PG",
		"releaseDate":    "2016-03-04",
		"tagline":        "无敌变拍档 玩转动物城",
		"genres":         []string{"动画", "冒险", "喜剧"},
		"tags":           []string{"cartoon", "buddy cop"},
		"directors":      []string{"拜恩·霍华德", "里奇·摩尔"},
		"writers":        []string{"杰拉德·布什"},
		"actors":         []string{"金妮弗·古德温", "杰森·贝特曼"},
		"studios":        []string{"Walt Disney Animation Studios"},
		"countries":      []string{"美国"},
		"metadataSource": "本地元数据",
		"providerId":     "local-zootopia",
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mediaDir, "Zootopia.2016.1080p.metadata.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	service := NewService()
	if _, err := service.CreateLibrary(ctx, CreateLibraryRequest{
		Name:             "电影",
		Type:             LibraryMovie,
		Paths:            []string{mediaDir},
		MetadataLanguage: "zh-CN",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	items, err := service.Items(ctx, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one item, got %d", len(items))
	}
	if _, err := service.CreateScrapeTask(ctx, CreateTaskRequest{ItemID: items[0].ID}); err != nil {
		t.Fatal(err)
	}
	item, err := service.Item(ctx, items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if item.Title != "疯狂动物城" || item.OriginalTitle != "Zootopia" || item.ContentRating != "PG" {
		t.Fatalf("metadata was not applied: %+v", item)
	}
	if len(item.Directors) != 2 || len(item.Actors) != 2 || len(item.Tags) != 2 || item.MetadataSource == "" {
		t.Fatalf("expected rich metadata fields: %+v", item)
	}
}

func TestServiceScrapeAppliesEpisodeMetadata(t *testing.T) {
	ctx := context.Background()
	mediaDir := t.TempDir()
	videoPath := filepath.Join(mediaDir, "Mystery.Show.S01E02.1080p.mkv")
	if err := os.WriteFile(videoPath, []byte("episode"), 0o600); err != nil {
		t.Fatal(err)
	}
	metadata := map[string]any{
		"seriesTitle":     "神秘剧集",
		"episodeTitle":    "第二集",
		"overview":        "整部剧的总简介。",
		"episodeOverview": "第二集的单集简介。",
		"season":          1,
		"episode":         2,
		"genres":          []string{"悬疑"},
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mediaDir, "Mystery.Show.S01E02.1080p.metadata.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	service := NewService()
	if _, err := service.CreateLibrary(ctx, CreateLibraryRequest{
		Name:  "电视剧",
		Type:  LibrarySeries,
		Paths: []string{mediaDir},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	items, err := service.Items(ctx, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one item, got %d", len(items))
	}
	if _, err := service.CreateScrapeTask(ctx, CreateTaskRequest{ItemID: items[0].ID}); err != nil {
		t.Fatal(err)
	}
	item, err := service.Item(ctx, items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if item.SeriesTitle != "神秘剧集" || item.EpisodeTitle != "第二集" || item.Overview != "第二集的单集简介。" {
		t.Fatalf("episode metadata was not applied: %+v", item)
	}
}

func TestServiceScrapeFetchesDoubanSubjectDetails(t *testing.T) {
	ctx := context.Background()
	mediaDir := t.TempDir()
	videoPath := filepath.Join(mediaDir, "Ne.Zha.2.2025.2160p.mkv")
	if err := os.WriteFile(videoPath, []byte("movie"), 0o600); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/j/subject_suggest":
			if r.URL.Query().Get("q") == "" {
				t.Fatalf("expected query title")
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			fmt.Fprint(w, `[{"id":"1292052","title":"哪吒之魔童闹海","year":"2025","img":"https://img.example.test/poster.jpg","type":"movie"}]`)
		case "/subject/1292052/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `<html><head><script type="application/ld+json">{
				"@context":"https://schema.org",
				"name":"哪吒之魔童闹海",
				"alternateName":"Ne Zha 2",
				"datePublished":"2025-01-29",
				"description":"天劫之后，哪吒和敖丙的灵魂保住了。",
				"image":"https://img.example.test/subject-poster.jpg",
				"genre":["动画","奇幻","冒险"],
				"countryOfOrigin":"中国大陆",
				"aggregateRating":{"ratingValue":"8.5"},
				"director":[{"name":"饺子"}],
				"author":[{"name":"饺子"}],
				"actor":[{"name":"吕艳婷"},{"name":"囧森瑟夫"}]
			}</script></head><body></body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("HIGO_VIDEO_DOUBAN_SUGGEST_URL", server.URL+"/j/subject_suggest")
	t.Setenv("HIGO_VIDEO_DOUBAN_REXXAR_BASE_URL", server.URL+"/rexxar/api/v2/movie/")
	t.Setenv("HIGO_VIDEO_DOUBAN_ABSTRACT_URL", server.URL+"/j/subject_abstract")
	t.Setenv("HIGO_VIDEO_DOUBAN_SUBJECT_BASE_URL", server.URL+"/subject/")

	service := NewService()
	if _, err := service.CreateLibrary(ctx, CreateLibraryRequest{
		Name:             "电影",
		Type:             LibraryMovie,
		Paths:            []string{mediaDir},
		MetadataLanguage: "zh-CN",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	items, err := service.Items(ctx, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one item, got %d", len(items))
	}
	if _, err := service.CreateScrapeTask(ctx, CreateTaskRequest{ItemID: items[0].ID}); err != nil {
		t.Fatal(err)
	}
	item, err := service.Item(ctx, items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if item.Title != "哪吒之魔童闹海" || item.OriginalTitle != "Ne Zha 2" || item.Year != "2025" || item.ReleaseDate != "2025-01-29" {
		t.Fatalf("douban identity metadata was not applied: %+v", item)
	}
	if item.Rating != "8.5" || !strings.Contains(item.Overview, "天劫之后") || item.PosterRemoteURL != "https://img.example.test/subject-poster.jpg" {
		t.Fatalf("douban detail metadata was not applied: %+v", item)
	}
	if len(item.Directors) != 1 || item.Directors[0] != "饺子" || len(item.Actors) != 2 || len(item.Genres) != 3 || len(item.Countries) != 1 {
		t.Fatalf("douban people/category metadata was not applied: %+v", item)
	}
	if item.MetadataSource != "豆瓣" || item.ProviderID != "1292052" {
		t.Fatalf("unexpected metadata source: %+v", item)
	}
}

func TestServiceScrapeFetchesDoubanRexxarDetailsAndCachesPoster(t *testing.T) {
	ctx := context.Background()
	mediaDir := t.TempDir()
	videoPath := filepath.Join(mediaDir, "奇异博士.2016.1080p.中英双语双字.mkv")
	if err := os.WriteFile(videoPath, []byte("movie"), 0o600); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/j/subject_suggest":
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			fmt.Fprintf(w, `[{"id":"3025375","title":"奇异博士","year":"2016","img":"%s/poster.jpg","type":"movie","sub_title":"Doctor Strange"}]`, serverURL(r))
		case "/rexxar/api/v2/movie/3025375":
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			fmt.Fprintf(w, `{
				"id":"3025375",
				"title":"奇异博士",
				"original_title":"Doctor Strange",
				"year":"2016",
				"intro":"斯特兰奇博士远赴尼泊尔，寻找治愈双手的神秘力量。",
				"rating":{"value":7.6},
				"pubdate":["2016-11-04(中国大陆)"],
				"genres":["动作","奇幻","冒险"],
				"countries":["美国"],
				"directors":[{"name":"斯科特·德瑞克森"}],
				"actors":[{"name":"本尼迪克特·康伯巴奇"},{"name":"蒂尔达·斯文顿"}],
				"pic":{"normal":"%s/poster.jpg","large":"%s/poster-large.jpg"},
				"aka":["斯特兰奇博士","怪奇医生"]
			}`, serverURL(r), serverURL(r))
		case "/poster.jpg", "/poster-large.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte{0xff, 0xd8, 0xff, 0xd9})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("HIGO_VIDEO_DOUBAN_SUGGEST_URL", server.URL+"/j/subject_suggest")
	t.Setenv("HIGO_VIDEO_DOUBAN_REXXAR_BASE_URL", server.URL+"/rexxar/api/v2/movie/")
	t.Setenv("HIGO_VIDEO_DOUBAN_SUBJECT_BASE_URL", server.URL+"/subject/")

	service, err := NewServiceWithStateDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateLibrary(ctx, CreateLibraryRequest{
		Name:             "电影",
		Type:             LibraryMovie,
		Paths:            []string{mediaDir},
		MetadataLanguage: "zh-CN",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	items, err := service.Items(ctx, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "奇异博士" {
		t.Fatalf("expected cleaned scanned item, got %+v", items)
	}
	if _, err := service.CreateScrapeTask(ctx, CreateTaskRequest{ItemID: items[0].ID}); err != nil {
		t.Fatal(err)
	}
	item, err := service.Item(ctx, items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if item.Title != "奇异博士" || item.OriginalTitle != "Doctor Strange" || item.Rating != "7.6" || item.ReleaseDate != "2016-11-04" {
		t.Fatalf("douban rexxar identity metadata was not applied: %+v", item)
	}
	if !strings.Contains(item.Overview, "斯特兰奇博士") || len(item.Genres) != 3 || len(item.Countries) != 1 {
		t.Fatalf("douban rexxar detail metadata was not applied: %+v", item)
	}
	if len(item.Directors) != 1 || item.Directors[0] != "斯科特·德瑞克森" || len(item.Actors) != 2 || len(item.Tags) != 2 {
		t.Fatalf("douban people/tag metadata was not applied: %+v", item)
	}
	if item.PosterPath == "" {
		t.Fatalf("expected poster to be cached locally: %+v", item)
	}
	poster, contentType, _, err := service.Poster(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(poster) != 4 || contentType != "image/jpeg" {
		t.Fatalf("unexpected cached poster: %q len=%d", contentType, len(poster))
	}
}

func TestServiceDoesNotCacheHTMLAsPoster(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<script>document.cookie='bot=1'</script>"))
	}))
	defer server.Close()

	service, err := NewServiceWithStateDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.cacheRemotePoster(ctx, "poster-test", server.URL+"/poster.jpg"); err == nil {
		t.Fatal("expected html poster response to be rejected")
	}
}

func TestBotChallengeCookieParsesDoubanImageChallenge(t *testing.T) {
	body := `<script>function a(a){function n(){t=a[_0x649a("0x7")](t,2710831104)}var e={WTKkN:1149163402,bOYDu:71521412,dtzqS:function(a,n){return a+n},wyeCN:170440136};document.cookie="__tst_status="+a(0)+"#;",document.cookie=a(1)+";";</script>EO_Bot_Ssid`
	cookie, ok := botChallengeCookie(body)
	if !ok {
		t.Fatal("expected challenge cookie")
	}
	if cookie != "__tst_status=1391124950#; EO_Bot_Ssid=2710831104" {
		t.Fatalf("unexpected cookie: %s", cookie)
	}
}

func TestServiceScrapeFallsBackToDoubanAbstract(t *testing.T) {
	ctx := context.Background()
	mediaDir := t.TempDir()
	videoPath := filepath.Join(mediaDir, "搜索.2012.1080p.mkv")
	if err := os.WriteFile(videoPath, []byte("movie"), 0o600); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/j/subject_suggest":
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			fmt.Fprint(w, `[{"id":"6799191","title":"搜索","year":"2012","img":"","type":"movie"}]`)
		case "/rexxar/api/v2/movie/6799191":
			http.NotFound(w, r)
		case "/j/subject_abstract":
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			fmt.Fprint(w, `{"r":0,"subject":{"title":"搜索 Caught in the Web‎ (2012)","release_year":"2012","rate":"7.4","types":["剧情","悬疑"],"region":"中国大陆","directors":["陈凯歌"],"actors":["高圆圆","姚晨","赵又廷"]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("HIGO_VIDEO_DOUBAN_SUGGEST_URL", server.URL+"/j/subject_suggest")
	t.Setenv("HIGO_VIDEO_DOUBAN_REXXAR_BASE_URL", server.URL+"/rexxar/api/v2/movie/")
	t.Setenv("HIGO_VIDEO_DOUBAN_ABSTRACT_URL", server.URL+"/j/subject_abstract")
	t.Setenv("HIGO_VIDEO_DOUBAN_SUBJECT_BASE_URL", server.URL+"/subject/")

	service := NewService()
	if _, err := service.CreateLibrary(ctx, CreateLibraryRequest{Name: "电影", Type: LibraryMovie, Paths: []string{mediaDir}}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	items, err := service.Items(ctx, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateScrapeTask(ctx, CreateTaskRequest{ItemID: items[0].ID}); err != nil {
		t.Fatal(err)
	}
	item, err := service.Item(ctx, items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if item.Title != "搜索 Caught in the Web" || item.Rating != "7.4" || len(item.Genres) != 2 || len(item.Actors) != 3 {
		t.Fatalf("douban abstract fallback was not applied: %+v", item)
	}
}

func TestServiceScrapeUsesDoubanIDFromFilename(t *testing.T) {
	ctx := context.Background()
	mediaDir := t.TempDir()
	videoPath := filepath.Join(mediaDir, "随便命名.2016.[doubanid-3025375].mkv")
	if err := os.WriteFile(videoPath, []byte("movie"), 0o600); err != nil {
		t.Fatal(err)
	}

	suggestCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/j/subject_suggest":
			suggestCalled = true
			http.NotFound(w, r)
		case "/rexxar/api/v2/movie/3025375":
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			fmt.Fprint(w, `{"id":"3025375","title":"奇异博士","original_title":"Doctor Strange","year":"2016","intro":"直接通过豆瓣 ID 识别。","rating":{"value":7.6}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("HIGO_VIDEO_DOUBAN_SUGGEST_URL", server.URL+"/j/subject_suggest")
	t.Setenv("HIGO_VIDEO_DOUBAN_REXXAR_BASE_URL", server.URL+"/rexxar/api/v2/movie/")
	t.Setenv("HIGO_VIDEO_DOUBAN_SUBJECT_BASE_URL", server.URL+"/subject/")

	service := NewService()
	if _, err := service.CreateLibrary(ctx, CreateLibraryRequest{Name: "电影", Type: LibraryMovie, Paths: []string{mediaDir}}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	items, err := service.Items(ctx, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ProviderID != "3025375" || strings.Contains(items[0].Title, "doubanid") {
		t.Fatalf("expected provider id from filename and clean title, got %+v", items)
	}
	if _, err := service.CreateScrapeTask(ctx, CreateTaskRequest{ItemID: items[0].ID}); err != nil {
		t.Fatal(err)
	}
	item, err := service.Item(ctx, items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if suggestCalled {
		t.Fatal("expected direct douban id lookup without suggest search")
	}
	if item.Title != "奇异博士" || item.OriginalTitle != "Doctor Strange" || item.Rating != "7.6" {
		t.Fatalf("douban id metadata was not applied: %+v", item)
	}
}

func serverURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func TestTitleAndYearRemovesChineseReleaseTags(t *testing.T) {
	title, year := titleAndYear("奇异博士.2016.1080p.中英双语双字.mkv")
	if title != "奇异博士" || year != "2016" {
		t.Fatalf("unexpected cleaned title/year: %q %q", title, year)
	}
}

func TestServiceParsesLiveSource(t *testing.T) {
	ctx := context.Background()
	playlist := filepath.Join(t.TempDir(), "live.m3u")
	if err := os.WriteFile(playlist, []byte(`#EXTM3U
#EXTINF:-1 tvg-id="news-1" tvg-logo="https://example.test/logo.png" group-title="新闻",综合频道
http://example.test/live/1.m3u8
#EXTINF:-1 group-title="电影",电影频道
http://example.test/live/2.m3u8
`), 0o600); err != nil {
		t.Fatal(err)
	}

	service := NewService()
	source, err := service.CreateLiveSource(ctx, CreateLiveSourceRequest{Name: "测试直播", URL: playlist})
	if err != nil {
		t.Fatal(err)
	}
	if source.ChannelCount != 2 {
		t.Fatalf("expected 2 channels, got %d", source.ChannelCount)
	}
	channels, err := service.LiveChannels(ctx, source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(channels) != 2 || channels[0].Name != "综合频道" || channels[0].Group != "新闻" {
		t.Fatalf("unexpected channels: %+v", channels)
	}
	if channels[0].GuideID != "news-1" {
		t.Fatalf("expected tvg-id to be preserved, got %+v", channels[0])
	}
}

func TestServiceImportsXMLTVGuideAndCreatesRecordingTimer(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	playlist := filepath.Join(dir, "live.m3u")
	if err := os.WriteFile(playlist, []byte(`#EXTM3U
#EXTINF:-1 tvg-id="cctv-news" tvg-logo="https://example.test/news.png" group-title="新闻",新闻频道
http://example.test/live/news.m3u8
`), 0o600); err != nil {
		t.Fatal(err)
	}
	xmltv := filepath.Join(dir, "guide.xml")
	start := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	stop := start.Add(30 * time.Minute)
	if err := os.WriteFile(xmltv, []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<tv>
  <channel id="cctv-news"><display-name>新闻频道</display-name></channel>
  <programme start="%s +0000" stop="%s +0000" channel="cctv-news">
    <title lang="zh">晚间新闻</title>
    <desc lang="zh">当天新闻摘要。</desc>
    <category lang="zh">新闻</category>
  </programme>
</tv>`, start.Format("20060102150405"), stop.Format("20060102150405"))), 0o600); err != nil {
		t.Fatal(err)
	}

	service, err := NewServiceWithStateDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	source, err := service.CreateLiveSource(ctx, CreateLiveSourceRequest{Name: "IPTV", URL: playlist})
	if err != nil {
		t.Fatal(err)
	}
	guide, err := service.CreateGuideSource(ctx, CreateGuideSourceRequest{Name: "XMLTV", URL: xmltv})
	if err != nil {
		t.Fatal(err)
	}
	if guide.ProgramCount != 1 || guide.Status != "已加载" {
		t.Fatalf("unexpected guide source: %+v", guide)
	}
	programs, err := service.LivePrograms(ctx, LiveProgramQuery{SourceID: source.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(programs) != 1 {
		t.Fatalf("expected one program, got %+v", programs)
	}
	program := programs[0]
	if program.ChannelID == "" || program.ChannelName != "新闻频道" || program.Title != "晚间新闻" {
		t.Fatalf("unexpected mapped program: %+v", program)
	}
	if program.StartAt != start.Format(time.RFC3339) || program.EndAt != stop.Format(time.RFC3339) {
		t.Fatalf("unexpected normalized time: %+v", program)
	}

	settings, err := service.UpdateDVRSettings(ctx, DVRSettings{
		RecordingPath:       filepath.Join(dir, "recordings"),
		PrePaddingSeconds:   60,
		PostPaddingSeconds:  120,
		MaxConcurrentRecord: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if settings.PrePaddingSeconds != 60 || settings.PostPaddingSeconds != 120 || settings.MaxConcurrentRecord != 2 {
		t.Fatalf("unexpected dvr settings: %+v", settings)
	}
	timer, err := service.CreateRecordingTimer(ctx, CreateRecordingTimerRequest{ProgramID: program.ID})
	if err != nil {
		t.Fatal(err)
	}
	if timer.ProgramID != program.ID || timer.Status != RecordingScheduled {
		t.Fatalf("unexpected recording timer: %+v", timer)
	}
	if timer.StartAt != start.Add(-time.Minute).Format(time.RFC3339) || timer.EndAt != stop.Add(2*time.Minute).Format(time.RFC3339) {
		t.Fatalf("expected padded timer times, got %+v", timer)
	}

	reloaded, err := NewServiceWithStateDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	reloadedTimers, err := reloaded.RecordingTimers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloadedTimers) != 1 || reloadedTimers[0].ID != timer.ID {
		t.Fatalf("expected persisted timer, got %+v", reloadedTimers)
	}
}

func TestServiceRejectsEndedRecordingTimer(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	service, err := NewServiceWithStateDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	playlist := filepath.Join(dir, "live.m3u")
	if err := os.WriteFile(playlist, []byte(`#EXTM3U
#EXTINF:-1 tvg-id="c1",测试频道
http://example.test/live.m3u8
`), 0o600); err != nil {
		t.Fatal(err)
	}
	source, err := service.CreateLiveSource(ctx, CreateLiveSourceRequest{Name: "IPTV", URL: playlist})
	if err != nil || source.ChannelCount != 1 {
		t.Fatalf("expected live source, got %+v err=%v", source, err)
	}
	channels, err := service.LiveChannels(ctx, source.ID)
	if err != nil || len(channels) != 1 {
		t.Fatalf("expected channel, got %+v err=%v", channels, err)
	}
	_, err = service.CreateRecordingTimer(ctx, CreateRecordingTimerRequest{
		ChannelID: channels[0].ID,
		Name:      "已结束节目",
		StartAt:   time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
		EndAt:     time.Now().UTC().Add(-time.Hour).Format(time.RFC3339),
	})
	if err == nil || !strings.Contains(err.Error(), "节目已结束") {
		t.Fatalf("expected ended program error, got %v", err)
	}
}

func TestServiceMapsXMLTVProgramsByTvgNameWhenTvgIDDiffers(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	playlist := filepath.Join(dir, "live.m3u")
	playlistContent := "#EXTM3U\n" +
		"#EXTINF:-1 tvg-id=\"1\" tvg-name=\"CCTV1\" group-title=\"央视频道\",\ufeffCCTV1\n" +
		"http://example.test/live/cctv1.m3u8\n"
	if err := os.WriteFile(playlist, []byte(playlistContent), 0o600); err != nil {
		t.Fatal(err)
	}
	xmltv := filepath.Join(dir, "guide.xml")
	if err := os.WriteFile(xmltv, []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tv>
  <channel id="CCTV1"><display-name>CCTV1</display-name></channel>
  <programme start="20260604080000 +0800" stop="20260604083000 +0800" channel="CCTV1">
    <title lang="zh">朝闻天下</title>
  </programme>
</tv>`), 0o600); err != nil {
		t.Fatal(err)
	}

	service := NewService()
	source, err := service.CreateLiveSource(ctx, CreateLiveSourceRequest{Name: "IPTV", URL: playlist})
	if err != nil {
		t.Fatal(err)
	}
	guide, err := service.CreateGuideSource(ctx, CreateGuideSourceRequest{Name: "节目指南", URL: xmltv})
	if err != nil {
		t.Fatal(err)
	}
	if guide.ProgramCount != 1 {
		t.Fatalf("expected mapped program count, got %+v", guide)
	}
	programs, err := service.LivePrograms(ctx, LiveProgramQuery{SourceID: source.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(programs) != 1 || programs[0].ChannelName != "CCTV1" || programs[0].Title != "朝闻天下" {
		t.Fatalf("unexpected mapped programs: %+v", programs)
	}
}
