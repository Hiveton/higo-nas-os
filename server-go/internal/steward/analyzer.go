package steward

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// FileInfo is the minimal file fact the analyzer reasons over. The httpapi layer
// builds these from the files service tree.
type FileInfo struct {
	ID        string
	Name      string
	Path      string
	Type      string
	Space     string
	SizeBytes int64
	Modified  time.Time
}

const (
	largeFileBytes = 100 << 20            // 100 MiB
	staleAfter     = 365 * 24 * time.Hour // ~1 year untouched
	// archiveSpace is the display name of the space archive candidates move into
	// (the backup-archive directory). Files already there are skipped.
	archiveSpace = "备份归档"
)

// Analyze derives real housekeeping suggestions from a file snapshot:
// duplicate names, and large/stale archive candidates. Returned suggestions are
// pending; empty when nothing actionable is found.
func Analyze(files []FileInfo, now time.Time) []Suggestion {
	var out []Suggestion
	if s, ok := duplicateSuggestion(files); ok {
		out = append(out, s)
	}
	if s, ok := archiveSuggestion(files, now); ok {
		out = append(out, s)
	}
	return out
}

// duplicateSuggestion groups files by name AND size (so two unrelated files that
// merely share a name aren't flagged), keeps the newest in each group, and
// proposes deleting the rest to the recycle bin.
func duplicateSuggestion(files []FileInfo) (Suggestion, bool) {
	type dupKey struct {
		name string
		size int64
	}
	byKey := map[dupKey][]FileInfo{}
	for _, f := range files {
		name := strings.ToLower(strings.TrimSpace(f.Name))
		if name == "" || f.ID == "" {
			continue
		}
		key := dupKey{name: name, size: f.SizeBytes}
		byKey[key] = append(byKey[key], f)
	}

	var ops []SuggestionOp
	dupGroups := 0
	var examples []string
	for key, group := range byKey {
		if len(group) < 2 {
			continue
		}
		dupGroups++
		// Keep the newest (tie-break by path for determinism); delete the rest.
		sort.Slice(group, func(i, j int) bool {
			if group[i].Modified.Equal(group[j].Modified) {
				return group[i].Path < group[j].Path
			}
			return group[i].Modified.After(group[j].Modified)
		})
		for _, dup := range group[1:] {
			ops = append(ops, SuggestionOp{Type: "delete", FileID: dup.ID})
		}
		if len(examples) < 3 {
			examples = append(examples, key.name)
		}
	}
	if len(ops) == 0 {
		return Suggestion{}, false
	}
	sort.Strings(examples)
	return Suggestion{
		ID:         "dup-files",
		Title:      "清理重复文件",
		Detail:     fmt.Sprintf("发现 %d 组同名同大小文件，如 %s，可保留最新版本、删除其余 %d 个到回收站（可还原）。", dupGroups, strings.Join(examples, "、"), len(ops)),
		Count:      fmt.Sprintf("%d 个", len(ops)),
		Risk:       RiskMedium,
		Action:     "预览清理",
		Status:     SuggestionPending,
		Operations: ops,
	}, true
}

// archiveSuggestion proposes moving large (≥100MiB) or long-untouched (≥1y)
// files that aren't already archived into the 备份归档 space.
func archiveSuggestion(files []FileInfo, now time.Time) (Suggestion, bool) {
	var ops []SuggestionOp
	var bytes int64
	for _, f := range files {
		if f.ID == "" || f.Space == archiveSpace {
			continue
		}
		stale := !f.Modified.IsZero() && now.Sub(f.Modified) > staleAfter
		if f.SizeBytes >= largeFileBytes || stale {
			ops = append(ops, SuggestionOp{Type: "move", FileID: f.ID, Dest: archiveSpace})
			bytes += f.SizeBytes
		}
	}
	if len(ops) == 0 {
		return Suggestion{}, false
	}
	return Suggestion{
		ID:         "archive-candidates",
		Title:      "归档大文件与陈旧文件",
		Detail:     fmt.Sprintf("发现 %d 个超大或一年未访问的文件（约 %s），可移动到「%s」空间释放容量（可移回）。", len(ops), humanBytes(bytes), archiveSpace),
		Count:      fmt.Sprintf("%d 个", len(ops)),
		Risk:       RiskLow,
		Action:     "预览归档",
		Status:     SuggestionPending,
		Operations: ops,
	}, true
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
