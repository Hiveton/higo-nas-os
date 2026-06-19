package aianalysis

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"higoos/server-go/internal/index"
	"higoos/server-go/internal/llm"
	"higoos/server-go/internal/tasks"
)

const (
	maxAnalyzeFileBytes  = 2 << 20  // 2 MiB cap for text extraction
	maxAnalyzeImageBytes = 8 << 20  // 8 MiB cap for vision uploads
	maxSummaryInputRunes = 4000     // cap text fed to the summariser
)

const mediaVisionPrompt = "你是相册整理助手。请观察这张照片，用简体中文严格按以下格式输出，不要多余文字：\n" +
	"摘要: <一句话描述>\n标签: <逗号分隔的3-6个关键词>\n地点: <地点或场景，未知填 未知>\n设备: <拍摄设备线索，未知填 未知>\n人物: <逗号分隔的每个人的简短特征描述，无人填 无>"

const videoVisionPrompt = "你是视频整理助手。请观察这个视频关键帧，用简体中文严格按以下格式输出：\n摘要: <一句话描述>\n标签: <逗号分隔的3-6个关键词>"

// visionFields is the parsed structure of a vision caption response.
type visionFields struct {
	Summary string
	Tags    []string
	Place   string
	Device  string
	People  []string
}

// analyzeFile analyses a file: type/text (basic), LLM summary+tags (standard),
// vector index (deep).
func (e *Engine) analyzeFile(ctx context.Context, rec Record, level Level, h *tasks.Handle) (AnalyzerResult, error) {
	res := AnalyzerResult{}
	if ext := fileExt(rec.SourcePath); ext != "" {
		res.Tags = append(res.Tags, ext)
	}

	var text string
	if rec.SourcePath != "" && index.IsTextPath(rec.SourcePath) {
		if data, err := readCapped(rec.SourcePath, maxAnalyzeFileBytes); err == nil {
			if t, ok := index.ExtractText(rec.SourcePath, data); ok {
				text = t
			}
		}
	}

	if level.atLeast(LevelStandard) && strings.TrimSpace(text) != "" {
		if p, ok := e.chatProvider(); ok {
			h.Progress(45, "生成摘要与标签")
			e.ledgers[DomainFile].progress(rec.Key, 45)
			summary, tags, err := e.summarizeText(ctx, p, rec.Title, text)
			if err != nil {
				return res, err
			}
			res.Summary = summary
			res.Tags = mergeTags(res.Tags, tags)
		}
	}
	if res.Summary == "" {
		res.Summary = localSummary(rec.Title, text)
	}

	if level.atLeast(LevelDeep) && strings.TrimSpace(text) != "" && e.indexEnabled() {
		h.Progress(75, "建立语义索引")
		if _, err := e.index.IndexDocument(ctx, index.DocInput{
			SourceURI: rec.Key,
			Domain:    "file",
			Title:     rec.Title,
			Text:      text,
			Summary:   res.Summary,
			Tags:      res.Tags,
			Checksum:  rec.ContentSig,
		}); err == nil {
			res.Embedded = true
		}
	}
	return res, nil
}

// analyzeMedia analyses a photo/video item: metadata (basic), LLM tags
// (standard), vision caption + face clustering + vector index (deep).
func (e *Engine) analyzeMedia(ctx context.Context, rec Record, level Level, h *tasks.Handle) (AnalyzerResult, error) {
	res := AnalyzerResult{}
	if ext := fileExt(rec.SourcePath); ext != "" {
		res.Tags = append(res.Tags, strings.ToUpper(ext))
	}

	if level.atLeast(LevelStandard) {
		if p, ok := e.chatProvider(); ok {
			h.Progress(35, "生成标签")
			summary, tags, err := e.summarizeText(ctx, p, rec.Title, "相册照片标题: "+rec.Title)
			if err == nil {
				res.Summary = summary
				res.Tags = mergeTags(res.Tags, tags)
			}
		}
	}

	if level.atLeast(LevelDeep) {
		var img []byte
		var descriptors []string
		if isImagePath(rec.SourcePath) {
			img, _ = readCapped(rec.SourcePath, maxAnalyzeImageBytes)
		}

		// Vision captioning (scene/place/device/people descriptors).
		if vp, ok := e.visionProvider(); ok && len(img) > 0 {
			h.Progress(55, "视觉分析")
			e.ledgers[DomainMedia].progress(rec.Key, 55)
			caption, err := llm.Caption(ctx, vp, img, mimeOfPath(rec.SourcePath), mediaVisionPrompt)
			if err != nil {
				return res, err
			}
			fields := parseVisionFields(caption)
			res.Caption = caption
			if fields.Summary != "" {
				res.Summary = fields.Summary
			}
			if fields.Place != "" {
				res.Place = fields.Place
			}
			if fields.Device != "" {
				res.Device = fields.Device
			}
			res.People = fields.People
			descriptors = fields.People
			res.Tags = mergeTags(res.Tags, fields.Tags)
		}

		// Face clustering: prefer a real face model (the self-training seam);
		// otherwise fall back to clustering the vision descriptors.
		if e.faces != nil && len(img) > 0 && e.embedder != nil && e.embedder.Available() {
			h.Progress(70, "人脸识别")
			if vecs, err := e.embedder.DetectAndEmbed(ctx, img, mimeOfPath(rec.SourcePath)); err == nil && len(vecs) > 0 {
				if labels := e.faces.assignVectors(vecs, rec.Key, e.training); len(labels) > 0 {
					res.People = labels
				}
			}
		} else if e.faces != nil && len(descriptors) > 0 {
			if ep, ok := e.embeddingProvider(); ok {
				h.Progress(70, "人脸聚类")
				if labels, err := e.faces.assign(ctx, ep, descriptors); err == nil && len(labels) > 0 {
					res.People = labels
				}
			}
		}

		if e.indexEnabled() {
			h.Progress(85, "建立语义索引")
			if _, err := e.index.IndexDocument(ctx, index.DocInput{
				SourceURI: rec.Key,
				Domain:    "photo",
				Title:     rec.Title,
				Text:      mediaIndexText(rec, res),
				Summary:   res.Summary,
				Tags:      res.Tags,
				Checksum:  rec.ContentSig,
			}); err == nil {
				res.Embedded = true
			}
		}
	}
	return res, nil
}

