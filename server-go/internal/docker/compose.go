package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// composeProjectDir returns the on-disk directory where a stack's
// docker-compose.yaml is persisted (alongside the docker state file).
func (s *DevService) composeProjectDir(name string) string {
	base := filepath.Dir(s.statePath)
	if s.statePath == "" {
		base = filepath.Join(os.TempDir(), "higoos-docker")
	}
	return filepath.Join(base, "compose", safeStackName(name))
}

var stackNamePattern = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)

func safeStackName(name string) string {
	clean := stackNamePattern.ReplaceAllString(strings.TrimSpace(name), "-")
	clean = strings.Trim(clean, "-")
	if clean == "" {
		return "stack"
	}
	return clean
}

// countComposeServices counts the top-level entries under the `services:` block
// of a compose YAML — enough for the UI summary without a full YAML parser.
func countComposeServices(yaml string) int {
	lines := strings.Split(yaml, "\n")
	inServices := false
	indent := -1
	count := 0
	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		lead := len(line) - len(strings.TrimLeft(line, " "))
		if !inServices {
			if trimmed == "services:" || strings.HasPrefix(trimmed, "services:") {
				inServices = true
			}
			continue
		}
		// A non-indented key ends the services block.
		if lead == 0 {
			break
		}
		if indent == -1 {
			indent = lead
		}
		if lead == indent && strings.HasSuffix(trimmed, ":") {
			count++
		}
	}
	return count
}

// DeployStack writes the compose YAML to disk and (when the docker CLI is
// available) runs `docker compose up -d`. The stack record is upserted either
// way so the dev environment reflects the deployment without a real engine.
func (s *DevService) DeployStack(ctx context.Context, req DeployStackRequest) (ComposeStack, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return ComposeStack{}, fmt.Errorf("stack name is required")
	}
	if strings.TrimSpace(req.Yaml) == "" {
		return ComposeStack{}, fmt.Errorf("compose yaml is required")
	}

	dir := s.composeProjectDir(name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ComposeStack{}, fmt.Errorf("create compose dir: %w", err)
	}
	yamlPath := filepath.Join(dir, "docker-compose.yaml")
	if err := os.WriteFile(yamlPath, []byte(req.Yaml), 0o644); err != nil {
		return ComposeStack{}, fmt.Errorf("write compose yaml: %w", err)
	}

	status := "已部署"
	if dockerCLIAvailable() {
		cmd := exec.CommandContext(ctx, "docker", "compose", "-f", yamlPath, "-p", safeStackName(name), "up", "-d")
		if out, err := cmd.CombinedOutput(); err != nil {
			return ComposeStack{}, fmt.Errorf("docker compose up failed: %v: %s", err, strings.TrimSpace(string(out)))
		}
		status = "运行中"
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	stack := ComposeStack{
		Name:      name,
		Status:    status,
		Services:  countComposeServices(req.Yaml),
		Volume:    dir,
		Network:   safeStackName(name) + "-default",
		Yaml:      req.Yaml,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	replaced := false
	for i := range s.stacks {
		if s.stacks[i].Name == name {
			stack.CreatedAt = firstNonEmpty(s.stacks[i].CreatedAt, stack.CreatedAt)
			s.stacks[i] = stack
			replaced = true
			break
		}
	}
	if !replaced {
		s.stacks = append(s.stacks, stack)
	}
	return stack, s.saveLocked()
}

// DownStack stops a stack (`docker compose down` on a real host) and marks the
// record stopped.
func (s *DevService) DownStack(ctx context.Context, name string) (ComposeStack, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ComposeStack{}, fmt.Errorf("stack name is required")
	}
	yamlPath := filepath.Join(s.composeProjectDir(name), "docker-compose.yaml")
	if dockerCLIAvailable() {
		if _, err := os.Stat(yamlPath); err == nil {
			cmd := exec.CommandContext(ctx, "docker", "compose", "-f", yamlPath, "-p", safeStackName(name), "down")
			if out, err := cmd.CombinedOutput(); err != nil {
				return ComposeStack{}, fmt.Errorf("docker compose down failed: %v: %s", err, strings.TrimSpace(string(out)))
			}
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.stacks {
		if s.stacks[i].Name == name {
			s.stacks[i].Status = "已停止"
			updated := s.stacks[i]
			return updated, s.saveLocked()
		}
	}
	return ComposeStack{}, fmt.Errorf("stack not found: %s", name)
}

// StackYaml returns the stored compose YAML for a stack (from the record, or
// from disk as a fallback).
func (s *DevService) StackYaml(ctx context.Context, name string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	name = strings.TrimSpace(name)
	s.mu.RLock()
	for i := range s.stacks {
		if s.stacks[i].Name == name && s.stacks[i].Yaml != "" {
			y := s.stacks[i].Yaml
			s.mu.RUnlock()
			return y, nil
		}
	}
	s.mu.RUnlock()
	yamlPath := filepath.Join(s.composeProjectDir(name), "docker-compose.yaml")
	if data, err := os.ReadFile(yamlPath); err == nil {
		return string(data), nil
	}
	return "", fmt.Errorf("no compose yaml on record for stack: %s", name)
}
