package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/agent"
	"higoos/server-go/internal/apiclient"
	"higoos/server-go/internal/llm"
	"higoos/server-go/internal/state"
)

// maxToolRounds caps how many tool-call iterations the agent loop will run before
// forcing a final textual answer.
const maxToolRounds = 4

// systemPrompt frames the assistant for bound LLM providers.
const systemPrompt = "你是 HiGoOS NAS 的智能助手。用简洁的中文回答关于文件、备份、设备、" +
	"Docker 和权限的问题。涉及移动、删除、分享、收紧权限等高风险操作时，先说明影响并等待用户确认，不要假装已经执行。"

type Service struct {
	mu          sync.RWMutex
	now         func() time.Time
	nextMessage int
	nextAction  int
	nextThread  int
	threads     map[string]Thread
	actions     map[string]Action
	searchItems []SearchItem
	statePath   string

	llmStore   *llm.Store
	llmFactory llm.Factory
	toolClient *apiclient.Client
	tools      []agent.Tool
}

// WithLLM binds a provider store and client factory so the assistant produces
// real, streamed model responses. When no provider is configured (or this is
// never called) the assistant falls back to a canned draft reply so the app
// still runs offline.
func (s *Service) WithLLM(store *llm.Store, factory llm.Factory) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.llmStore = store
	s.llmFactory = factory
	return s
}

// WithTools enables the read-only tool-calling agent loop: the bound model may
// call the given tools (executed against the API via client) to ground its
// answers. Only effective for OpenAI-compatible providers. Safe to leave unset —
// the assistant then answers from chat history alone.
func (s *Service) WithTools(client *apiclient.Client, tools []agent.Tool) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.toolClient = client
	s.tools = tools
	return s
}

type snapshot struct {
	NextMessage int               `json:"nextMessage"`
	NextAction  int               `json:"nextAction"`
	NextThread  int               `json:"nextThread"`
	Threads     map[string]Thread `json:"threads"`
	Actions     map[string]Action `json:"actions"`
	SearchItems []SearchItem      `json:"searchItems"`
}

func NewService() *Service {
	service := &Service{
		now:     time.Now,
		threads: make(map[string]Thread),
		actions: make(map[string]Action),
	}
	service.seed()
	return service
}

func NewServiceWithStateDir(stateDir string) (*Service, error) {
	service := NewService()
	if stateDir == "" {
		return service, nil
	}
	service.statePath = filepath.Join(stateDir, "assistant.json")
	var persisted snapshot
	if err := state.LoadJSON(service.statePath, &persisted); err != nil {
		return nil, err
	}
	if len(persisted.Threads) > 0 {
		service.nextMessage = persisted.NextMessage
		service.nextAction = persisted.NextAction
		service.nextThread = persisted.NextThread
		service.threads = cloneThreadMap(persisted.Threads)
		service.actions = cloneActionMap(persisted.Actions)
		service.searchItems = append([]SearchItem(nil), persisted.SearchItems...)
	}
	return service, nil
}

// ListThreads returns lightweight summaries of every conversation, newest first.
func (s *Service) ListThreads(ctx context.Context) []ThreadSummary {
	if err := ctx.Err(); err != nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	summaries := make([]ThreadSummary, 0, len(s.threads))
	for _, thread := range s.threads {
		summaries = append(summaries, ThreadSummary{
			ID:           thread.ID,
			Title:        thread.Title,
			UpdatedAt:    thread.UpdatedAt,
			MessageCount: len(thread.Messages),
		})
	}
	sort.SliceStable(summaries, func(i, j int) bool {
		return summaries[i].UpdatedAt.After(summaries[j].UpdatedAt)
	})
	return summaries
}

// CreateThread starts a new, empty conversation and returns it.
func (s *Service) CreateThread(ctx context.Context, title string) (Thread, error) {
	return s.CreateThreadWithPreset(ctx, title, "")
}

