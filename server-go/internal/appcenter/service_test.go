package appcenter

import (
	"context"
	"testing"

	hdocker "higoos/server-go/internal/docker"
)

// fakeDocker records the container operations the app center drives.
type fakeDocker struct {
	created []hdocker.CreateContainerRequest
	started []string
	stopped []string
	removed []string
	failNew bool
}

func (f *fakeDocker) CreateContainer(ctx context.Context, req hdocker.CreateContainerRequest) (hdocker.Container, error) {
	if f.failNew {
		return hdocker.Container{}, context.DeadlineExceeded
	}
	f.created = append(f.created, req)
	return hdocker.Container{ID: "ctr-" + req.Name, Image: req.Image}, nil
}

func (f *fakeDocker) Start(ctx context.Context, id string) (hdocker.Container, error) {
	f.started = append(f.started, id)
	return hdocker.Container{ID: id}, nil
}

func (f *fakeDocker) Stop(ctx context.Context, id string) (hdocker.Container, error) {
	f.stopped = append(f.stopped, id)
	return hdocker.Container{ID: id}, nil
}

func (f *fakeDocker) RemoveContainer(ctx context.Context, id string, req hdocker.RemoveContainerRequest) error {
	f.removed = append(f.removed, id)
	return nil
}

func TestInstallCreatesAndStartsRealContainer(t *testing.T) {
	svc := NewService()
	fake := &fakeDocker{}
	svc.AttachDocker(fake)
	ctx := context.Background()

	app, err := svc.Install(ctx, "paperless")
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if app.ContainerID == "" {
		t.Fatalf("install should record a container id, got %#v", app)
	}
	if len(fake.created) != 1 || fake.created[0].Image != "ghcr.io/paperless-ngx/paperless-ngx:latest" {
		t.Fatalf("expected paperless image created, got %#v", fake.created)
	}
	if fake.created[0].Name != "paperless" {
		t.Fatalf("expected container named after app id, got %q", fake.created[0].Name)
	}
	if len(fake.started) != 1 {
		t.Fatalf("expected container started, got %#v", fake.started)
	}
	if !app.Running || app.Status != "运行中" {
		t.Fatalf("app should be running after install: %#v", app)
	}
}

func TestStopStopsRealContainer(t *testing.T) {
	svc := NewService()
	fake := &fakeDocker{}
	svc.AttachDocker(fake)
	ctx := context.Background()

	installed, err := svc.Install(ctx, "paperless")
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	stopped, err := svc.Stop(ctx, "paperless")
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if stopped.Running {
		t.Fatalf("app should not be running after stop: %#v", stopped)
	}
	if len(fake.stopped) != 1 || fake.stopped[0] != installed.ContainerID {
		t.Fatalf("expected the app's container to be stopped, got %#v", fake.stopped)
	}
}

func TestInstallReusesExistingContainer(t *testing.T) {
	svc := NewService()
	fake := &fakeDocker{}
	svc.AttachDocker(fake)
	ctx := context.Background()

	if _, err := svc.Install(ctx, "paperless"); err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, err := svc.Stop(ctx, "paperless"); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if _, err := svc.Start(ctx, "paperless"); err != nil {
		t.Fatalf("start: %v", err)
	}
	// Should not create a second container for the same app.
	if len(fake.created) != 1 {
		t.Fatalf("expected exactly one container created across install+start, got %d", len(fake.created))
	}
	if len(fake.started) != 2 {
		t.Fatalf("expected two start calls (install + start), got %d", len(fake.started))
	}
}

func TestInstallDegradesGracefullyOnDockerError(t *testing.T) {
	svc := NewService()
	svc.AttachDocker(&fakeDocker{failNew: true})
	app, err := svc.Install(context.Background(), "paperless")
	if err != nil {
		t.Fatalf("install should not hard-error on docker failure: %v", err)
	}
	// Optimistic fallback: app stays installed/running but no container is bound
	// and the status records the degradation.
	if !app.Installed || !app.Running {
		t.Fatalf("app should stay optimistically installed/running: %#v", app)
	}
	if app.ContainerID != "" {
		t.Fatalf("no container should be bound on failure, got %q", app.ContainerID)
	}
	if app.Status == "运行中" {
		t.Fatalf("status should record the degradation, got %q", app.Status)
	}
}

func TestInstallWithoutDockerKeepsLegacyBehaviour(t *testing.T) {
	svc := NewService() // no docker attached
	app, err := svc.Install(context.Background(), "paperless")
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !app.Running || app.ContainerID != "" {
		t.Fatalf("without docker, install should flip running state and not set a container id: %#v", app)
	}
}
