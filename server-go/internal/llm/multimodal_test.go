package llm

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEmbedParsesAndOrders(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		// Return out of order to verify sorting by index.
		io.WriteString(w, `{"data":[{"index":1,"embedding":[0.3,0.4]},{"index":0,"embedding":[0.1,0.2]}]}`)
	}))
	defer srv.Close()

	vecs, err := Embed(context.Background(),
		Provider{Kind: KindOpenAI, Purpose: PurposeEmbedding, BaseURL: srv.URL, APIKey: "k", Model: "embed-1"},
		[]string{"a", "b"})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if gotPath != "/embeddings" {
		t.Fatalf("path = %s", gotPath)
	}
	if !strings.Contains(gotBody, `"model":"embed-1"`) {
		t.Fatalf("body missing model: %s", gotBody)
	}
	if len(vecs) != 2 || vecs[0][0] != 0.1 || vecs[1][0] != 0.3 {
		t.Fatalf("unexpected ordered vectors: %v", vecs)
	}
}

func TestDefaultForPurposeIsolation(t *testing.T) {
	s := NewStore()
	chat := PurposeChat
	emb := PurposeEmbedding
	c, _ := s.Create(ProviderInput{Name: sp("chat"), Kind: kp(KindOpenAI), Model: sp("m"), Purpose: &chat, IsDefault: bp(true)})
	e, _ := s.Create(ProviderInput{Name: sp("emb"), Kind: kp(KindOpenAI), Model: sp("e"), Purpose: &emb, IsDefault: bp(true)})

	gotChat, err := s.DefaultFor(PurposeChat)
	if err != nil || gotChat.ID != c.ID {
		t.Fatalf("chat default = %v err=%v", gotChat.ID, err)
	}
	gotEmb, err := s.DefaultFor(PurposeEmbedding)
	if err != nil || gotEmb.ID != e.ID {
		t.Fatalf("embedding default = %v err=%v", gotEmb.ID, err)
	}
	// Making the embedding provider default must NOT clear the chat default.
	chatView, _ := s.Get(c.ID)
	if !chatView.IsDefault {
		t.Fatal("chat default was cleared by embedding default (purposes must be independent)")
	}
}

func sp(s string) *string       { return &s }
func bp(b bool) *bool           { return &b }
func kp(k ProviderKind) *ProviderKind { return &k }
