package appcenter

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	hdocker "higoos/server-go/internal/docker"
)

var configRefPattern = regexp.MustCompile(`\{\{\s*config\.([a-zA-Z0-9_]+)\s*\}\}`)

// interpolate resolves {{config.KEY}} references in a manifest value against the
// user-supplied config collected at install time. Unknown keys resolve to "".
func interpolate(value string, config map[string]string) string {
	return configRefPattern.ReplaceAllStringFunc(value, func(match string) string {
		key := configRefPattern.FindStringSubmatch(match)[1]
		return config[key]
	})
}

// containerName derives the docker container name for a spec. Single-container
// apps keep the app id (backward compatible with the old seed behavior); multi
// container apps are namespaced as "<appId>-<spec>".
func containerName(m AppManifest, spec ContainerSpec) string {
	if len(m.Containers) == 1 {
		return m.ID
	}
	return m.ID + "-" + spec.Name
}

// buildContainerRequests translates a manifest's container specs into docker
// create requests, fully mapping volumes, env (with config interpolation),
// resource limits and restart policy — the fields the previous install pipeline
// silently dropped.
func buildContainerRequests(m AppManifest, config map[string]string) []hdocker.CreateContainerRequest {
	reqs := make([]hdocker.CreateContainerRequest, 0, len(m.Containers))
	for _, c := range m.Containers {
		reqs = append(reqs, hdocker.CreateContainerRequest{
			Image:         c.Image,
			Name:          containerName(m, c),
			Ports:         specPortStrings(c.Ports),
			Mounts:        specMounts(m.ID, c.Volumes),
			Env:           specEnv(c.Env, config),
			Command:       interpolate(c.Command, config),
			RestartPolicy: c.RestartPolicy,
			LimitCPU:      c.Resources.CPU,
			LimitMemory:   c.Resources.MemoryMB,
			Labels:        []string{"higoos.app=" + m.ID},
		})
	}
	return reqs
}

func specPortStrings(ports []PortMapping) []string {
	out := make([]string, 0, len(ports))
	for _, p := range ports {
		out = append(out, formatPort(p))
	}
	return out
}

// specMounts maps named volumes to docker mount strings, namespacing the volume
// by app id so two apps can't collide on a generic name like "data".
func specMounts(appID string, vols []VolumeMount) []string {
	out := make([]string, 0, len(vols))
	for _, v := range vols {
		out = append(out, fmt.Sprintf("%s_%s:%s", appID, v.Name, v.Path))
	}
	return out
}

func specEnv(env []EnvVar, config map[string]string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		out = append(out, e.Key+"="+interpolate(e.Value, config))
	}
	return out
}

// orchestrateRunning creates (if needed) and starts every container the manifest
// declares, recording the bound container ids on the app. Without a docker
// runner (or with an empty image) it is a no-op and the app keeps its optimistic
// in-memory running state. On docker failure it degrades gracefully rather than
// leaving the UI in a broken half-state.
func (s *Service) orchestrateRunning(ctx context.Context, id string, m AppManifest, config map[string]string) (App, error) {
	if s.docker == nil || m.primaryImage() == "" {
		return s.snapshot(id)
	}
	bound := s.containersFor(id)
	for _, req := range buildContainerRequests(m, config) {
		cid := bound[req.Name]
		if cid == "" {
			container, err := s.docker.CreateContainer(ctx, req)
			if err != nil {
				return s.degrade(ctx, id, "容器创建失败")
			}
			cid = container.ID
		}
		if _, err := s.docker.Start(ctx, cid); err != nil {
			return s.degrade(ctx, id, "容器启动失败")
		}
		bound[req.Name] = cid
	}
	return s.update(ctx, id, func(a *persistedApp) {
		a.Containers = bound
		a.ContainerID = primaryContainerID(m, bound)
		a.Running = true
		a.Status = "运行中"
	})
}

// stopContainers stops every running container bound to the app.
func (s *Service) stopContainers(ctx context.Context, id string) error {
	if s.docker == nil {
		return nil
	}
	for _, cid := range s.containersFor(id) {
		if cid == "" {
			continue
		}
		if _, err := s.docker.Stop(ctx, cid); err != nil {
			return err
		}
	}
	return nil
}

// removeContainers force-removes every container bound to the app and clears the
// binding. Used by uninstall and by update (recreate with the new image).
func (s *Service) removeContainers(ctx context.Context, id string, removeVolumes bool) {
	if s.docker == nil {
		return
	}
	for _, cid := range s.containersFor(id) {
		if cid == "" {
			continue
		}
		_ = s.docker.RemoveContainer(ctx, cid, hdocker.RemoveContainerRequest{Force: true, RemoveVolumes: removeVolumes})
	}
}

// degrade keeps the app's optimistic installed/running state when container
// orchestration is unavailable, recording the degradation in the status. This
// matches the platform's dev-fallback norm.
func (s *Service) degrade(ctx context.Context, id, reason string) (App, error) {
	return s.update(ctx, id, func(a *persistedApp) {
		a.Running = true
		a.Status = "运行中（" + reason + "，已降级为本地状态）"
	})
}

func primaryContainerID(m AppManifest, bound map[string]string) string {
	if m.WebEntry != nil && m.WebEntry.Container != "" {
		for _, c := range m.Containers {
			if c.Name == m.WebEntry.Container {
				return bound[containerName(m, c)]
			}
		}
	}
	if len(m.Containers) > 0 {
		return bound[containerName(m, m.Containers[0])]
	}
	return ""
}

// hostFromImage is a tiny helper kept for readability in impact summaries.
func imagesOf(m AppManifest) string {
	imgs := make([]string, 0, len(m.Containers))
	for _, c := range m.Containers {
		imgs = append(imgs, c.Image)
	}
	return strings.Join(imgs, ", ")
}
