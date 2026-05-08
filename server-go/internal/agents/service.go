package agents

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

type Service struct {
	mu          sync.RWMutex
	now         func() time.Time
	nextPreview int
	nextRun     int
	nextEvent   int
	templates   []Template
	nodes       []WorkflowNode
	runs        map[string]WorkflowRun
	events      map[string][]WorkflowEvent
	statePath   string
}

type snapshot struct {
	NextPreview int                        `json:"nextPreview"`
	NextRun     int                        `json:"nextRun"`
	NextEvent   int                        `json:"nextEvent"`
	Templates   []Template                 `json:"templates"`
	Nodes       []WorkflowNode             `json:"nodes"`
	Runs        map[string]WorkflowRun     `json:"runs"`
	Events      map[string][]WorkflowEvent `json:"events"`
}

func NewService() *Service {
	service := &Service{
		now:    time.Now,
		runs:   make(map[string]WorkflowRun),
		events: make(map[string][]WorkflowEvent),
	}
	service.seed()
	return service
}

func NewServiceWithStateDir(stateDir string) (*Service, error) {
	service := NewService()
	if stateDir == "" {
		return service, nil
	}
	service.statePath = filepath.Join(stateDir, "agents.json")
	var persisted snapshot
	if err := state.LoadJSON(service.statePath, &persisted); err != nil {
		return nil, err
	}
	if len(persisted.Templates) > 0 {
		service.nextPreview = persisted.NextPreview
		service.nextRun = persisted.NextRun
		service.nextEvent = persisted.NextEvent
		service.templates = cloneTemplates(persisted.Templates)
		service.nodes = cloneNodes(persisted.Nodes)
		service.runs = cloneRunMap(persisted.Runs)
		service.events = cloneEventMap(persisted.Events)
	}
	return service, nil
}

func (s *Service) ListTemplates(ctx context.Context) []Template {
	if ctx.Err() != nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneTemplates(s.templates)
}

func (s *Service) Tools(ctx context.Context, templateID string, scopes []string) ([]Tool, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	template, ok := s.templateLocked(templateID)
	if !ok {
		return nil, fmt.Errorf("agent template not found: %s", templateID)
	}
	tools := make([]Tool, 0, len(template.Tools))
	for _, tool := range template.Tools {
		if tool.Scope == "global" || scopeAllowed(tool.Scope, scopes) {
			tools = append(tools, tool)
		}
	}
	return tools, nil
}

func (s *Service) WorkflowNodes(ctx context.Context) []WorkflowNode {
	if ctx.Err() != nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneNodes(s.nodes)
}

