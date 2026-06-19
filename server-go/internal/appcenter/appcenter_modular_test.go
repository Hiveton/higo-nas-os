package appcenter

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestManifestValidateRejectsBadInput(t *testing.T) {
	cases := map[string]AppManifest{
		"bad schema":   {SchemaVersion: "9", ID: "x", Name: "X", Version: "1", Risk: "low", Containers: []ContainerSpec{{Name: "c", Image: "i"}}},
		"bad id":       {SchemaVersion: "1", ID: "Bad ID", Name: "X", Version: "1", Risk: "low", Containers: []ContainerSpec{{Name: "c", Image: "i"}}},
		"no container": {SchemaVersion: "1", ID: "x", Name: "X", Version: "1", Risk: "low"},
		"bad risk":     {SchemaVersion: "1", ID: "x", Name: "X", Version: "1", Risk: "scary", Containers: []ContainerSpec{{Name: "c", Image: "i"}}},
		"bad webentry": {SchemaVersion: "1", ID: "x", Name: "X", Version: "1", Risk: "low", Containers: []ContainerSpec{{Name: "c", Image: "i"}}, WebEntry: &WebEntry{Port: 80, Container: "missing"}},
	}
	for name, m := range cases {
		if err := m.Validate(); err == nil {
			t.Errorf("%s: expected validation error, got nil", name)
		}
	}
	good := AppManifest{SchemaVersion: "1", ID: "ok-app", Name: "OK", Version: "1.0", Risk: "low", Containers: []ContainerSpec{{Name: "c", Image: "img"}}}
	if err := good.Validate(); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
}

func TestCatalogBuiltinAndLocalPrecedence(t *testing.T) {
	dir := t.TempDir()
	cat := NewCatalog(dir, nil)
	if _, ok := cat.Lookup("paperless"); !ok {
		t.Fatalf("expected builtin paperless in catalog")
	}
	builtinCount := len(cat.Entries())
	if builtinCount < 4 {
		t.Fatalf("expected >=4 builtin entries, got %d", builtinCount)
	}

	// Drop a local package that overrides paperless and adds a new app.
	writeLocalManifest(t, dir, "paperless", `{"schemaVersion":"1","id":"paperless","name":"Paperless LOCAL","version":"9.9","category":"x","risk":"low","containers":[{"name":"c","image":"img"}]}`)
	writeLocalManifest(t, dir, "hello", `{"schemaVersion":"1","id":"hello","name":"Hello","version":"1.0","category":"demo","risk":"low","containers":[{"name":"web","image":"nginx"}],"webEntry":{"port":80}}`)
	cat.ScanLocal()

	entry, ok := cat.Lookup("paperless")
	if !ok || entry.Origin != OriginLocal || entry.Manifest.Name != "Paperless LOCAL" {
		t.Fatalf("local package should override builtin: %#v", entry)
	}
	if _, ok := cat.Lookup("hello"); !ok {
		t.Fatalf("expected local-only app 'hello' in catalog")
	}
}

func TestBuildContainerRequestsMapsEverything(t *testing.T) {
	m := AppManifest{
		SchemaVersion: "1", ID: "demo", Name: "Demo", Version: "1", Risk: "low",
		Containers: []ContainerSpec{{
			Name:          "web",
			Image:         "nginx:latest",
			Ports:         []PortMapping{{Container: 80, Host: 8080, Protocol: "tcp"}},
			Volumes:       []VolumeMount{{Name: "data", Path: "/data"}},
			Env:           []EnvVar{{Key: "TOKEN", Value: "{{config.token}}"}, {Key: "STATIC", Value: "x"}},
			Resources:     Resources{CPU: 2, MemoryMB: 512},
			RestartPolicy: "unless-stopped",
		}},
	}
	reqs := buildContainerRequests(m, map[string]string{"token": "secret123"})
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	req := reqs[0]
	if req.Name != "demo" || req.Image != "nginx:latest" {
		t.Fatalf("unexpected name/image: %#v", req)
	}
	if len(req.Ports) != 1 || req.Ports[0] != "8080:80/tcp" {
		t.Fatalf("port mapping wrong: %#v", req.Ports)
	}
	if len(req.Mounts) != 1 || req.Mounts[0] != "demo_data:/data" {
		t.Fatalf("mount wrong: %#v", req.Mounts)
	}
	if req.LimitCPU != 2 || req.LimitMemory != 512 {
		t.Fatalf("resource limits wrong: cpu=%d mem=%d", req.LimitCPU, req.LimitMemory)
	}
	foundInterp := false
	for _, e := range req.Env {
		if e == "TOKEN=secret123" {
			foundInterp = true
		}
	}
	if !foundInterp {
		t.Fatalf("config interpolation failed: %#v", req.Env)
	}
}