// CreateThreadWithPreset starts a thread optionally bound to an agent preset,
// which sets a custom system prompt and a tool-domain scope for the run.
func (s *Service) CreateThreadWithPreset(ctx context.Context, title, presetID string) (Thread, error) {
	if err := ctx.Err(); err != nil {
		return Thread{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextThread++
	now := s.now().UTC()
	title = strings.TrimSpace(title)

	thread := Thread{
		ID:        fmt.Sprintf("thread-%03d", s.nextThread),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if preset, ok := PresetByID(presetID); ok {
		thread.Preset = preset.ID
		thread.SystemPrompt = preset.SystemPrompt
		thread.ToolScope = append([]string(nil), preset.ToolDomains...)
		if title == "" {
			title = preset.Name
		}
	}
	if title == "" {
		title = "新对话"
	}
	thread.Title = title
	s.threads[thread.ID] = thread
	return cloneThread(thread), s.saveLocked()
}

// DeleteThread removes a conversation and its pending actions.
func (s *Service) DeleteThread(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	thread, ok := s.threads[id]
	if !ok {
		return fmt.Errorf("assistant thread not found: %s", id)
	}
	for _, action := range thread.PendingActions {
		delete(s.actions, action.ID)
	}
	delete(s.threads, id)
	return s.saveLocked()
}

func (s *Service) GetThread(ctx context.Context, id string) (Thread, error) {
	if err := ctx.Err(); err != nil {
		return Thread{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	thread, ok := s.threads[id]
	if !ok {
		return Thread{}, fmt.Errorf("assistant thread not found: %s", id)
	}
	return cloneThread(thread), nil
}

// AddMessage records a user turn and returns the assistant's full reply. It is
// AddMessageStream with no incremental callback.
func (s *Service) AddMessage(ctx context.Context, threadID string, request MessageRequest) (MessageResult, error) {
	return s.AddMessageStream(ctx, threadID, request, nil)
}

// AddMessageStream records a user turn, streams the bound model's reply through
// emit (which also receives the terminal Done chunk), then persists the assistant
// message and any pending high-risk action. emit may be nil for a non-streamed
// call. The model is invoked without holding the service lock.
func (s *Service) AddMessageStream(ctx context.Context, threadID string, request MessageRequest, emit func(StreamEvent)) (MessageResult, error) {
	if err := ctx.Err(); err != nil {
		return MessageResult{}, err
	}
	text := strings.TrimSpace(request.Text)
	if text == "" {
		return MessageResult{}, fmt.Errorf("assistant message text is required")
	}

	// Phase 1: record the user message and snapshot history for the model.
	s.mu.Lock()
	thread, ok := s.threads[threadID]
	if !ok {
		s.mu.Unlock()
		return MessageResult{}, fmt.Errorf("assistant thread not found: %s", threadID)
	}
	userMessage := s.newMessageLocked(threadID, RoleUser, text, request.ModelPolicy, nil)
	thread.Messages = append(thread.Messages, userMessage)
	thread.UpdatedAt = s.now().UTC()
	s.threads[threadID] = thread
	history := chatHistoryLocked(thread)
	preset := presetContext{systemPrompt: thread.SystemPrompt, toolScope: append([]string(nil), thread.ToolScope...)}
	if err := s.saveLocked(); err != nil {
		s.mu.Unlock()
		return MessageResult{}, err
	}
	s.mu.Unlock()

	// Phase 2: call the bound model (no lock held; may do network I/O).
	res, err := s.generateReply(ctx, text, history, preset, emit)
	if err != nil {
		return MessageResult{}, err
	}

	// Phase 3: persist the assistant reply and any pending action.
	s.mu.Lock()
	defer s.mu.Unlock()

	thread = s.threads[threadID]
	now := s.now().UTC()
	assistantMessage := s.newMessageLocked(threadID, RoleAssistant, res.Text, request.ModelPolicy, nil)
	assistantMessage.Tools = res.Traces

	var action *Action
	switch {
	case res.Pending != nil:
		// A real write tool call awaiting confirmation. Confirming it executes
		// the stored tool (see ConfirmAction).
		s.nextAction++
		created := Action{
			ID:             fmt.Sprintf("assistant-action-%03d", s.nextAction),
			ThreadID:       threadID,
			MessageID:      assistantMessage.ID,
			ActorID:        request.ActorID,
			Intent:         res.Pending.Impact,
			Risk:           RiskHigh,
			Status:         ActionPending,
			ConfirmationID: fmt.Sprintf("assistant-confirm-%03d", s.nextAction),
			Impact:         res.Pending.Impact,
			ToolName:       res.Pending.Name,
			ToolArgs:       res.Pending.Args,
			CreatedAt:      now,
		}
		assistantMessage.ActionID = created.ID
		assistantMessage.RequiresConfirmation = true
		s.actions[created.ID] = created
		thread.PendingActions = append(thread.PendingActions, created)
		action = &created
	case !res.Agent:
		// Plain-chat fallback path: keep the keyword-based heuristic action.
		risk, confirmable := classifyAssistantAction(text)
		if confirmable {
			s.nextAction++
			created := Action{
				ID:             fmt.Sprintf("assistant-action-%03d", s.nextAction),
				ThreadID:       threadID,
				MessageID:      assistantMessage.ID,
				ActorID:        request.ActorID,
				Intent:         text,
				Risk:           risk,
				Status:         ActionPending,
				ConfirmationID: fmt.Sprintf("assistant-confirm-%03d", s.nextAction),
				Impact:         impactForAssistantAction(text, risk),
				RollbackID:     fmt.Sprintf("assistant-rollback-%03d", s.nextAction),
				CreatedAt:      now,
			}
			assistantMessage.ActionID = created.ID
			assistantMessage.RequiresConfirmation = true
			s.actions[created.ID] = created
			thread.PendingActions = append(thread.PendingActions, created)
			action = &created
		}
	}

	thread.Messages = append(thread.Messages, assistantMessage)
	thread.UpdatedAt = now
	s.threads[threadID] = thread

	result := MessageResult{
		Thread:               cloneThread(thread),
		UserMessage:          cloneMessage(userMessage),
		AssistantMessage:     cloneMessage(assistantMessage),
		RequiresConfirmation: action != nil,
	}
	if action != nil {
		cloned := cloneAction(*action)
		result.Action = &cloned
	}
	return result, s.saveLocked()
}

// generateReply streams a reply from the bound model. When no provider is
// configured (or the factory can't resolve one) it falls back to a canned draft
// so the assistant keeps working offline. A mid-stream error is surfaced to the
// caller. emit, when non-nil, receives every delta plus the terminal Done chunk.
// pendingWrite is a confirmation-gated write tool call captured during the agent
// loop instead of being executed.
type pendingWrite struct {
	Name   string
	Args   string
	Impact string
}

// replyResult bundles the assistant reply with tool traces and any pending write
// action. Agent marks whether the tool-calling loop produced it.
type replyResult struct {
	Text    string
	Traces  []ToolTrace
	Agent   bool
	Pending *pendingWrite
}

// presetContext carries per-thread agent customization into a reply.
type presetContext struct {
	systemPrompt string
	toolScope    []string
}

func (s *Service) generateReply(ctx context.Context, userText string, history []llm.ChatMessage, preset presetContext, emit func(StreamEvent)) (replyResult, error) {
	s.mu.RLock()
	store := s.llmStore
	factory := s.llmFactory
	s.mu.RUnlock()

	if store == nil || factory == nil {
		return replyResult{Text: cannedReply(userText, emit)}, nil
	}
	provider, err := store.Default()
	if err != nil {
		return replyResult{Text: cannedReply(userText, emit)}, nil
	}
	client, err := factory(provider.Kind)
	if err != nil {
		return replyResult{Text: cannedReply(userText, emit)}, nil
	}

	s.mu.RLock()
	toolClient := s.toolClient
	tools := s.tools
	s.mu.RUnlock()

	prompt := systemPrompt
	if strings.TrimSpace(preset.systemPrompt) != "" {
		prompt = preset.systemPrompt
	}
	tools = scopeTools(tools, preset.toolScope)
	messages := append([]llm.ChatMessage{{Role: "system", Content: prompt}}, history...)

	// Tool-calling agent loop (OpenAI-compatible providers only).
	if toolClient != nil && len(tools) > 0 && provider.Kind == llm.KindOpenAI {
		return agentReply(ctx, provider, messages, tools, toolClient, emit)
	}

	// Plain streamed chat.
	var sb strings.Builder
	streamErr := client.Stream(ctx, provider, llm.ChatRequest{Messages: messages}, func(chunk llm.StreamChunk) {
		sb.WriteString(chunk.Delta)
		emitDelta(emit, chunk.Delta)
	})
	if streamErr != nil {
		return replyResult{}, streamErr
	}
	reply := strings.TrimSpace(sb.String())
	if reply == "" {
		reply = "（模型没有返回内容）"
	}
	return replyResult{Text: reply}, nil
}

// emitDelta forwards a text delta to emit when both are non-empty.
func emitDelta(emit func(StreamEvent), delta string) {
	if emit != nil && delta != "" {
		emit(StreamEvent{Delta: delta})
	}
}

// ToolInfo is the catalog metadata surfaced to the UI.
type ToolInfo struct {
	Name        string `json:"name"`
	Domain      string `json:"domain"`
	Description string `json:"description"`
	ReadOnly    bool   `json:"readOnly"`
}

// Tools returns metadata for the bound MCP tool catalog (for the workbench
// capability panel).
func (s *Service) Tools() []ToolInfo {
	s.mu.RLock()
	tools := s.tools
	s.mu.RUnlock()
	out := make([]ToolInfo, 0, len(tools))
	for _, t := range tools {
		out = append(out, ToolInfo{
			Name:        t.Name,
			Domain:      toolDomain(t.Name),
			Description: t.Description,
			ReadOnly:    !t.Write,
		})
	}
	return out
}

// toolDomain extracts the domain segment from a sanitized MCP tool name
// ("higo_storage_pools_list" -> "storage").
func toolDomain(name string) string {
	parts := strings.Split(name, "_")
	if len(parts) >= 2 && parts[0] == "higo" {
		return parts[1]
	}
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// scopeTools keeps only tools whose domain is in scope. An empty scope returns
// all tools unchanged.
func scopeTools(tools []agent.Tool, scope []string) []agent.Tool {
	if len(scope) == 0 {
		return tools
	}
	allowed := make(map[string]bool, len(scope))
	for _, d := range scope {
		allowed[d] = true
	}
	out := make([]agent.Tool, 0, len(tools))
	for _, t := range tools {
		if allowed[toolDomain(t.Name)] {
			out = append(out, t)
		}
	}
	return out
}

// agentReply runs the read-only tool-calling loop: it streams completions with
// the tool catalog attached, executes any requested tools against the API, feeds
// results back, and repeats until the model answers in text (or the round cap is
// hit, after which a final tool-free pass forces an answer). Only content deltas
// reach emit, so tool-call rounds are invisible in the persisted message.
func agentReply(ctx context.Context, provider llm.Provider, messages []llm.ChatMessage, tools []agent.Tool, client *apiclient.Client, emit func(StreamEvent)) (replyResult, error) {
	toolDefs := make([]llm.ToolDef, 0, len(tools))
	byName := agent.ByName(tools)
	for _, t := range tools {
		toolDefs = append(toolDefs, llm.ToolDef{Name: t.Name, Description: t.Description, Parameters: t.Parameters})
	}

	var sb strings.Builder
	var traces []ToolTrace
	var pending *pendingWrite
	sink := func(chunk llm.StreamChunk) {
		sb.WriteString(chunk.Delta)
		emitDelta(emit, chunk.Delta)
	}

	for round := 0; round < maxToolRounds; round++ {
		calls, err := llm.StreamWithTools(ctx, provider, messages, toolDefs, sink)
		if err != nil {
			return replyResult{}, err
		}
		if len(calls) == 0 {
			break // model answered in text; content already streamed
		}
		messages = append(messages, llm.ChatMessage{Role: "assistant", ToolCalls: calls})
		for _, call := range calls {
			tool, isWrite := byName[call.Name], false
			if t, ok := byName[call.Name]; ok {
				isWrite = t.Write
			}

			// Write tools are NOT executed here — they become a single pending
			// action the user must confirm. Only the first write per turn is
			// captured; further writes are deferred so confirmations stay 1-at-a-time.
			if isWrite {
				var result json.RawMessage
				if pending == nil {
					var args map[string]any
					if strings.TrimSpace(call.Arguments) != "" {
						_ = json.Unmarshal([]byte(call.Arguments), &args)
					}
					impact := tool.Description
					if tool.Impact != nil {
						impact = tool.Impact(args)
					}
					pending = &pendingWrite{Name: call.Name, Args: call.Arguments, Impact: impact}
					traces = append(traces, ToolTrace{Name: call.Name, Summary: "待确认：" + impact})
					if emit != nil {
						emit(StreamEvent{Tool: &ToolEvent{Phase: "confirm", Name: call.Name, Args: call.Arguments, Summary: impact}})
					}
					result = json.RawMessage(`{"status":"awaiting_user_confirmation","note":"已生成待确认卡片，用户确认后才会执行；不要假设已执行。"}`)
				} else {
					result = json.RawMessage(`{"status":"deferred","note":"已有一个操作待确认，请让用户先确认后再继续。"}`)
				}
				messages = append(messages, llm.ChatMessage{Role: "tool", ToolCallID: call.ID, Content: string(result)})
				continue
			}

			// Read-only tools execute inline.
			if emit != nil {
				emit(StreamEvent{Tool: &ToolEvent{Phase: "start", Name: call.Name, Args: call.Arguments}})
			}
			result := runTool(ctx, client, byName, call)
			summary := summarizeToolResult(result)
			traces = append(traces, ToolTrace{Name: call.Name, Summary: summary})
			if emit != nil {
				emit(StreamEvent{Tool: &ToolEvent{Phase: "done", Name: call.Name, Summary: summary}})
			}
			messages = append(messages, llm.ChatMessage{Role: "tool", ToolCallID: call.ID, Content: string(result)})
		}
	}

	// If the model only ever called tools (hit the cap), force a textual answer.
	if strings.TrimSpace(sb.String()) == "" {
		if _, err := llm.StreamWithTools(ctx, provider, messages, nil, sink); err != nil {
			return replyResult{}, err
		}
	}

	reply := strings.TrimSpace(sb.String())
	if reply == "" {
		reply = "（模型没有返回内容）"
	}
	return replyResult{Text: reply, Traces: traces, Agent: true, Pending: pending}, nil
}

// runTool executes a single tool call and returns its JSON result (or a JSON
// error object the model can read).
func runTool(ctx context.Context, client *apiclient.Client, byName map[string]agent.Tool, call llm.ToolCall) json.RawMessage {
	tool, ok := byName[call.Name]
	if !ok {
		return toolError("unknown tool: " + call.Name)
	}
	var args map[string]any
	if strings.TrimSpace(call.Arguments) != "" {
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return toolError("invalid tool arguments: " + err.Error())
		}
	}
	out, err := tool.Run(ctx, client, args)
	if err != nil {
		return toolError(err.Error())
	}
	if len(out) == 0 {
		return json.RawMessage("null")
	}
	return out
}

// summarizeToolResult produces a short human-readable note for the analysis view:
// the surfaced error, or an element/field count, falling back to a truncated
// preview of the raw JSON.
func summarizeToolResult(result json.RawMessage) string {
	var asErr struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(result, &asErr) == nil && asErr.Error != "" {
		return "出错：" + asErr.Error
	}
	var arr []json.RawMessage
	if json.Unmarshal(result, &arr) == nil {
		return fmt.Sprintf("返回 %d 条记录", len(arr))
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(result, &obj) == nil {
		return fmt.Sprintf("返回 %d 个字段", len(obj))
	}
	preview := strings.TrimSpace(string(result))
	if len(preview) > 60 {
		preview = preview[:60] + "…"
	}
	return preview
}

func toolError(msg string) json.RawMessage {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return b
}

// cannedReply produces the offline draft response and pushes it through emit as a
// single delta plus a terminal Done chunk.
func cannedReply(userText string, emit func(StreamEvent)) string {
	text := fmt.Sprintf("已根据当前权限生成「%s」的执行草案，高风险动作会等待你确认。", userText)
	emitDelta(emit, text)
	return text
}

// chatHistoryLocked maps thread messages to provider-agnostic chat turns. Callers
// hold s.mu.
func chatHistoryLocked(thread Thread) []llm.ChatMessage {
	out := make([]llm.ChatMessage, 0, len(thread.Messages))
	for _, m := range thread.Messages {
		if strings.TrimSpace(m.Text) == "" {
			continue
		}
		role := "user"
		if m.Role == RoleAssistant {
			role = "assistant"
		}
		out = append(out, llm.ChatMessage{Role: role, Content: m.Text})
	}
	return out
}

func (s *Service) ConfirmAction(ctx context.Context, id string, request ConfirmActionRequest) (Action, error) {
	if err := ctx.Err(); err != nil {
		return Action{}, err
	}

	// Phase 1: validate and capture the tool call under the lock.
	s.mu.Lock()
	action, ok := s.actions[id]
	if !ok {
		s.mu.Unlock()
		return Action{}, fmt.Errorf("assistant action not found: %s", id)
	}
	if action.Status != ActionPending {
		s.mu.Unlock()
		return Action{}, fmt.Errorf("assistant action is not pending: %s", id)
	}
	if request.Intent != "" && request.Intent != action.Intent {
		s.mu.Unlock()
		return Action{}, fmt.Errorf("assistant action intent mismatch")
	}
	toolClient := s.toolClient
	byName := agent.ByName(s.tools)
	toolName, toolArgs, impact, threadID := action.ToolName, action.ToolArgs, action.Impact, action.ThreadID
	s.mu.Unlock()

	// Phase 2: execute the confirmed write tool, if any (no lock held).
	var resultText string
	if toolName != "" {
		tool, found := byName[toolName]
		switch {
		case !found || toolClient == nil:
			resultText = "⚠️ 无法执行（工具不可用）：" + impact
		default:
			var args map[string]any
			if strings.TrimSpace(toolArgs) != "" {
				_ = json.Unmarshal([]byte(toolArgs), &args)
			}
			out, err := tool.Run(ctx, toolClient, args)
			if err != nil {
				resultText = fmt.Sprintf("❌ 执行失败：%s（%s）", impact, err.Error())
			} else {
				resultText = fmt.Sprintf("✅ 已执行：%s（%s）", impact, summarizeToolResult(out))
			}
		}
	}

	// Phase 3: mark confirmed and append the result message.
	s.mu.Lock()
	defer s.mu.Unlock()
	action = s.actions[id]
	action.Status = ActionConfirmed
	action.ConfirmedAt = s.now().UTC()
	action.ConfirmedBy = request.ActorID
	s.actions[id] = action

	thread := s.threads[threadID]
	for i := range thread.PendingActions {
		if thread.PendingActions[i].ID == id {
			thread.PendingActions[i] = action
		}
	}
	if resultText != "" {
		resultMsg := s.newMessageLocked(threadID, RoleAssistant, resultText, "", nil)
		thread.Messages = append(thread.Messages, resultMsg)
	}
	thread.UpdatedAt = action.ConfirmedAt
	s.threads[threadID] = thread
	return cloneAction(action), s.saveLocked()
}

// CancelAction rejects a pending action without executing it.
func (s *Service) CancelAction(ctx context.Context, id string, request ConfirmActionRequest) (Action, error) {
	if err := ctx.Err(); err != nil {
		return Action{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	action, ok := s.actions[id]
	if !ok {
		return Action{}, fmt.Errorf("assistant action not found: %s", id)
	}
	if action.Status != ActionPending {
		return Action{}, fmt.Errorf("assistant action is not pending: %s", id)
	}
	action.Status = ActionCanceled
	action.ConfirmedBy = request.ActorID
	s.actions[id] = action

	thread := s.threads[action.ThreadID]
	for i := range thread.PendingActions {
		if thread.PendingActions[i].ID == id {
			thread.PendingActions[i] = action
		}
	}
	thread.UpdatedAt = s.now().UTC()
	s.threads[action.ThreadID] = thread
	return cloneAction(action), s.saveLocked()
}

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, snapshot{
		NextMessage: s.nextMessage,
		NextAction:  s.nextAction,
		NextThread:  s.nextThread,
		Threads:     cloneThreadMap(s.threads),
		Actions:     cloneActionMap(s.actions),
		SearchItems: append([]SearchItem(nil), s.searchItems...),
	})
}

func (s *Service) SemanticSearch(ctx context.Context, request SemanticSearchRequest) (SemanticSearchResponse, error) {
	if err := ctx.Err(); err != nil {
		return SemanticSearchResponse{}, err
	}
	query := strings.TrimSpace(request.Query)
	if query == "" {
		return SemanticSearchResponse{}, fmt.Errorf("semantic search query is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	type scored struct {
		item  SearchItem
		score float64
	}
	var matches []scored
	for _, item := range s.searchItems {
		if !scopeAllowed(item.Scope, request.Scopes) {
			continue
		}
		score := semanticScore(item, query)
		if score == 0 {
			continue
		}
		item.Score = score
		matches = append(matches, scored{item: item, score: score})
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].item.Title < matches[j].item.Title
	})

	limit := request.Limit
	if limit <= 0 || limit > len(matches) {
		limit = len(matches)
	}
	items := make([]SearchItem, 0, limit)
	citations := make([]Citation, 0, minInt(limit, 3))
	for i := 0; i < limit; i++ {
		item := matches[i].item
		items = append(items, item)
		if i < 3 {
			citations = append(citations, Citation{
				ItemID: item.ID,
				Title:  item.Title,
				Path:   item.Path,
				Scope:  item.Scope,
				Quote:  item.Snippet,
			})
		}
	}

	answer := "没有找到当前权限范围内的相关结果。"
	if len(items) > 0 {
		answer = fmt.Sprintf("找到了 %d 个当前权限范围内的相关结果，优先参考「%s」。", len(items), items[0].Title)
	}
	return SemanticSearchResponse{Answer: answer, Items: items, Citations: citations}, nil
}

func (s *Service) seed() {
	now := s.now().UTC()
	thread := Thread{
		ID:        "thread-current",
		Title:     "当前对话",
		CreatedAt: now,
		UpdatedAt: now,
	}
	for _, seed := range []struct {
		role MessageRole
		text string
	}{
		{RoleUser, "找一下上个月客户 A 的最终合同，并确认有没有备份。"},
		{RoleAssistant, "找到了 1 份最终版合同，位于团队空间/客户A/合同。该文件已进入每日快照和异地备份，权限为项目组可见。"},
		{RoleAssistant, "我还发现 3 个相关附件未加入项目资料图谱，是否需要生成整理计划？"},
	} {
		thread.Messages = append(thread.Messages, s.newMessageLocked(thread.ID, seed.role, seed.text, "local-first", nil))
	}
	s.threads[thread.ID] = thread
	s.searchItems = []SearchItem{
		{ID: "doc-contract-a-final", Title: "客户 A 合同最终版", Path: "/团队空间/客户A/合同/客户 A 合同最终版.docx", Scope: "team", Snippet: "最终合同已纳入每日快照和异地备份，权限为项目组可见。"},
		{ID: "doc-warranty-2026", Title: "2026 家庭保险与保修资料", Path: "/家庭空间/保修/2026 家庭保险与保修资料.pdf", Scope: "home", Snippet: "包含家电保修单和到期提醒，适合由家庭资料助手生成提醒。"},
		{ID: "doc-invoices-unarchived", Title: "下载目录未归档发票", Path: "/财务票据/下载目录/未归档发票", Scope: "finance", Snippet: "31 张发票中有 4 张可能重复，建议按年月归档。"},
		{ID: "doc-share-risks", Title: "过期公开分享链接", Path: "/团队空间/分享审计/公开链接.csv", Scope: "team", Snippet: "发现 3 个公开链接仍可访问，包含团队空间资料。"},
	}
}

func (s *Service) newMessageLocked(threadID string, role MessageRole, text, modelPolicy string, citations []Citation) Message {
	s.nextMessage++
	return Message{
		ID:          fmt.Sprintf("assistant-message-%03d", s.nextMessage),
		ThreadID:    threadID,
		Role:        role,
		Text:        text,
		Citations:   cloneCitations(citations),
		CreatedAt:   s.now().UTC(),
		ModelPolicy: modelPolicy,
	}
}

func classifyAssistantAction(text string) (RiskLevel, bool) {
	lowered := strings.ToLower(text)
	highRiskWords := []string{"删除", "公开链接", "权限", "收紧", "外发", "cloud model"}
	for _, word := range highRiskWords {
		if strings.Contains(lowered, word) {
			return RiskHigh, true
		}
	}
	mediumRiskWords := []string{"整理", "移动", "重命名", "分享", "计划", "docker", "归档"}
	for _, word := range mediumRiskWords {
		if strings.Contains(lowered, word) {
			return RiskMedium, true
		}
	}
	return RiskLow, false
}

func impactForAssistantAction(text string, risk RiskLevel) string {
	if risk == RiskHigh {
		return fmt.Sprintf("将预览「%s」涉及的高风险变更，确认前不执行。", text)
	}
	return fmt.Sprintf("将预览「%s」涉及的中风险变更，并附带回滚计划。", text)
}

func semanticScore(item SearchItem, query string) float64 {
	haystack := strings.ToLower(strings.Join([]string{item.Title, item.Path, item.Snippet, item.Scope}, " "))
	score := 0.0
	for _, term := range strings.Fields(strings.ToLower(query)) {
		if strings.Contains(haystack, term) {
			score += 1
		}
	}
	if score == 0 && strings.Contains(haystack, strings.ToLower(query)) {
		score = 1
	}
	return score
}

func scopeAllowed(scope string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, candidate := range allowed {
		if strings.EqualFold(strings.TrimSpace(candidate), scope) {
			return true
		}
	}
	return false
}

func cloneThread(thread Thread) Thread {
	thread.Messages = cloneMessages(thread.Messages)
	thread.PendingActions = cloneActions(thread.PendingActions)
	thread.ToolScope = append([]string(nil), thread.ToolScope...)
	return thread
}

func cloneThreadMap(threads map[string]Thread) map[string]Thread {
	out := make(map[string]Thread, len(threads))
	for key, thread := range threads {
		out[key] = cloneThread(thread)
	}
	return out
}

func cloneMessages(messages []Message) []Message {
	out := make([]Message, len(messages))
	for i := range messages {
		out[i] = cloneMessage(messages[i])
	}
	return out
}

func cloneMessage(message Message) Message {
	message.Citations = cloneCitations(message.Citations)
	return message
}

func cloneActions(actions []Action) []Action {
	out := make([]Action, len(actions))
	copy(out, actions)
	return out
}

func cloneAction(action Action) Action {
	return action
}

func cloneActionMap(actions map[string]Action) map[string]Action {
	out := make(map[string]Action, len(actions))
	for key, action := range actions {
		out[key] = cloneAction(action)
	}
	return out
}

func cloneCitations(citations []Citation) []Citation {
	out := make([]Citation, len(citations))
	copy(out, citations)
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
