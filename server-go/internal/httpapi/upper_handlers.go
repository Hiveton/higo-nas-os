package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"higoos/server-go/internal/assistant"
	"higoos/server-go/internal/files"
	"higoos/server-go/internal/media"
	"higoos/server-go/internal/platform"
	"higoos/server-go/internal/security"
	"higoos/server-go/internal/steward"
)

func (a *API) mediaItems(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	items, err := a.media.Items(r.Context(), media.ItemFilter{
		Dimension: media.DimensionKey(r.URL.Query().Get("dimension")),
		Facet:     r.URL.Query().Get("facet"),
	})
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "media_items_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, items)
}

func (a *API) mediaScan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	result, err := a.media.Scan(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "media_scan_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, result)
}

func (a *API) mediaAlbums(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		albums, err := a.media.Albums(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "media_albums_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, albums)
	case http.MethodPost:
		var body media.CreateAlbumRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		album, err := a.media.CreateAlbum(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "media_album_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, album)
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPost)
	}
}

func (a *API) mediaMemories(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body media.CreateMemoryRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	run, err := a.media.CreateMemory(r.Context(), body)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "media_memory_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapTaskResponse(run.ID, string(run.Status), run.Message))
}

func (a *API) mediaMergePeople(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body media.MergePeopleRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	result, err := a.media.MergePeople(r.Context(), body)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "media_people_merge_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapTaskResponse(result.ID, string(result.Status), result.Message))
}

func (a *API) mediaSubtitleJobs(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body media.CreateMediaJobRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	job, err := a.media.CreateSubtitleJob(r.Context(), body)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "media_subtitle_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapTaskResponse(job.ID, string(job.Status), job.Message))
}

func (a *API) mediaTranscodeJobs(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body media.CreateMediaJobRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	job, err := a.media.CreateTranscodeJob(r.Context(), body)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "media_transcode_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapTaskResponse(job.ID, string(job.Status), job.Message))
}

func (a *API) mediaShares(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body media.CreateShareRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	share, err := a.media.CreateShare(r.Context(), body)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "media_share_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{
		"id":        share.ID,
		"name":      share.AlbumName,
		"target":    fmt.Sprintf("/media/albums/%d", share.AlbumID),
		"access":    fmt.Sprintf("密码 + %d 天", share.ExpiresInDays),
		"downloads": 0,
		"risk":      "中风险",
		"active":    true,
		"message":   share.Message,
	})
}

func (a *API) assistantSemanticSearch(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body assistant.SemanticSearchRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	// Prefer the real pgvector index when a database is configured; otherwise
	// fall back to the assistant's built-in keyword search over seed data.
	if a.runSemanticSearch(w, r, body.Query, body.Scopes, body.Limit) {
		return
	}
	result, err := a.assistant.SemanticSearch(r.Context(), body)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "semantic_search_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapSemanticSearch(result))
}

func (a *API) assistantThreads(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		platform.WriteJSON(w, r, http.StatusOK, a.assistant.ListThreads(r.Context()))
	case http.MethodPost:
		var body struct {
			Title    string `json:"title"`
			PresetID string `json:"presetId"`
		}
		if r.ContentLength != 0 {
			if err := decodeJSON(r, &body); err != nil {
				platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
		}
		thread, err := a.assistant.CreateThreadWithPreset(r.Context(), body.Title, body.PresetID)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "assistant_thread_create_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, mapAssistantThread(thread))
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPost)
	}
}

func (a *API) assistantThreadByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/assistant/threads/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		platform.WriteError(w, r, http.StatusNotFound, "assistant_route_not_found", "assistant thread route not found")
		return
	}
	threadID := parts[0]
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			thread, err := a.assistant.GetThread(r.Context(), threadID)
			if err != nil {
				platform.WriteError(w, r, http.StatusNotFound, "assistant_thread_not_found", err.Error())
				return
			}
			platform.WriteJSON(w, r, http.StatusOK, mapAssistantThread(thread))
		case http.MethodDelete:
			if err := a.assistant.DeleteThread(r.Context(), threadID); err != nil {
				platform.WriteError(w, r, http.StatusNotFound, "assistant_thread_not_found", err.Error())
				return
			}
			platform.WriteJSON(w, r, http.StatusOK, map[string]any{"id": threadID, "deleted": true})
		default:
			allowMethod(w, r, http.MethodGet, http.MethodDelete)
		}
		return
	}
	if len(parts) == 2 && parts[1] == "messages" {
		if !allowMethod(w, r, http.MethodPost) {
			return
		}
		var body struct {
			ActorID     string   `json:"actorId"`
			Role        string   `json:"role"`
			Text        string   `json:"text"`
			Scopes      []string `json:"scopes"`
			ModelPolicy string   `json:"modelPolicy"`
		}
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		messageReq := assistant.MessageRequest{
			ActorID:     body.ActorID,
			Text:        body.Text,
			Scopes:      body.Scopes,
			ModelPolicy: body.ModelPolicy,
		}
		if wantsEventStream(r) {
			a.streamAssistantMessage(w, r, threadID, messageReq)
			return
		}
		result, err := a.assistant.AddMessage(r.Context(), threadID, messageReq)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "assistant_message_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, mapAssistantMessage(result.AssistantMessage))
		return
	}
	platform.WriteError(w, r, http.StatusNotFound, "assistant_route_not_found", "assistant thread route not found")
}

