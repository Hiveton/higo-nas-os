package aianalysis

import (
	"context"
	"testing"
	"time"
)

func newTestTrainingStore(t *testing.T) *FaceTrainingStore {
	t.Helper()
	s, err := newFaceTrainingStore("", func() time.Time { return time.Now().UTC() })
	if err != nil {
		t.Fatalf("newFaceTrainingStore: %v", err)
	}
	return s
}

func TestTrainingRecordConfirmExport(t *testing.T) {
	s := newTestTrainingStore(t)
	s.record("家庭成员 1", "media:1", []float32{0.1, 0.2})
	s.record("家庭成员 1", "media:2", []float32{0.1, 0.21})
	s.record("家庭成员 2", "media:3", []float32{0.9, 0.1})

	if st := s.Stats(); st.TotalSamples != 3 || st.ConfirmedSamples != 0 {
		t.Fatalf("unexpected stats before confirm: %+v", st)
	}

	n := s.Confirm("家庭成员 1", "爸爸")
	if n != 2 {
		t.Fatalf("expected 2 confirmed, got %d", n)
	}
	st := s.Stats()
	if st.ConfirmedSamples != 2 || st.NamedPeople != 1 {
		t.Fatalf("unexpected stats after confirm: %+v", st)
	}
	if got := len(s.Export(true)); got != 2 {
		t.Fatalf("confirmed-only export should be 2, got %d", got)
	}
	if got := len(s.Export(false)); got != 3 {
		t.Fatalf("full export should be 3, got %d", got)
	}
}

func TestRegisterModelActivatesLatest(t *testing.T) {
	s := newTestTrainingStore(t)
	s.RegisterModel(FaceModelVersion{ID: "m1", SampleCount: 10})
	s.RegisterModel(FaceModelVersion{ID: "m2", SampleCount: 20})
	models := s.Models()
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	// newest first, and only it is active
	if models[0].ID != "m2" || !models[0].Active {
		t.Fatalf("latest model should be active and first: %+v", models[0])
	}
	if models[1].Active {
		t.Fatalf("older model should be deactivated")
	}
	if s.Stats().ActiveModel != "m2" {
		t.Fatalf("active model should be m2")
	}
}

func TestSeamsDefaultToNoop(t *testing.T) {
	emb := NewFaceEmbedder("")
	if emb.Available() || emb.Name() != "none" {
		t.Fatalf("empty url should yield no-op embedder")
	}
	vecs, err := emb.DetectAndEmbed(context.Background(), []byte("x"), "image/jpeg")
	if err != nil || vecs != nil {
		t.Fatalf("no-op embedder should return (nil,nil), got %v %v", vecs, err)
	}

	tr := NewFaceTrainer("")
	if tr.Available() {
		t.Fatalf("empty url should yield no-op trainer")
	}
	if _, err := tr.Train(context.Background(), nil); err == nil {
		t.Fatalf("no-op trainer should return a clear not-configured error")
	}

	// configured seams report ready
	if !NewFaceEmbedder("http://x/embed").Available() {
		t.Fatalf("configured embedder should be available")
	}
	if !NewFaceTrainer("http://x/train").Available() {
		t.Fatalf("configured trainer should be available")
	}
}

func TestClustererRealVectorsAndRename(t *testing.T) {
	f, err := newFaceClusterer("", func() time.Time { return time.Now().UTC() })
	if err != nil {
		t.Fatalf("newFaceClusterer: %v", err)
	}
	store := newTestTrainingStore(t)
	// two near-identical vectors cluster together; a distant one splits off.
	labels := f.assignVectors([]FaceVector{
		{Embedding: []float32{1, 0, 0}},
		{Embedding: []float32{0.98, 0.02, 0}},
		{Embedding: []float32{0, 1, 0}},
	}, "media:9", store)
	if len(labels) != 2 {
		t.Fatalf("expected 2 clusters, got %d (%v)", len(labels), labels)
	}
	if got := store.Stats().TotalSamples; got != 3 {
		t.Fatalf("expected 3 training samples recorded, got %d", got)
	}
	first := f.clustersView()[0].Label
	if !f.rename(first, "妈妈") {
		t.Fatalf("rename should succeed")
	}
	for _, c := range f.clustersView() {
		if c.Label == first {
			t.Fatalf("old label should be gone after rename")
		}
	}
}