func TestGovernancePreviewConfirmRollback(t *testing.T) {
	svc := NewService()
	fake := &fakeDocker{}
	svc.AttachDocker(fake)
	ctx := context.Background()

	preview, err := svc.PreviewAction(ctx, "paperless", ActionInstall, map[string]string{"secretKey": "abc"})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.ConfirmationID == "" || !preview.RequiresConfirmation || preview.Impact == "" {
		t.Fatalf("bad preview: %#v", preview)
	}

	// Wrong confirmation id is rejected with no side effect.
	if _, err := svc.ConfirmAction(ctx, "paperless", ActionInstall, "bogus", "tester"); err == nil {
		t.Fatalf("expected rejection of bad confirmation id")
	}
	if app, _ := svc.Get("paperless"); app.Installed {
		t.Fatalf("paperless should not be installed after a rejected confirm")
	}

	res, err := svc.ConfirmAction(ctx, "paperless", ActionInstall, preview.ConfirmationID, "tester")
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if !res.App.Installed || !res.App.Running || res.AuditID == "" {
		t.Fatalf("install confirm did not take effect: %#v", res)
	}
	if len(fake.created) != 1 {
		t.Fatalf("expected one container created on confirmed install, got %d", len(fake.created))
	}

	log, _ := svc.AuditLog(ctx)
	if len(log) != 1 || log[0].Action != ActionInstall || !log[0].Rollbackable {
		t.Fatalf("unexpected audit log: %#v", log)
	}

	// Rollback the install -> app returns to installable, container removed.
	if _, err := svc.Rollback(ctx, res.AuditID, "tester"); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if app, _ := svc.Get("paperless"); app.Installed {
		t.Fatalf("rollback should uninstall paperless: %#v", app)
	}
	if len(fake.removed) == 0 {
		t.Fatalf("rollback should remove the container")
	}
}

func TestUninstallClearsContainersAndState(t *testing.T) {
	svc := NewService()
	fake := &fakeDocker{}
	svc.AttachDocker(fake)
	ctx := context.Background()

	if _, err := svc.Install(ctx, "paperless"); err != nil {
		t.Fatalf("install: %v", err)
	}
	app, err := svc.Uninstall(ctx, "paperless", false)
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if app.Installed || app.Running || app.ContainerID != "" || app.Status != "可安装" {
		t.Fatalf("uninstall did not reset state: %#v", app)
	}
	if len(fake.removed) != 1 {
		t.Fatalf("expected the container to be removed on uninstall, got %#v", fake.removed)
	}
}

func TestExampleManifestIsValid(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "examples", "appcenter", "hello-app", "manifest.json"))
	if err != nil {
		t.Fatalf("read example manifest: %v", err)
	}
	m, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("shipped example manifest is invalid: %v", err)
	}
	if m.ID != "hello-app" || m.WebEntry == nil || m.WebEntry.Container != "web" {
		t.Fatalf("unexpected example manifest: %#v", m)
	}
}

func writeLocalManifest(t *testing.T, stateDir, id, body string) {
	t.Helper()
	dir := filepath.Join(stateDir, "appcenter", "apps", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}
