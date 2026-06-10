package downloads

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNewServiceStartsWithEmptyQueueAndSpeedProfiles(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	if tasks := service.ListTasks(ctx); len(tasks) != 0 {
		t.Fatalf("tasks = %d, want empty real queue", len(tasks))
	}

	profiles := service.SpeedProfiles(ctx)
	if len(profiles) != 3 {
		t.Fatalf("speed profiles = %d, want 3", len(profiles))
	}
	if profiles[0].Name != "智能限速" || !profiles[0].Active {
		t.Fatalf("default active speed profile mismatch: %#v", profiles[0])
	}
}

func TestCreateTaskInfersSourceCategoryAndHandling(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	tests := []struct {
		name         string
		request      CreateTaskRequest
		wantSource   SourceType
		wantCategory string
		wantHandling string
	}{
		{
			name:         "bt torrent",
			request:      CreateTaskRequest{Link: "https://nas.local/releases/family-movie.torrent", Name: "家庭电影合集"},
			wantSource:   SourceBT,
			wantCategory: "影视",
			wantHandling: "完成后可归档到 /Media/Movies",
		},
		{
			name:         "http package",
			request:      CreateTaskRequest{Link: "https://releases.ubuntu.com/ubuntu-26.04-live-server.iso", Category: "软件"},
			wantSource:   SourceHTTP,
			wantCategory: "软件",
			wantHandling: "完成后可校验 SHA256",
		},
		{
			name:         "magnet",
			request:      CreateTaskRequest{Link: "magnet:?xt=urn:btih:higo-family-media"},
			wantSource:   SourceMagnet,
			wantCategory: "影视",
			wantHandling: "完成后可归档到 /Media/TV",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := service.CreateTask(ctx, tt.request)
			if err != nil {
				t.Fatalf("create task: %v", err)
			}
			if task.Source != tt.wantSource || task.Category != tt.wantCategory {
				t.Fatalf("source/category = %q/%q, want %q/%q", task.Source, task.Category, tt.wantSource, tt.wantCategory)
			}
			if task.Status != StatusQueued || task.Progress != 0 || task.Speed != "排队中" {
				t.Fatalf("unexpected initial queue state: %#v", task)
			}
			if task.Handling != tt.wantHandling {
				t.Fatalf("handling = %q, want %q", task.Handling, tt.wantHandling)
			}
		})
	}
}