// analyzeVideo analyses a video item: ffprobe metadata (basic), LLM overview
// (standard), keyframe vision + optional ASR + vector index (deep).
func (e *Engine) analyzeVideo(ctx context.Context, rec Record, level Level, h *tasks.Handle) (AnalyzerResult, error) {
	res := AnalyzerResult{TechMeta: map[string]any{}}

	h.Progress(20, "读取技术元数据")
	if probe, err := probeVideo(ctx, rec.SourcePath); err == nil {
		res.TechMeta["container"] = probe.Container
		res.TechMeta["codec"] = probe.Codec
		res.TechMeta["resolution"] = probe.Resolution
		res.TechMeta["durationSeconds"] = probe.DurationSeconds
		if probe.Resolution != "" {
			res.Tags = mergeTags(res.Tags, []string{probe.Resolution})
		}
	}

	if level.atLeast(LevelStandard) {
		if p, ok := e.chatProvider(); ok {
			h.Progress(40, "生成简介")
			summary, tags, err := e.summarizeText(ctx, p, rec.Title, "视频文件: "+rec.Title)
			if err == nil {
				res.Summary = summary
				res.Tags = mergeTags(res.Tags, tags)
			}
		}
	}

	if level.atLeast(LevelDeep) {
		if vp, ok := e.visionProvider(); ok {
			h.Progress(55, "关键帧分析")
			if frame, err := extractFrame(ctx, rec.SourcePath); err == nil && len(frame) > 0 {
				if caption, err := llm.Caption(ctx, vp, frame, "image/jpeg", videoVisionPrompt); err == nil {
					fields := parseVisionFields(caption)
					res.Caption = caption
					if res.Summary == "" && fields.Summary != "" {
						res.Summary = fields.Summary
					}
					res.Tags = mergeTags(res.Tags, fields.Tags)
				}
			}
		}
		if ap, ok := e.asrProvider(); ok {
			h.Progress(70, "语音转写")
			if wav, err := extractAudio(ctx, rec.SourcePath); err == nil && len(wav) > 0 {
				if txt, err := llm.Transcribe(ctx, ap, "audio.wav", wav); err == nil {
					res.Transcript = strings.TrimSpace(txt)
				}
			}
		}
		if e.indexEnabled() {
			text := strings.TrimSpace(res.Summary + "\n" + res.Caption + "\n" + res.Transcript)
			if text != "" {
				h.Progress(88, "建立语义索引")
				if _, err := e.index.IndexDocument(ctx, index.DocInput{
					SourceURI: rec.Key,
					Domain:    "video",
					Title:     rec.Title,
					Text:      text,
					Summary:   res.Summary,
					Tags:      res.Tags,
					Checksum:  rec.ContentSig,
				}); err == nil {
					res.Embedded = true
				}
			}
		}
	}
	if res.Summary == "" {
		res.Summary = localSummary(rec.Title, "")
	}
	return res, nil
}

// summarizeText asks a chat model for a one-line summary and tags.
func (e *Engine) summarizeText(ctx context.Context, p llm.Provider, title, text string) (string, []string, error) {
	client, err := llm.NewClient(p.Kind)
	if err != nil {
		return "", nil, err
	}
	prompt := "你是文件整理助手。请用简体中文为下面的内容生成概括，严格按以下格式输出，不要多余文字：\n" +
		"摘要: <一句话概括>\n标签: <逗号分隔的3-6个中文关键词>\n\n标题: " + title + "\n内容:\n" + capRunes(text, maxSummaryInputRunes)
	out, err := llm.Complete(ctx, client, p, llm.ChatRequest{
		Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
		MaxTokens:   300,
		Temperature: 0.2,
	})
	if err != nil {
		return "", nil, err
	}
	summary, tags := parseSummaryTags(out)
	if summary == "" {
		summary = localSummary(title, out)
	}
	return summary, tags, nil
}

