package aianalysis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

// FaceTrainingSample is one labeled face vector captured during analysis. The
// clustering layer records a sample per detected face; when the user names a
// cluster the matching samples are confirmed. Confirmed samples are the ground
// truth a self-training run consumes.
type FaceTrainingSample struct {
	ID         string    `json:"id"`
	ClusterID  string    `json:"clusterId"`
	PersonName string    `json:"personName,omitempty"`
	MediaKey   string    `json:"mediaKey,omitempty"`
	Embedding  []float32 `json:"embedding"`
	Confirmed  bool      `json:"confirmed"`
	CreatedAt  time.Time `json:"createdAt"`
}

// FaceModelVersion is one entry in the trained-model registry. The active
// version is the one a configured embedder is expected to be serving.
type FaceModelVersion struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	SampleCount int       `json:"sampleCount"`
	Source      string    `json:"source"`
	Note        string    `json:"note,omitempty"`
	Active      bool      `json:"active"`
}

// FaceTrainingStats summarises the dataset and registry for the API.
type FaceTrainingStats struct {
	TotalSamples     int    `json:"totalSamples"`
	ConfirmedSamples int    `json:"confirmedSamples"`
	NamedPeople      int    `json:"namedPeople"`
	Models           int    `json:"models"`
	ActiveModel      string `json:"activeModel,omitempty"`
}

type faceTrainingSnapshot struct {
	Seq     int                  `json:"seq"`
	Samples []FaceTrainingSample `json:"samples"`
	Models  []FaceModelVersion   `json:"models"`
}

// FaceTrainingStore is the persistent data layer of the self-training framework:
// it captures labeled samples, records user confirmations, exports the dataset,
// and registers trained model versions. JSON-backed (single writer = higo-api).
type FaceTrainingStore struct {
	mu      sync.Mutex
	path    string
	now     func() time.Time
	seq     int
	samples []FaceTrainingSample
	models  []FaceModelVersion
}

func newFaceTrainingStore(path string, now func() time.Time) (*FaceTrainingStore, error) {
	s := &FaceTrainingStore{path: path, now: now}
	if path != "" {
		var snap faceTrainingSnapshot
		if err := state.LoadJSON(path, &snap); err != nil {
			return nil, err
		}
		s.seq = snap.Seq
		s.samples = snap.Samples
		s.models = snap.Models
	}
	return s, nil
}

// record appends an (initially unconfirmed) training sample for a clustered face.
func (s *FaceTrainingStore) record(clusterID, mediaKey string, embedding []float32) {
	if s == nil || len(embedding) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	s.samples = append(s.samples, FaceTrainingSample{
		ID:        fmt.Sprintf("fs-%d", s.seq),
		ClusterID: clusterID,
		MediaKey:  mediaKey,
		Embedding: append([]float32(nil), embedding...),
		CreatedAt: s.now(),
	})
	_ = s.saveLocked()
}

// Confirm names a cluster: all samples of that cluster are marked confirmed and
// tagged with the person name. Returns how many samples were confirmed.
func (s *FaceTrainingStore) Confirm(clusterID, name string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for i := range s.samples {
		if s.samples[i].ClusterID == clusterID {
			s.samples[i].PersonName = name
			s.samples[i].Confirmed = true
			n++
		}
	}
	if n > 0 {
		_ = s.saveLocked()
	}
	return n
}

// Export returns a copy of the dataset; confirmedOnly restricts it to
// user-confirmed (ground-truth) samples.
func (s *FaceTrainingStore) Export(confirmedOnly bool) []FaceTrainingSample {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]FaceTrainingSample, 0, len(s.samples))
	for _, sample := range s.samples {
		if confirmedOnly && !sample.Confirmed {
			continue
		}
		out = append(out, sample)
	}
	return out
}

// RegisterModel records a trained model version and makes it the active one.
func (s *FaceTrainingStore) RegisterModel(version FaceModelVersion) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.models {
		s.models[i].Active = false
	}
	version.Active = true
	if version.CreatedAt.IsZero() {
		version.CreatedAt = s.now()
	}
	s.models = append([]FaceModelVersion{version}, s.models...)
	_ = s.saveLocked()
}

func (s *FaceTrainingStore) Models() []FaceModelVersion {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]FaceModelVersion(nil), s.models...)
}

func (s *FaceTrainingStore) Stats() FaceTrainingStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := FaceTrainingStats{TotalSamples: len(s.samples), Models: len(s.models)}
	names := map[string]bool{}
	for _, sample := range s.samples {
		if sample.Confirmed {
			st.ConfirmedSamples++
			if sample.PersonName != "" {
				names[sample.PersonName] = true
			}
		}
	}
	st.NamedPeople = len(names)
	for _, m := range s.models {
		if m.Active {
			st.ActiveModel = m.ID
			break
		}
	}
	return st
}

func (s *FaceTrainingStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	return state.SaveJSON(s.path, faceTrainingSnapshot{
		Seq:     s.seq,
		Samples: append([]FaceTrainingSample(nil), s.samples...),
		Models:  append([]FaceModelVersion(nil), s.models...),
	})
}

// FaceTrainer triggers a (re)training run from the exported dataset. The default
// is a no-op (the framework is reserved but inert); a configured sidecar runs a
// real training job and returns the resulting model version.
type FaceTrainer interface {
	Name() string
	Available() bool
	Train(ctx context.Context, samples []FaceTrainingSample) (FaceModelVersion, error)
}

// NewFaceTrainer returns a remote trainer when url is set, else a no-op.
func NewFaceTrainer(url string) FaceTrainer {
	if strings.TrimSpace(url) == "" {
		return noopFaceTrainer{}
	}
	return &remoteFaceTrainer{
		url:    strings.TrimRight(strings.TrimSpace(url), "/"),
		client: &http.Client{Timeout: 30 * time.Minute},
	}
}

type noopFaceTrainer struct{}

func (noopFaceTrainer) Name() string   { return "none" }
func (noopFaceTrainer) Available() bool { return false }
func (noopFaceTrainer) Train(context.Context, []FaceTrainingSample) (FaceModelVersion, error) {
	return FaceModelVersion{}, fmt.Errorf("aianalysis: face trainer not configured (set HIGO_FACE_TRAINER_URL)")
}

// remoteFaceTrainer posts the dataset to a training sidecar:
//
//	POST <url>
//	  request : {"samples": [FaceTrainingSample...]}
//	  response: {"id": "...", "sampleCount": N, "source": "...", "note": "..."}
type remoteFaceTrainer struct {
	url    string
	client *http.Client
}

func (r *remoteFaceTrainer) Name() string   { return "remote:" + r.url }
func (r *remoteFaceTrainer) Available() bool { return r.url != "" }

func (r *remoteFaceTrainer) Train(ctx context.Context, samples []FaceTrainingSample) (FaceModelVersion, error) {
	buf, err := json.Marshal(map[string]any{"samples": samples})
	if err != nil {
		return FaceModelVersion{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.url, bytes.NewReader(buf))
	if err != nil {
		return FaceModelVersion{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		return FaceModelVersion{}, fmt.Errorf("aianalysis: face trainer: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return FaceModelVersion{}, fmt.Errorf("aianalysis: face trainer returned %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var version FaceModelVersion
	if err := json.Unmarshal(raw, &version); err != nil {
		return FaceModelVersion{}, fmt.Errorf("aianalysis: decode face trainer response: %w", err)
	}
	if version.ID == "" {
		version.ID = fmt.Sprintf("model-%d", len(samples))
	}
	version.Source = r.Name()
	version.SampleCount = len(samples)
	return version, nil
}
