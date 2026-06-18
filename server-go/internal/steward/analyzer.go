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
	Name      string
	Path      string
	Type      string
	Space     string
	SizeBytes int64
	Modified  time.Time
}

const (
	largeFileBytes = 100 << 20             // 100 MiB
	staleAfter     = 365 * 24 * time.Hour  // ~1 year untouched
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

func duplicateSuggestion(files []FileInfo) (Suggestion, bool) {
	byName := map[string][]FileInfo{}
	for _, f := range files {
		key := strings.ToLower(strings.TrimSpace(f.Name))
		if key == "" {
			continue
		}
		byName[key] = append(byName[key], f)
	}
	dupFiles, dupGroups := 0, 0
	var examples []string
	for name, group := range byName {
		if len(group) < 2 {
			continue
		}
		dupGroups++
		dupFiles += len(group)
		if len(examples) < 3 {
			examples = append(examples, name)
		}
	}
	if dupGroups == 0 {
		return Suggestion{}, false
	}
	sort.Strings(examples)
	return Suggestion{
		ID:     "dup-files",
		Title:  "清理重复文件",
		Detail: fmt.Sprintf("发现 %d 组同名文件（共 %d 个），如 %s，可保留最新版本并归档其余。", dupGroups, dupFiles, strings.Join(examples, "、")),
		Count:  fmt.Sprintf("%d 个", dupFiles),
		Risk:   RiskMedium,
		Action: "review-duplicates",
		Status: SuggestionPending,
	}, true
}

func archiveSuggestion(files []FileInfo, now time.Time) (Suggestion, bool) {
	count := 0
	var bytes int64
	for _, f := range files {
		stale := !f.Modified.IsZero() && now.Sub(f.Modified) > staleAfter
		if f.SizeBytes >= largeFileBytes || stale {
			count++
			bytes += f.SizeBytes
		}
	}
	if count == 0 {
		return Suggestion{}, false
	}
	return Suggestion{
		ID:     "archive-candidates",
		Title:  "归档大文件与陈旧文件",
		Detail: fmt.Sprintf("发现 %d 个超大或一年未访问的文件（约 %s），可移动到归档空间释放容量。", count, humanBytes(bytes)),
		Count:  fmt.Sprintf("%d 个", count),
		Risk:   RiskLow,
		Action: "archive-cold-data",
		Status: SuggestionPending,
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
