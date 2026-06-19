package aianalysis

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// FaceVector is one detected face: its bounding box, detector confidence, and a
// (typically L2-normalized) recognition embedding suitable for cosine
// clustering. This is the unit the self-training framework standardizes on, so
// any model — an off-the-shelf ArcFace/SFace ONNX model or a self-trained one —
// plugs in by producing FaceVectors.
type FaceVector struct {
	Box       [4]float32 `json:"box"` // x, y, w, h (pixels or normalized; model-defined)
	Score     float32    `json:"score"`
	Embedding []float32  `json:"embedding"`
}

// FaceEmbedder detects faces in an image and returns one embedding per face. It
// is the pluggable inference seam: the default is a no-op (falling back to the
// vision-descriptor approximation); a configured sidecar serves a real model.
type FaceEmbedder interface {
	Name() string
	Available() bool
	DetectAndEmbed(ctx context.Context, image []byte, mime string) ([]FaceVector, error)
}

// NewFaceEmbedder returns a remote embedder when url is set, else a no-op.
func NewFaceEmbedder(url string) FaceEmbedder {
	if strings.TrimSpace(url) == "" {
		return noopFaceEmbedder{}
	}
	return &remoteFaceEmbedder{
		url:    strings.TrimRight(strings.TrimSpace(url), "/"),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// noopFaceEmbedder is the default when no face model sidecar is configured.
type noopFaceEmbedder struct{}

func (noopFaceEmbedder) Name() string    { return "none" }
func (noopFaceEmbedder) Available() bool  { return false }
func (noopFaceEmbedder) DetectAndEmbed(context.Context, []byte, string) ([]FaceVector, error) {
	return nil, nil
}

// remoteFaceEmbedder calls a face detection+embedding sidecar over HTTP. The
// contract is intentionally model-agnostic so x86_64 and arm64 sidecars (ONNX
// Runtime, or a self-trained server) implement the same surface:
//
//	POST <url>
//	  request : {"image": "<base64>", "mime": "image/jpeg"}
//	  response: {"faces": [{"box":[x,y,w,h], "score":0.99, "embedding":[...]}]}
type remoteFaceEmbedder struct {
	url    string
	client *http.Client
}

func (r *remoteFaceEmbedder) Name() string   { return "remote:" + r.url }
func (r *remoteFaceEmbedder) Available() bool { return r.url != "" }

func (r *remoteFaceEmbedder) DetectAndEmbed(ctx context.Context, image []byte, mime string) ([]FaceVector, error) {
	if mime == "" {
		mime = "image/jpeg"
	}
	payload := map[string]any{
		"image": base64.StdEncoding.EncodeToString(image),
		"mime":  mime,
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("aianalysis: face embedder: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("aianalysis: face embedder returned %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out struct {
		Faces []FaceVector `json:"faces"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("aianalysis: decode face embedder response: %w", err)
	}
	return out.Faces, nil
}