func (s *Service) PreviewWorkflow(ctx context.Context, request WorkflowPreviewRequest) (WorkflowPreview, error) {
	if err := ctx.Err(); err != nil {
		return WorkflowPreview{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	template, ok := s.templateLocked(request.TemplateID)
	if !ok {
		return WorkflowPreview{}, fmt.Errorf("agent template not found: %s", request.TemplateID)
	}
	if strings.TrimSpace(request.Goal) == "" {
		return WorkflowPreview{}, fmt.Errorf("workflow goal is required")
	}
	s.nextPreview++
	return s.previewLocked(template, request, fmt.Sprintf("workflow-preview-%03d", s.nextPreview)), nil
}

func (s *Service) StartRun(ctx context.Context, request WorkflowRunRequest) (WorkflowRun, error) {
	if err := ctx.Err(); err != nil {
		return WorkflowRun{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	template, ok := s.templateLocked(request.TemplateID)
	if !ok {
		return WorkflowRun{}, fmt.Errorf("agent template not found: %s", request.TemplateID)
	}
	if strings.TrimSpace(request.Goal) == "" {
		return WorkflowRun{}, fmt.Errorf("workflow goal is required")
	}

	now := s.now().UTC()
	s.nextRun++
	run := WorkflowRun{
		ID:         fmt.Sprintf("workflow-run-%03d", s.nextRun),
		TemplateID: template.ID,
		Goal:       request.Goal,
		ActorID:    request.ActorID,
		Risk:       template.DefaultRisk,
		StartedAt:  now,
		UpdatedAt:  now,
	}
	if requiresConfirmation(template.DefaultRisk) {
		run.Status = RunWaitingConfirmation
		run.RequiresConfirmation = true
		run.ConfirmationID = fmt.Sprintf("workflow-confirm-%03d", s.nextRun)
		s.runs[run.ID] = run
		s.appendEventLocked(run.ID, "run.waiting_confirmation", "工作流已生成计划，等待确认后执行。", "confirm")
		return cloneRun(run), s.saveLocked()
	}

	run.Status = RunCompleted
	s.runs[run.ID] = run
	s.appendEventLocked(run.ID, "run.started", "低风险工作流已开始执行。", "trigger")
	s.appendEventLocked(run.ID, "run.completed", "工作流已完成并写入审计事件。", "execute")
	return cloneRun(run), s.saveLocked()
}

func (s *Service) ConfirmRun(ctx context.Context, runID string, request ConfirmRunRequest) (WorkflowRun, error) {
	if err := ctx.Err(); err != nil {
		return WorkflowRun{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[runID]
	if !ok {
		return WorkflowRun{}, fmt.Errorf("workflow run not found: %s", runID)
	}
	if run.Status != RunWaitingConfirmation {
		return WorkflowRun{}, fmt.Errorf("workflow run is not waiting for confirmation: %s", runID)
	}
	if request.ConfirmationID != run.ConfirmationID {
		return WorkflowRun{}, fmt.Errorf("workflow confirmation mismatch")
	}

	run.Status = RunCompleted
	run.RequiresConfirmation = false
	run.UpdatedAt = s.now().UTC()
	s.runs[run.ID] = run
	s.appendEventLocked(run.ID, "run.confirmed", "用户已确认中风险检查点。", "confirm")
	s.appendEventLocked(run.ID, "node.completed", "重命名、归档、审计步骤已完成。", "execute")
	s.appendEventLocked(run.ID, "run.completed", "工作流已完成，可从事件面板按 run id 查询。", "execute")
	return cloneRun(run), s.saveLocked()
}

func (s *Service) CancelRun(ctx context.Context, runID string, request CancelRunRequest) (WorkflowRun, error) {
	if err := ctx.Err(); err != nil {
		return WorkflowRun{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[runID]
	if !ok {
		return WorkflowRun{}, fmt.Errorf("workflow run not found: %s", runID)
	}
	if run.Status == RunCompleted {
		return WorkflowRun{}, fmt.Errorf("completed workflow run cannot be canceled: %s", runID)
	}
	run.Status = RunCanceled
	run.RequiresConfirmation = false
	run.RollbackID = fmt.Sprintf("workflow-rollback-%s", strings.TrimPrefix(run.ID, "workflow-run-"))
	run.UpdatedAt = s.now().UTC()
	s.runs[run.ID] = run
	message := "工作流已取消，未产生文件变更。"
	if request.Reason != "" {
		message = fmt.Sprintf("%s 原因：%s", message, request.Reason)
	}
	s.appendEventLocked(run.ID, "run.canceled", message, "")
	s.appendEventLocked(run.ID, "run.rollback_registered", "已登记 Agent workflow 回滚操作。", "")
	return cloneRun(run), s.saveLocked()
}

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, snapshot{
		NextPreview: s.nextPreview,
		NextRun:     s.nextRun,
		NextEvent:   s.nextEvent,
		Templates:   cloneTemplates(s.templates),
		Nodes:       cloneNodes(s.nodes),
		Runs:        cloneRunMap(s.runs),
		Events:      cloneEventMap(s.events),
	})
}

func (s *Service) Events(ctx context.Context, runID string) ([]WorkflowEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.runs[runID]; !ok {
		return nil, fmt.Errorf("workflow run not found: %s", runID)
	}
	return cloneEvents(s.events[runID]), nil
}

func (s *Service) seed() {
	s.templates = []Template{
		{
			ID: "home-assistant", Name: "家庭资料助手", Description: "整理保修单、说明书、证件和医疗资料，提供问答与提醒。", DefaultRisk: RiskMedium,
			Tools: []Tool{
				{ID: "file-search", Name: "文件搜索", Scope: "files", Risk: RiskLow, Schema: "query,space,tags"},
				{ID: "summarize", Name: "摘要", Scope: "home", Risk: RiskLow, Schema: "fileIds"},
				{ID: "reminder", Name: "提醒", Scope: "global", Risk: RiskMedium, Schema: "title,dueAt"},
				{ID: "share", Name: "分享", Scope: "home", Risk: RiskMedium, Schema: "fileId,recipient,expiresAt"},
			},
		},
		{
			ID: "project-agent", Name: "项目资料 Agent", Description: "汇总项目文件、合同、会议纪要和素材，生成资料包。", DefaultRisk: RiskMedium,
			Tools: []Tool{
				{ID: "semantic-search", Name: "语义搜索", Scope: "team", Risk: RiskLow, Schema: "query,space"},
				{ID: "folder-summary", Name: "文件夹摘要", Scope: "team", Risk: RiskLow, Schema: "folderId"},
				{ID: "package", Name: "打包", Scope: "files", Risk: RiskMedium, Schema: "fileIds,target"},
				{ID: "permission-check", Name: "权限检查", Scope: "team", Risk: RiskMedium, Schema: "fileIds"},
			},
		},
		{
			ID: "ops-agent", Name: "设备运维 Agent", Description: "监控硬盘、备份、Docker 和网络状态，异常时建议处理。", DefaultRisk: RiskLow,
			Tools: []Tool{
				{ID: "device-monitor", Name: "设备监控", Scope: "monitoring", Risk: RiskLow, Schema: "metrics"},
				{ID: "backup-check", Name: "备份检查", Scope: "backup", Risk: RiskLow, Schema: "jobIds"},
				{ID: "notify", Name: "通知", Scope: "global", Risk: RiskLow, Schema: "channel,message"},
				{ID: "logs-read", Name: "日志读取", Scope: "monitoring", Risk: RiskLow, Schema: "source,tail"},
			},
		},
	}
	s.nodes = []WorkflowNode{
		{ID: "trigger", Label: "触发", Value: "新文件进入下载目录", ToolID: "file-search", Risk: RiskLow},
		{ID: "understand", Label: "理解", Value: "OCR + 发票识别 + 重复检测", ToolID: "semantic-search", Risk: RiskLow},
		{ID: "confirm", Label: "确认", Value: "中风险，等待用户确认", Risk: RiskMedium, Confirm: true},
		{ID: "execute", Label: "执行", Value: "重命名、归档、写入审计", ToolID: "package", Risk: RiskMedium},
	}
}

func (s *Service) previewLocked(template Template, request WorkflowPreviewRequest, id string) WorkflowPreview {
	risk := template.DefaultRisk
	preview := WorkflowPreview{
		ID:         id,
		TemplateID: template.ID,
		Goal:       request.Goal,
		Risk:       risk,
		Impact:     fmt.Sprintf("%s 将按最小权限执行「%s」，写入审计日志。", template.Name, request.Goal),
		Nodes:      cloneNodes(s.nodes),
	}
	if requiresConfirmation(risk) {
		preview.RequiresConfirmation = true
		preview.ConfirmationID = strings.Replace(id, "preview", "confirm", 1)
		preview.Checkpoints = []Checkpoint{{NodeID: "confirm", Summary: "中高风险动作确认前不执行。", Risk: risk}}
	}
	return preview
}

func (s *Service) templateLocked(id string) (Template, bool) {
	for _, template := range s.templates {
		if template.ID == id {
			return cloneTemplate(template), true
		}
	}
	return Template{}, false
}

func (s *Service) appendEventLocked(runID, eventType, message, nodeID string) {
	s.nextEvent++
	s.events[runID] = append(s.events[runID], WorkflowEvent{
		ID:        fmt.Sprintf("workflow-event-%03d", s.nextEvent),
		RunID:     runID,
		Type:      eventType,
		Message:   message,
		NodeID:    nodeID,
		CreatedAt: s.now().UTC(),
	})
}

func requiresConfirmation(risk RiskLevel) bool {
	return risk == RiskMedium || risk == RiskHigh
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

func cloneTemplates(templates []Template) []Template {
	out := make([]Template, len(templates))
	for i := range templates {
		out[i] = cloneTemplate(templates[i])
	}
	return out
}

func cloneTemplate(template Template) Template {
	template.Tools = cloneTools(template.Tools)
	return template
}

func cloneTools(tools []Tool) []Tool {
	out := make([]Tool, len(tools))
	copy(out, tools)
	return out
}

func cloneNodes(nodes []WorkflowNode) []WorkflowNode {
	out := make([]WorkflowNode, len(nodes))
	copy(out, nodes)
	return out
}

func cloneRun(run WorkflowRun) WorkflowRun {
	return run
}

func cloneRunMap(runs map[string]WorkflowRun) map[string]WorkflowRun {
	out := make(map[string]WorkflowRun, len(runs))
	for key, run := range runs {
		out[key] = cloneRun(run)
	}
	return out
}

func cloneEvents(events []WorkflowEvent) []WorkflowEvent {
	out := make([]WorkflowEvent, len(events))
	copy(out, events)
	return out
}

func cloneEventMap(events map[string][]WorkflowEvent) map[string][]WorkflowEvent {
	out := make(map[string][]WorkflowEvent, len(events))
	for key, value := range events {
		out[key] = cloneEvents(value)
	}
	return out
}
