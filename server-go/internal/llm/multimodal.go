package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sort"
	"strings"
)

// openAIBase returns the provider base URL with the OpenAI default.
func openAIBase(p Provider) string {
	if p.BaseURL != "" {
		return p.BaseURL
	}
	return "https://api.openai.com/v1"
}

// postJSON issues a non-streaming JSON POST with bearer auth and decodes the
// response into out.
func postJSON(ctx context.Context, url, apiKey string, body, out any) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	return doDecode(req, out)
}

func doDecode(req *http.Request, out any) error {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("call provider: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("provider returned %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// Embed returns one embedding vector per input string using the provider's
// OpenAI-compatible /embeddings endpoint.
func Embed(ctx context.Context, p Provider, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	var resp struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	payload := map[string]any{"model": p.Model, "input": inputs}
	if err := postJSON(ctx, openAIBase(p)+"/embeddings", p.APIKey, payload, &resp); err != nil {
		return nil, err
	}
	if len(resp.Data) != len(inputs) {
		return nil, fmt.Errorf("embedding count mismatch: got %d want %d", len(resp.Data), len(inputs))
	}
	sort.Slice(resp.Data, func(i, j int) bool { return resp.Data[i].Index < resp.Data[j].Index })
	out := make([][]float32, len(resp.Data))
	for i := range resp.Data {
		out[i] = resp.Data[i].Embedding
	}
	return out, nil
}

// Caption describes an image using a vision-capable chat model (OpenAI-style
// image_url content). prompt steers the description (e.g. "用中文列出场景、物体、事件标签").
func Caption(ctx context.Context, p Provider, image []byte, mimeType, prompt string) (string, error) {
	if mimeType == "" {
		mimeType = "image/jpeg"
	}
	dataURL := "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(image)
	payload := map[string]any{
		"model":      p.Model,
		"max_tokens": 512,
		"messages": []map[string]any{{
			"role": "user",
			"content": []map[string]any{
				{"type": "text", "text": prompt},
				{"type": "image_url", "image_url": map[string]any{"url": dataURL}},
			},
		}},
	}
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := postJSON(ctx, openAIBase(p)+"/chat/completions", p.APIKey, payload, &resp); err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("vision provider returned no choices")
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// Transcribe converts audio to text via the provider's OpenAI-compatible
// /audio/transcriptions endpoint (multipart upload).
func Transcribe(ctx context.Context, p Provider, filename string, audio []byte) (string, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(audio); err != nil {
		return "", err
	}
	_ = w.WriteField("model", p.Model)
	_ = w.WriteField("response_format", "text")
	w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIBase(p)+"/audio/transcriptions", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call provider: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("provider returned %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	// response_format=text returns plain text; some servers still return JSON.
	text := strings.TrimSpace(string(raw))
	if strings.HasPrefix(text, "{") {
		var j struct {
			Text string `json:"text"`
		}
		if json.Unmarshal(raw, &j) == nil && j.Text != "" {
			return strings.TrimSpace(j.Text), nil
		}
	}
	return text, nil
}
