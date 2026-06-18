package assistant

import (
	"context"
	"strings"
	"testing"

	"higoos/server-go/internal/llm"
)

type fakeLLMClient struct{ deltas []string }

func (f fakeLLMClient) Stream(_ context.Context, _ llm.Provider, _ llm.ChatRequest, emit func(llm.StreamChunk)) error {
	for _, d := range f.deltas {
		emit(llm.StreamChunk{Delta: d})
	}
	emit(llm.StreamChunk{Done: true})
	return nil
}

func ptr[T any](v T) *T { return &v }

func TestAddMessageStreamUsesBoundLLM(t *testing.T) {
	store := llm.NewStore()
	kind := llm.KindOpenAI
	if _, err := store.Create(llm.ProviderInput{
		Name: ptr("p"), Kind: &kind, Model: ptr("m"), APIKey: ptr("secret"),
	}); err != nil {
		t.Fatalf("create provider: %v", err)
	}

	svc := NewService().WithLLM(store, func(llm.ProviderKind) (llm.Client, error) {
		return fakeLLMClient{deltas: []string{"Hel", "lo ", "world"}}, nil
	})

	var streamed strings.Builder
	res, err := svc.AddMessageStream(context.Background(), "thread-current", MessageRequest{Text: "hi"}, func(e StreamEvent) {
		streamed.WriteString(e.Delta)
	})
	if err != nil {
		t.Fatalf("AddMessageStream: %v", err)
	}
	if streamed.String() != "Hello world" {
		t.Fatalf("streamed text = %q, want %q", streamed.String(), "Hello world")
	}
	if res.AssistantMessage.Text != "Hello world" {
		t.Fatalf("persisted assistant text = %q", res.AssistantMessage.Text)
	}
}

func TestAddMessageFallsBackWithoutProvider(t *testing.T) {
	svc := NewService() // no WithLLM
	res, err := svc.AddMessage(context.Background(), "thread-current", MessageRequest{Text: "整理下载目录"})
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	if !strings.Contains(res.AssistantMessage.Text, "执行草案") {
		t.Fatalf("expected canned fallback, got %q", res.AssistantMessage.Text)
	}
}