func TestHTTPDownloadWritesFileAndPersistsTask(t *testing.T) {
	payload := []byte("hello higo downloads\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="sample.txt"`)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	stateDir := t.TempDir()
	downloadDir := filepath.Join(stateDir, "downloads")
	service, err := NewServiceWithStateDirAndDownloadDir(stateDir, downloadDir)
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	task, err := service.CreateTask(context.Background(), CreateTaskRequest{
		Source:   SourceHTTP,
		Link:     server.URL + "/sample.txt",
		Category: "文档",
	})
	if err != nil {
		t.Fatalf("create http task: %v", err)
	}

	var completed DownloadTask
	waitFor(t, func() bool {
		for _, item := range service.ListTasks(context.Background()) {
			if item.ID == task.ID && item.Status == StatusCompleted {
				completed = item
				return true
			}
		}
		return false
	})
	if completed.Progress != 100 || completed.FilePath == "" {
		t.Fatalf("completed task missing file state: %#v", completed)
	}
	data, err := os.ReadFile(completed.FilePath)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(data) != string(payload) {
		t.Fatalf("downloaded data = %q, want %q", data, payload)
	}

	reloaded, err := NewServiceWithStateDirAndDownloadDir(stateDir, downloadDir)
	if err != nil {
		t.Fatalf("reload service: %v", err)
	}
	if tasks := reloaded.ListTasks(context.Background()); len(tasks) != 1 || tasks[0].FilePath != completed.FilePath {
		t.Fatalf("persisted tasks mismatch: %#v", tasks)
	}
}

func TestHTTPDownloadUsesRangeWhenServerSupportsRanges(t *testing.T) {
	payload := []byte("range capable mirror payload")
	var seenRange string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Disposition", `attachment; filename="mirror.iso"`)
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		switch r.Method {
		case http.MethodHead:
			return
		case http.MethodGet:
			seenRange = r.Header.Get("Range")
			if seenRange != "bytes=0-" {
				http.Error(w, "range required", http.StatusForbidden)
				return
			}
			w.Header().Set("Content-Range", "bytes 0-"+strconv.Itoa(len(payload)-1)+"/"+strconv.Itoa(len(payload)))
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	stateDir := t.TempDir()
	service, err := NewServiceWithStateDirAndDownloadDir(stateDir, filepath.Join(stateDir, "downloads"))
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	task, err := service.CreateTask(context.Background(), CreateTaskRequest{
		Source:   SourceHTTP,
		Link:     server.URL + "/ubuntu.iso",
		Category: "软件",
	})
	if err != nil {
		t.Fatalf("create http task: %v", err)
	}

	var completed DownloadTask
	waitFor(t, func() bool {
		for _, item := range service.ListTasks(context.Background()) {
			if item.ID == task.ID && item.Status == StatusCompleted {
				completed = item
				return true
			}
		}
		return false
	})
	if seenRange != "bytes=0-" {
		t.Fatalf("GET Range = %q, want bytes=0-", seenRange)
	}
	data, err := os.ReadFile(completed.FilePath)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(data) != string(payload) {
		t.Fatalf("downloaded data = %q, want %q", data, payload)
	}
}

func TestPauseResumeAndArchiveTask(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	task, err := service.CreateTask(ctx, CreateTaskRequest{Link: "magnet:?xt=urn:btih:higo-family-media"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	paused, err := service.PauseTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("pause: %v", err)
	}
	if paused.Status != StatusPaused || paused.Speed != "0 KB/s" {
		t.Fatalf("paused task mismatch: %#v", paused)
	}

	resumed, err := service.ResumeTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if resumed.Status != StatusQueued || resumed.Speed != "排队中" {
		t.Fatalf("resumed task mismatch: %#v", resumed)
	}

	result, err := service.ArchiveTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	if result.Task.Status != StatusCompleted || !result.Task.Archived || result.Task.Progress != 100 {
		t.Fatalf("archived task mismatch: %#v", result.Task)
	}
	if !strings.Contains(result.Message, "已归档") || result.FilePath == "" {
		t.Fatalf("archive result missing file manager linkage: %#v", result)
	}
}

func TestDeleteTaskCanRemoveUnfinishedDownloadFiles(t *testing.T) {
	stateDir := t.TempDir()
	downloadDir := filepath.Join(stateDir, "downloads")
	service, err := NewServiceWithStateDirAndDownloadDir(stateDir, downloadDir)
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	task, err := service.CreateTask(context.Background(), CreateTaskRequest{
		Source:   SourceHTTP,
		Link:     "https://example.com/ubuntu.iso",
		Category: "软件",
	})
	if err != nil {
		t.Fatalf("create http task: %v", err)
	}
	finalPath := filepath.Join(downloadDir, "软件", "ubuntu.iso")
	partPath := finalPath + ".part"
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(partPath, []byte("partial"), 0o644); err != nil {
		t.Fatalf("write part: %v", err)
	}
	service.updateTask(task.ID, func(t *DownloadTask) {
		t.Status = StatusPaused
		t.FilePath = finalPath
	})

	result, err := service.DeleteTask(context.Background(), task.ID, DeleteTaskOptions{DeleteFile: true})
	if err != nil {
		t.Fatalf("delete task: %v", err)
	}
	if !strings.Contains(result.Message, "已删除任务和文件") {
		t.Fatalf("delete message = %q, want file deletion message", result.Message)
	}
	if _, err := os.Stat(partPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("part file still exists or unexpected stat error: %v", err)
	}
	if tasks := service.ListTasks(context.Background()); len(tasks) != 0 {
		t.Fatalf("task should be removed, got %#v", tasks)
	}
}

func TestDeleteTaskCanRemoveAria2TaskDirectory(t *testing.T) {
	stateDir := t.TempDir()
	downloadDir := filepath.Join(stateDir, "downloads")
	service, err := NewServiceWithStateDirAndDownloadDir(stateDir, downloadDir)
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	task := DownloadTask{
		ID:       12,
		Name:     "wired-cd.torrent",
		Source:   SourceBT,
		Category: "影视",
		Status:   StatusPaused,
	}
	task.FilePath = service.aria2TaskDir(task)
	service.tasks = []DownloadTask{task}
	service.nextID = 13
	if err := os.MkdirAll(filepath.Join(task.FilePath, "content"), 0o755); err != nil {
		t.Fatalf("mkdir aria2 dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(task.FilePath, "content", "piece.bin"), []byte("partial"), 0o644); err != nil {
		t.Fatalf("write aria2 partial: %v", err)
	}

	result, err := service.DeleteTask(context.Background(), task.ID, DeleteTaskOptions{DeleteFile: true})
	if err != nil {
		t.Fatalf("delete task: %v", err)
	}
	if !strings.Contains(result.Message, "已删除任务和文件") {
		t.Fatalf("delete message = %q, want file deletion message", result.Message)
	}
	if _, err := os.Stat(task.FilePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("aria2 dir still exists or unexpected stat error: %v", err)
	}
}

func TestUpdateActiveSpeedProfileUpdatesRunningTasks(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	task, err := service.CreateTask(ctx, CreateTaskRequest{Link: "https://example.com/file.txt", Source: SourceHTTP, Category: "文档"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	profile, err := service.UpdateActiveSpeedProfile(ctx, "家庭优先")
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if !profile.Active || profile.DownloadLimit != "6 MB/s" {
		t.Fatalf("active profile mismatch: %#v", profile)
	}

	var found DownloadTask
	for _, item := range service.ListTasks(ctx) {
		if item.ID == task.ID {
			found = item
			break
		}
	}
	if found.ID == 0 {
		t.Fatalf("task not found")
	}
	if found.Status == StatusRunning && found.Speed != "6 MB/s" {
		t.Fatalf("running speed not updated: %#v", found)
	}
}

func TestRSSSubscriptionParsesDownloadableEntry(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/feed.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0"?><rss><channel><item><title>测试附件</title><enclosure url="` + "http://" + r.Host + `/file.txt"/></item></channel></rss>`))
	})
	mux.HandleFunc("/file.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("rss item"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	service, err := NewServiceWithStateDirAndDownloadDir(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	task, err := service.CreateTask(context.Background(), CreateTaskRequest{
		Source: SourceRSS,
		Link:   server.URL + "/feed.xml",
	})
	if err != nil {
		t.Fatalf("create rss: %v", err)
	}

	waitFor(t, func() bool {
		for _, item := range service.ListTasks(context.Background()) {
			if item.ID == task.ID && item.Status == StatusCompleted && strings.Contains(item.Handling, "已添加订阅下载") {
				return true
			}
		}
		return false
	})
	if tasks := service.ListTasks(context.Background()); len(tasks) != 2 {
		t.Fatalf("rss should enqueue child download, got %#v", tasks)
	}
}

func waitFor(t *testing.T, done func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if done() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("condition was not met before timeout")
}
