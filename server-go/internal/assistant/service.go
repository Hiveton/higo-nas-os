package assistant

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

type Service struct {
	mu          sync.RWMutex
	now         func() time.Time
	nextMessage int
	nextAction  int
	threads     map[string]Thread
	actions     map[string]Action
	searchItems []SearchItem
	statePath   string
}

type snapshot struct {
	NextMessage int               `json:"nextMessage"`
	NextAction  int               `json:"nextAction"`
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
		service.threads = cloneThreadMap(persisted.Threads)
		service.actions = cloneActionMap(persisted.Actions)
		service.searchItems = append([]SearchItem(nil), persisted.SearchItems...)
	}
	return service, nil
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

func (s *Service) AddMessage(ctx context.Context, threadID string, request MessageRequest) (MessageResult, error) {
	if err := ctx.Err(); err != nil {
		return MessageResult{}, err
	}
	text := strings.TrimSpace(request.Text)
	if text == "" {
		return MessageResult{}, fmt.Errorf("assistant message text is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	thread, ok := s.threads[threadID]
	if !ok {
		return MessageResult{}, fmt.Errorf("assistant thread not found: %s", threadID)
	}
	now := s.now().UTC()
	userMessage := s.newMessageLocked(threadID, RoleUser, text, request.ModelPolicy, nil)
	thread.Messages = append(thread.Messages, userMessage)

	assistantText := fmt.Sprintf("已根据当前权限生成「%s」的执行草案，高风险动作会等待你确认。", text)
	assistantMessage := s.newMessageLocked(threadID, RoleAssistant, assistantText, request.ModelPolicy, nil)

	var action *Action
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

func (s *Service) ConfirmAction(ctx context.Context, id string, request ConfirmActionRequest) (Action, error) {
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
	if request.Intent != "" && request.Intent != action.Intent {
		return Action{}, fmt.Errorf("assistant action intent mismatch")
	}
	action.Status = ActionConfirmed
	action.ConfirmedAt = s.now().UTC()
	action.ConfirmedBy = request.ActorID
	s.actions[id] = action

	thread := s.threads[action.ThreadID]
	for i := range thread.PendingActions {
		if thread.PendingActions[i].ID == id {
			thread.PendingActions[i] = action
		}
	}
	thread.UpdatedAt = action.ConfirmedAt
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
