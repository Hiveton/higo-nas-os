package aianalysis

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"higoos/server-go/internal/llm"
	"higoos/server-go/internal/state"
)

// faceThreshold is the cosine similarity above which two textual person
// descriptors are considered the same cluster (the fallback approximation).
const faceThreshold = 0.86

// faceVectorThreshold is the cosine similarity for real face-recognition
// embeddings (ArcFace/SFace are L2-normalized; same-person cosine is typically
// ~0.4–0.5). Tune against your own photos when wiring a model.
const faceVectorThreshold = 0.45

// faceClusterer performs approximate, incremental people clustering. Without a
// dedicated face-recognition model, it embeds the textual person descriptors a
// vision model returns and clusters them online by cosine similarity against
// persisted centroids. It is the pluggable seam for a real face embedder later:
// swap the descriptor embedding for face-vector embedding and the clustering is
// unchanged.
type faceClusterer struct {
	mu       sync.Mutex
	path     string
	now      func() time.Time
	seq      int
	clusters []faceCluster
}

type faceCluster struct {
	Label    string    `json:"label"`
	Centroid []float32 `json:"centroid"`
	Count    int       `json:"count"`
}

// faceClusterView is the API-facing projection of a cluster (no centroid).
type faceClusterView struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type faceSnapshot struct {
	Seq      int           `json:"seq"`
	Clusters []faceCluster `json:"clusters"`
}

func newFaceClusterer(path string, now func() time.Time) (*faceClusterer, error) {
	f := &faceClusterer{path: path, now: now}
	if path != "" {
		var snap faceSnapshot
		if err := state.LoadJSON(path, &snap); err != nil {
			return nil, err
		}
		f.seq = snap.Seq
		f.clusters = snap.Clusters
	}
	return f, nil
}

// assign embeds the given person descriptors and assigns each to a cluster,
// returning the deduplicated set of cluster labels present in the image.
func (f *faceClusterer) assign(ctx context.Context, provider llm.Provider, descriptors []string) ([]string, error) {
	if len(descriptors) == 0 {
		return nil, nil
	}
	vecs, err := llm.Embed(ctx, provider, descriptors)
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	seen := map[string]bool{}
	labels := make([]string, 0, len(vecs))
	for _, v := range vecs {
		if len(v) == 0 {
			continue
		}
		label := f.assignOneLocked(v, faceThreshold)
		if !seen[label] {
			seen[label] = true
			labels = append(labels, label)
		}
	}
	_ = f.saveLocked()
	return labels, nil
}

// assignVectors clusters real face-recognition embeddings (from a FaceEmbedder),
// records one training sample per face, and returns the deduped cluster labels
// present in the image. This is the path a configured face model drives.
func (f *faceClusterer) assignVectors(vecs []FaceVector, mediaKey string, training *FaceTrainingStore) []string {
	if len(vecs) == 0 {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	seen := map[string]bool{}
	labels := make([]string, 0, len(vecs))
	for _, fv := range vecs {
		if len(fv.Embedding) == 0 {
			continue
		}
		label := f.assignOneLocked(fv.Embedding, faceVectorThreshold)
		if training != nil {
			training.record(label, mediaKey, fv.Embedding)
		}
		if !seen[label] {
			seen[label] = true
			labels = append(labels, label)
		}
	}
	_ = f.saveLocked()
	return labels
}

// clustersView returns a snapshot of clusters for the API.
func (f *faceClusterer) clustersView() []faceClusterView {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]faceClusterView, 0, len(f.clusters))
	for _, c := range f.clusters {
		out = append(out, faceClusterView{Label: c.Label, Count: c.Count})
	}
	return out
}

// rename relabels a cluster (e.g. to a user-supplied person name) so future
// faces matching its centroid carry the new label. Returns false if not found.
func (f *faceClusterer) rename(oldLabel, newLabel string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.clusters {
		if f.clusters[i].Label == oldLabel {
			f.clusters[i].Label = newLabel
			_ = f.saveLocked()
			return true
		}
	}
	return false
}

func (f *faceClusterer) assignOneLocked(v []float32, threshold float64) string {
	best := -1
	bestSim := threshold
	for i := range f.clusters {
		sim := cosine(v, f.clusters[i].Centroid)
		if sim >= bestSim {
			bestSim = sim
			best = i
		}
	}
	if best >= 0 {
		f.clusters[best].Centroid = blend(f.clusters[best].Centroid, v, f.clusters[best].Count)
		f.clusters[best].Count++
		return f.clusters[best].Label
	}
	f.seq++
	label := fmt.Sprintf("家庭成员 %d", f.seq)
	f.clusters = append(f.clusters, faceCluster{
		Label:    label,
		Centroid: append([]float32(nil), v...),
		Count:    1,
	})
	return label
}

func (f *faceClusterer) saveLocked() error {
	if f.path == "" {
		return nil
	}
	return state.SaveJSON(f.path, faceSnapshot{Seq: f.seq, Clusters: append([]faceCluster(nil), f.clusters...)})
}

// cosine returns the cosine similarity of two vectors (0 when shapes differ).
func cosine(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// blend folds a new vector into a centroid as a running average over count
// samples already merged.
func blend(centroid, v []float32, count int) []float32 {
	if len(centroid) != len(v) || count <= 0 {
		return append([]float32(nil), v...)
	}
	out := make([]float32, len(centroid))
	w := float32(count)
	for i := range centroid {
		out[i] = (centroid[i]*w + v[i]) / (w + 1)
	}
	return out
}