func (e *Engine) chatProvider() (llm.Provider, bool)      { return e.providerFor(llm.PurposeChat) }
func (e *Engine) visionProvider() (llm.Provider, bool)    { return e.providerFor(llm.PurposeVision) }
func (e *Engine) embeddingProvider() (llm.Provider, bool) { return e.providerFor(llm.PurposeEmbedding) }
func (e *Engine) asrProvider() (llm.Provider, bool)       { return e.providerFor(llm.PurposeASR) }

func (e *Engine) providerFor(purpose llm.Purpose) (llm.Provider, bool) {
	if e.llm == nil {
		return llm.Provider{}, false
	}
	p, err := e.llm.DefaultFor(purpose)
	return p, err == nil
}

func (e *Engine) hasProvider(purpose llm.Purpose) bool {
	_, ok := e.providerFor(purpose)
	return ok
}

func (e *Engine) indexEnabled() bool { return e.index != nil && e.index.Enabled() }

// --- parsing / text helpers --------------------------------------------------

func parseVisionFields(caption string) visionFields {
	var f visionFields
	for _, line := range strings.Split(caption, "\n") {
		label, value, ok := splitLabel(line)
		if !ok {
			continue
		}
		switch label {
		case "摘要", "描述", "summary":
			f.Summary = value
		case "标签", "tags":
			f.Tags = splitList(value)
		case "地点", "场景", "place":
			f.Place = cleanUnknown(value)
		case "设备", "device":
			f.Device = cleanUnknown(value)
		case "人物", "people":
			f.People = filterPeople(splitList(value))
		}
	}
	if f.Summary == "" {
		f.Summary = firstLine(caption)
	}
	return f
}

func parseSummaryTags(out string) (string, []string) {
	var summary string
	var tags []string
	for _, line := range strings.Split(out, "\n") {
		label, value, ok := splitLabel(line)
		if !ok {
			continue
		}
		switch label {
		case "摘要", "summary":
			summary = value
		case "标签", "tags":
			tags = splitList(value)
		}
	}
	if summary == "" {
		summary = firstLine(out)
	}
	return summary, tags
}

// splitLabel splits "标签: a, b" into ("标签", "a, b"). Accepts both ASCII and
// fullwidth colons (the latter is a 3-byte rune, so we trim the colon set off the
// remainder rather than slicing by a fixed offset).
func splitLabel(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	line = strings.TrimLeft(line, "-*•· ")
	idx := strings.IndexAny(line, ":：")
	if idx <= 0 {
		return "", "", false
	}
	label := strings.ToLower(strings.TrimSpace(line[:idx]))
	value := strings.TrimSpace(strings.TrimLeft(line[idx:], ":："))
	return label, value, value != ""
}

func splitList(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '，' || r == '、' || r == ';' || r == '；' || r == '/'
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

func filterPeople(names []string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		if cleanUnknown(n) == "" {
			continue
		}
		out = append(out, n)
	}
	return out
}

func cleanUnknown(value string) string {
	v := strings.TrimSpace(value)
	switch v {
	case "未知", "无", "none", "n/a", "na", "unknown", "":
		return ""
	}
	return v
}

func mergeTags(existing, additions []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(existing)+len(additions))
	for _, t := range existing {
		key := strings.ToLower(strings.TrimSpace(t))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, strings.TrimSpace(t))
	}
	for _, t := range additions {
		t = strings.TrimSpace(t)
		key := strings.ToLower(t)
		if t == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, t)
	}
	return out
}

func mediaIndexText(rec Record, res AnalyzerResult) string {
	parts := []string{rec.Title, res.Summary, res.Caption, res.Place, res.Device}
	parts = append(parts, res.People...)
	parts = append(parts, res.Tags...)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, strings.TrimSpace(p))
		}
	}
	return strings.Join(out, " · ")
}

func localSummary(title, text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return strings.TrimSpace(title)
	}
	runes := []rune(text)
	if len(runes) > 120 {
		return strings.TrimSpace(string(runes[:120])) + "…"
	}
	return text
}

func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return strings.TrimSpace(s)
}

func capRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func readCapped(path string, max int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err == nil && info.Size() > max {
		buf := make([]byte, max)
		n, _ := f.Read(buf)
		return buf[:n], nil
	}
	data := make([]byte, 0, 64<<10)
	tmp := make([]byte, 32<<10)
	var total int64
	for {
		n, err := f.Read(tmp)
		if n > 0 {
			data = append(data, tmp[:n]...)
			total += int64(n)
			if total >= max {
				break
			}
		}
		if err != nil {
			break
		}
	}
	return data, nil
}

func fileExt(path string) string {
	return strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
}

func isImagePath(path string) bool {
	switch fileExt(path) {
	case "jpg", "jpeg", "png", "gif", "webp", "bmp", "heic", "heif", "tif", "tiff":
		return true
	default:
		return false
	}
}

func mimeOfPath(path string) string {
	switch fileExt(path) {
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "bmp":
		return "image/bmp"
	default:
		return "image/jpeg"
	}
}

// toInt coerces a TechMeta value (int in-process, float64 after a JSON round
// trip) to an int.
func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func toStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