func (a *API) assistantActionByID(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/assistant/actions/"), "/"), "/")
	if len(parts) != 2 || (parts[1] != "confirm" && parts[1] != "cancel") {
		platform.WriteError(w, r, http.StatusNotFound, "assistant_action_route_not_found", "assistant action route not found")
		return
	}
	var body assistant.ConfirmActionRequest
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
	}
	var (
		action assistant.Action
		err    error
	)
	if parts[1] == "cancel" {
		action, err = a.assistant.CancelAction(r.Context(), parts[0], body)
	} else {
		action, err = a.assistant.ConfirmAction(r.Context(), parts[0], body)
	}
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "assistant_action_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapTaskResponse(action.ID, string(action.Status), action.Impact))
}

func (a *API) stewardSuggestions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		platform.WriteJSON(w, r, http.StatusOK, mapStewardSuggestions(a.steward.ListSuggestions(r.Context())))
	case http.MethodPost:
		// Re-analyze the file tree and replace pending suggestions with real ones.
		suggestions, err := a.refreshSteward(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "steward_refresh_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, mapStewardSuggestions(suggestions))
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPost)
	}
}

// refreshSteward walks the file tree and replaces pending steward suggestions
// with freshly analyzed ones (duplicates, archive candidates).
func (a *API) refreshSteward(ctx context.Context) ([]steward.Suggestion, error) {
	if a.files == nil {
		return a.steward.ListSuggestions(ctx), nil
	}
	tree, err := a.files.Tree(ctx, "")
	if err != nil {
		return nil, err
	}
	var infos []steward.FileInfo
	collectFileInfos(tree, &infos)
	return a.steward.ReplaceSuggestions(ctx, steward.Analyze(infos, time.Now()))
}

func collectFileInfos(node files.FileNode, out *[]steward.FileInfo) {
	if !node.IsDir {
		*out = append(*out, steward.FileInfo{
			ID: node.ID, Name: node.Name, Path: node.Path, Type: node.Type,
			Space: node.Space, SizeBytes: node.SizeBytes, Modified: node.Modified,
			Tags: node.Tags, Summary: node.Summary,
		})
	}
	for _, child := range node.Children {
		collectFileInfos(child, out)
	}
}

