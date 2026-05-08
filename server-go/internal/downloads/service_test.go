package downloads

import (
	"context"
	"strings"
	"testing"
)

func TestNewServiceSeedsFrontendTasksAndSpeedProfiles(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	tasks := service.ListTasks(ctx)
	if len(tasks) != 4 {
		t.Fatalf("seeded tasks = %d, want 4", len(tasks))
	}
	if tasks[0].Name != "纪录片合集 S02" || tasks[0].Source != SourceBT {
		t.Fatalf("first task does not match frontend seed: %#v", tasks[0])
	}
	if tasks[3].Source != SourceRSS || tasks[3].Status != StatusPaused || tasks[3].Speed != "等待 RSS" {
		t.Fatalf("rss seed does not match frontend seed: %#v", tasks[3])
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
		wantStatus   TaskStatus
		wantProgress int
		wantHandling string
	}{
		{
			name:         "bt torrent",
			request:      CreateTaskRequest{Link: "https://nas.local/releases/family-movie.torrent", Name: "家庭电影合集"},
			wantSource:   SourceBT,
			wantCategory: "影视",
			wantStatus:   StatusRunning,
			wantProgress: 3,
			wantHandling: "完成后自动归档到 /Media/Movies",
		},
		{
			name:         "http package",
			request:      CreateTaskRequest{Link: "https://releases.ubuntu.com/ubuntu-26.04-live-server.iso", Category: "软件"},
			wantSource:   SourceHTTP,
			wantCategory: "软件",
			wantStatus:   StatusRunning,
			wantProgress: 3,
			wantHandling: "完成后校验 SHA256",
		},
		{
			name:         "magnet",
			request:      CreateTaskRequest{Link: "magnet:?xt=urn:btih:higo-family-media"},
			wantSource:   SourceMagnet,
			wantCategory: "影视",
			wantStatus:   StatusRunning,
			wantProgress: 3,
			wantHandling: "完成后刮削海报并归档到 /Media/TV",
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
			if task.Status != tt.wantStatus || task.Progress != tt.wantProgress || task.Speed != "排队中" {
				t.Fatalf("unexpected initial queue state: %#v", task)
			}
			if task.Handling != tt.wantHandling {
				t.Fatalf("handling = %q, want %q", task.Handling, tt.wantHandling)
			}
		})
	}
}

func TestPauseAndResumeTask(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	paused, err := service.PauseTask(ctx, 1)
	if err != nil {
		t.Fatalf("pause: %v", err)
	}
	if paused.Status != StatusPaused || paused.Speed != "0 KB/s" {
		t.Fatalf("paused task mismatch: %#v", paused)
	}

	resumed, err := service.ResumeTask(ctx, 1)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if resumed.Status != StatusRunning || resumed.Speed != "18 MB/s" {
		t.Fatalf("resumed task should use active profile speed: %#v", resumed)
	}
}

func TestArchiveTaskMarksCompletedAndReturnsFileManagerMessage(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	result, err := service.ArchiveTask(ctx, 1)
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	if result.Task.Status != StatusCompleted || !result.Task.Archived || result.Task.Progress != 100 {
		t.Fatalf("archived task mismatch: %#v", result.Task)
	}
	if result.Task.Handling != "已归档到文件管家 /影视" {
		t.Fatalf("archive handling mismatch: %q", result.Task.Handling)
	}
	if !strings.Contains(result.Message, "文件管家已自动归档：纪录片合集 S02") || result.FilePath != "/Media/TV" {
		t.Fatalf("archive result missing file manager linkage: %#v", result)
	}
}

func TestDeleteTaskRemovesArchivedRecord(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	if _, err := service.ArchiveTask(ctx, 2); err != nil {
		t.Fatalf("archive completed seed: %v", err)
	}
	result, err := service.DeleteTask(ctx, 2)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if result.Task.ID != 2 || !strings.Contains(result.Message, "已清理 1 条已归档记录") {
		t.Fatalf("delete result mismatch: %#v", result)
	}
	for _, task := range service.ListTasks(ctx) {
		if task.ID == 2 {
			t.Fatalf("deleted task still present: %#v", task)
		}
	}
}

func TestUpdateActiveSpeedProfileUpdatesRunningTasks(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	profile, err := service.UpdateActiveSpeedProfile(ctx, "家庭优先")
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if !profile.Active || profile.DownloadLimit != "6 MB/s" {
		t.Fatalf("active profile mismatch: %#v", profile)
	}

	tasks := service.ListTasks(ctx)
	if tasks[0].Speed != "6 MB/s" || tasks[1].Speed != "0 KB/s" || tasks[3].Speed != "等待 RSS" {
		t.Fatalf("speed update touched wrong tasks: %#v", tasks)
	}
}

func TestCreateRSSSubscriptionStartsPausedAndUsesSubscriptionArchive(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	task, err := service.CreateTask(ctx, CreateTaskRequest{
		Source: SourceRSS,
		Link:   "https://example.com/courses.xml",
	})
	if err != nil {
		t.Fatalf("create rss: %v", err)
	}
	if task.Source != SourceRSS || task.Category != "订阅" || task.Status != StatusPaused {
		t.Fatalf("rss task state mismatch: %#v", task)
	}
	if task.Size != "等待解析" || task.Speed != "等待订阅" || task.Progress != 0 {
		t.Fatalf("rss task queue metadata mismatch: %#v", task)
	}
	if task.ArchiveRule.TargetPath != "/Downloads/Subscriptions" || task.Handling != "新条目自动下载到 /Downloads/Courses" {
		t.Fatalf("rss archive behavior mismatch: %#v", task)
	}
}