func (a *API) stewardSuggestionByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/steward/suggestions/"), "/"), "/")
	if len(parts) != 2 {
		platform.WriteError(w, r, http.StatusNotFound, "steward_route_not_found", "steward suggestion route not found")
		return
	}
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	switch parts[1] {
	case "preview":
		var body steward.PreviewRequest
		if r.ContentLength != 0 {
			if err := decodeJSON(r, &body); err != nil {
				platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
		}
		preview, err := a.steward.Preview(r.Context(), parts[0], body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "steward_preview_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, preview)
	case "confirm":
		var body steward.ConfirmRequest
		if r.ContentLength != 0 {
			if err := decodeJSON(r, &body); err != nil {
				platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
		}
		result, err := a.steward.Confirm(r.Context(), parts[0], body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "steward_confirm_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, mapTaskResponse(result.AuditEntry.ID, string(result.Suggestion.Status), result.AuditEntry.Message))
	case "dismiss":
		var body steward.DismissRequest
		if r.ContentLength != 0 {
			if err := decodeJSON(r, &body); err != nil {
				platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
		}
		suggestion, err := a.steward.Dismiss(r.Context(), parts[0], body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "steward_dismiss_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, mapTaskResponse(suggestion.ID, string(suggestion.Status), "suggestion dismissed"))
	default:
		platform.WriteError(w, r, http.StatusNotFound, "steward_route_not_found", "steward suggestion route not found")
	}
}

func (a *API) stewardAudit(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapStewardAudit(a.steward.Audit(r.Context())))
}

func (a *API) stewardAuditByID(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/steward/audit/"), "/"), "/")
	if len(parts) != 2 || parts[1] != "rollback" {
		platform.WriteError(w, r, http.StatusNotFound, "steward_audit_route_not_found", "steward audit route not found")
		return
	}
	var body steward.RollbackRequest
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
	}
	entry, err := a.steward.Rollback(r.Context(), parts[0], body)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "steward_rollback_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapTaskResponse(entry.ID, string(entry.Result), entry.Message))
}

func (a *API) securityIdentities(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	identities, err := a.security.Identities(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "security_identities_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, identities)
}

func (a *API) securityIdentityByID(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPut) {
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/security/identities/"), "/"), "/")
	if len(parts) != 2 || parts[1] != "permissions" {
		platform.WriteError(w, r, http.StatusNotFound, "security_identity_route_not_found", "security identity route not found")
		return
	}
	var body security.IdentityPermissions
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	identity, err := a.security.UpdateIdentityPermissions(r.Context(), parts[0], body)
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "security_identity_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, identity)
}

func (a *API) securityAIPolicies(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	policies, err := a.security.AiPolicies(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "security_ai_policies_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, policies)
}

func (a *API) securityAIPolicyByID(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPut) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/security/ai-policies/"), "/")
	var body security.AiPolicyUpdate
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	policy, err := a.security.UpdateAiPolicy(r.Context(), id, body)
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "security_ai_policy_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, policy)
}

func (a *API) securityInspect(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	actions, err := a.security.Inspect(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "security_inspect_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapSecurityRiskActions(actions))
}

func (a *API) securityRiskActions(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	actions, err := a.security.RiskActions(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "security_risks_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapSecurityRiskActions(actions))
}

func (a *API) securityRiskActionByID(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/security/risk-actions/"), "/"), "/")
	if len(parts) != 2 {
		platform.WriteError(w, r, http.StatusNotFound, "security_risk_route_not_found", "security risk action route not found")
		return
	}
	var body struct {
		ActorID string `json:"actorId"`
		Reason  string `json:"reason"`
	}
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
	}
	var (
		action security.SecurityRiskAction
		err    error
	)
	switch parts[1] {
	case "confirm":
		action, err = a.security.ConfirmRiskAction(r.Context(), parts[0], body.ActorID)
	case "block":
		action, err = a.security.BlockRiskAction(r.Context(), parts[0], body.ActorID, body.Reason)
	default:
		platform.WriteError(w, r, http.StatusNotFound, "security_risk_route_not_found", "security risk action route not found")
		return
	}
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "security_risk_action_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapTaskResponse(action.ID, string(action.State), action.Title))
}

func (a *API) securityAudit(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	entries, err := a.security.Audit(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "security_audit_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapSecurityAudit(entries))
}

func (a *API) securityHostPorts(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	ports, err := a.security.ScanPorts(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "security_ports_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, ports)
}

func (a *API) securityHostFirewall(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	fw, err := a.security.Firewall(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "security_firewall_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, fw)
}

func (a *API) securityHostScan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	result, err := a.security.Scan(r.Context(), actorFromRequest(r))
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "security_scan_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, result)
}

func (a *API) securityAuditByID(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/security/audit/"), "/"), "/")
	if len(parts) != 2 || parts[1] != "rollback" {
		platform.WriteError(w, r, http.StatusNotFound, "security_audit_route_not_found", "security audit route not found")
		return
	}
	var body struct {
		ActorID string `json:"actorId"`
	}
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
	}
	entry, err := a.security.RollbackAudit(r.Context(), parts[0], body.ActorID)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "security_rollback_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapTaskResponse(entry.ID, string(entry.Result), entry.Event))
}

func (a *API) securityShares(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	shares, err := a.security.Shares(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "security_shares_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapSecurityShares(shares))
}

func (a *API) securityShareByID(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodDelete) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/shares/"), "/")
	share, err := a.security.DeleteShare(r.Context(), id, r.Header.Get("X-Actor-ID"))
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "security_share_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapTaskResponse(share.ID, "revoked", "share link revoked"))
}

func mapTaskResponse(id, state, message string) map[string]any {
	return map[string]any{"id": id, "state": state, "message": message}
}

func mapAssistantThread(thread assistant.Thread) map[string]any {
	messages := make([]map[string]any, 0, len(thread.Messages))
	for _, message := range thread.Messages {
		messages = append(messages, mapAssistantMessage(message))
	}
	return map[string]any{
		"id":        thread.ID,
		"title":     thread.Title,
		"preset":    thread.Preset,
		"messages":  messages,
		"createdAt": thread.CreatedAt,
		"updatedAt": thread.UpdatedAt,
	}
}

func mapAssistantMessage(message assistant.Message) map[string]any {
	tools := make([]map[string]any, 0, len(message.Tools))
	for _, t := range message.Tools {
		tools = append(tools, map[string]any{"name": t.Name, "summary": t.Summary})
	}
	return map[string]any{
		"id":              message.ID,
		"role":            message.Role,
		"text":            message.Text,
		"createdAt":       message.CreatedAt,
		"citations":       message.Citations,
		"tools":           tools,
		"pendingActionId": message.ActionID,
	}
}

func mapSemanticSearch(result assistant.SemanticSearchResponse) map[string]any {
	items := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, map[string]any{
			"id":         item.ID,
			"name":       item.Title,
			"title":      item.Title,
			"type":       "document",
			"space":      item.Scope,
			"size":       "",
			"modified":   "",
			"tags":       []string{"semantic"},
			"permission": item.Scope,
			"aiSummary":  item.Snippet,
			"path":       item.Path,
			"snippet":    item.Snippet,
			"score":      item.Score,
		})
	}
	return map[string]any{"answer": result.Answer, "items": items, "citations": result.Citations}
}

func mapStewardSuggestions(suggestions []steward.Suggestion) []map[string]any {
	out := make([]map[string]any, 0, len(suggestions))
	for _, suggestion := range suggestions {
		out = append(out, map[string]any{
			"id":        suggestion.ID,
			"title":     suggestion.Title,
			"detail":    suggestion.Detail,
			"count":     suggestion.Count,
			"risk":      cnRisk(string(suggestion.Risk)),
			"action":    suggestion.Action,
			"status":    suggestion.Status,
			"updatedAt": suggestion.UpdateAt,
		})
	}
	return out
}

func mapStewardAudit(entries []steward.AuditEntry) []map[string]any {
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		out = append(out, map[string]any{
			"id":       entry.ID,
			"event":    entry.Message,
			"actor":    entry.ActorID,
			"risk":     cnRisk(string(entry.Risk)),
			"reverted": entry.Result == steward.AuditRolledBack,
			"rollback": entry.RollbackID,
			"result":   entry.Result,
			"time":     entry.CreatedAt,
		})
	}
	return out
}

func mapSecurityRiskActions(actions []security.SecurityRiskAction) []map[string]any {
	out := make([]map[string]any, 0, len(actions))
	for _, action := range actions {
		out = append(out, map[string]any{
			"id":         action.ID,
			"title":      action.Title,
			"level":      cnRisk(string(action.Level)),
			"levelLabel": action.LevelLabel,
			"scope":      action.Scope,
			"actor":      action.Actor,
			"state":      action.State,
			"confirmed":  action.Confirmed,
			"rollback":   action.Rollback,
		})
	}
	return out
}

func mapSecurityAudit(entries []security.SecurityAuditEntry) []map[string]any {
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		out = append(out, map[string]any{
			"id":        entry.ID,
			"event":     entry.Event,
			"actor":     entry.Actor,
			"risk":      cnRisk(string(entry.Risk)),
			"riskLabel": entry.RiskLabel,
			"reverted":  entry.Reverted,
			"rollback":  entry.Rollback,
			"result":    entry.Result,
			"time":      entry.Time,
		})
	}
	return out
}

func mapSecurityShares(shares []security.ShareLinkRisk) []map[string]any {
	out := make([]map[string]any, 0, len(shares))
	for _, share := range shares {
		out = append(out, map[string]any{
			"id":        share.ID,
			"name":      share.Name,
			"target":    share.Target,
			"access":    share.Access,
			"downloads": share.Downloads,
			"risk":      cnRisk(string(share.Risk)),
			"riskLabel": share.RiskLabel,
			"active":    share.Active,
		})
	}
	return out
}

func cnRisk(value string) string {
	switch strings.ToLower(value) {
	case "low":
		return "低风险"
	case "medium":
		return "中风险"
	case "high":
		return "高风险"
	default:
		return value
	}
}

func intFromQuery(r *http.Request, key string) int {
	value, _ := strconv.Atoi(r.URL.Query().Get(key))
	return value
}
