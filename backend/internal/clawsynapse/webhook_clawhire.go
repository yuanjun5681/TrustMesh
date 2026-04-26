package clawsynapse

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/protocol"
	"trustmesh/backend/internal/store"
	"trustmesh/backend/internal/transport"
)

// ── Inbound: ClawHire → TrustMesh ─────────────────────────────────────────

func (h *WebhookHandler) handleClawHireTaskAwarded(c *gin.Context, webhook protocol.WebhookPayload) {
	var payload protocol.ClawHireTaskAwardedPayload
	if err := decodeWebhookMessage(webhook.Message, &payload); err != nil || payload.TaskID == "" {
		transport.WriteError(c, transport.BadRequest("BAD_PAYLOAD", "invalid clawhire.task.awarded message"))
		return
	}

	remoteUserID := remoteUserIDFromMetadata(webhook.Metadata)
	if remoteUserID == "" {
		transport.WriteError(c, transport.BadRequest("BAD_PAYLOAD", "missing remoteUserId in metadata"))
		return
	}

	conn, appErr := h.store.LookupPlatformConnection(webhook.From, remoteUserID)
	if appErr != nil {
		if h.log != nil {
			h.log.Warn("clawhire.task.awarded: no platform connection",
				zap.String("from_node", webhook.From),
				zap.String("remote_user_id", remoteUserID),
			)
		}
		// Graceful degradation: binding not found, skip silently
		transport.WriteData(c, http.StatusOK, gin.H{"status": "skipped", "reason": "no_binding"})
		return
	}

	projectID, appErr := h.store.EnsureClawHireProject(conn.UserID, conn)
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}

	desc := strings.TrimSpace(payload.Description)
	if desc == "" {
		desc = payload.Title
	}

	task, appErr := h.store.CreateExternalTask(conn.UserID, store.ExternalTaskCreateInput{
		ProjectID:   projectID,
		Title:       payload.Title,
		Description: desc,
		ExternalRef: model.ExternalTaskRef{
			Platform:       "clawhire",
			ExternalTaskID: payload.TaskID,
			RemoteUserID:   remoteUserID,
			PlatformNodeID: webhook.From,
			ContractID:     payload.ContractID,
		},
	})
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}

	h.publishClawHirePMTaskMessage(context.Background(), conn.UserID, task)
	transport.WriteData(c, http.StatusOK, gin.H{"status": "ok", "task_id": task.ID})
}

func (h *WebhookHandler) handleClawHireSubmissionAccepted(c *gin.Context, webhook protocol.WebhookPayload) {
	var payload protocol.ClawHireSubmissionAcceptedPayload
	if err := decodeWebhookMessage(webhook.Message, &payload); err != nil || payload.TaskID == "" {
		transport.WriteError(c, transport.BadRequest("BAD_PAYLOAD", "invalid clawhire.submission.accepted message"))
		return
	}

	detail := payload.SubmissionID
	if detail == "" {
		detail = "submission accepted"
	}
	if appErr := h.store.RecordExternalPlatformEvent("clawhire", payload.TaskID, "clawhire_submission_accepted", detail); appErr != nil {
		if h.log != nil {
			h.log.Warn("clawhire.submission.accepted: task not found", zap.String("external_task_id", payload.TaskID))
		}
		transport.WriteData(c, http.StatusOK, gin.H{"status": "skipped", "reason": "task_not_found"})
		return
	}
	transport.WriteData(c, http.StatusOK, gin.H{"status": "ok"})
}

func (h *WebhookHandler) handleClawHireSubmissionRejected(c *gin.Context, webhook protocol.WebhookPayload) {
	var payload protocol.ClawHireSubmissionRejectedPayload
	if err := decodeWebhookMessage(webhook.Message, &payload); err != nil || payload.TaskID == "" {
		transport.WriteError(c, transport.BadRequest("BAD_PAYLOAD", "invalid clawhire.submission.rejected message"))
		return
	}

	detail := payload.Reason
	if detail == "" {
		detail = "submission rejected"
	}
	if appErr := h.store.RecordExternalPlatformEvent("clawhire", payload.TaskID, "clawhire_submission_rejected", detail); appErr != nil {
		if h.log != nil {
			h.log.Warn("clawhire.submission.rejected: task not found", zap.String("external_task_id", payload.TaskID))
		}
		transport.WriteData(c, http.StatusOK, gin.H{"status": "skipped", "reason": "task_not_found"})
		return
	}
	transport.WriteData(c, http.StatusOK, gin.H{"status": "ok"})
}

// ── Outbound: TrustMesh → ClawHire ────────────────────────────────────────

// NotifyClawHireSubmission publishes clawhire.submission.created when a task reaches
// "done" status on a task that originated from ClawHire.
func (h *WebhookHandler) NotifyClawHireSubmission(ctx context.Context, task *model.TaskDetail) {
	if task.ExternalRef == nil || task.ExternalRef.Platform != "clawhire" {
		return
	}
	ref := task.ExternalRef
	h.publish(ctx, ref.PlatformNodeID, "clawhire.submission.created", protocol.ClawHireSubmissionCreatedPayload{
		TaskID:      ref.ExternalTaskID,
		ContractID:  ref.ContractID,
		Summary:     task.Result.Summary,
		SubmittedAt: time.Now().UTC().Format(time.RFC3339),
	}, ref.ExternalTaskID)
}

// NotifyClawHireTaskStarted publishes clawhire.task.started when the first execution
// agent is dispatched on a task that originated from ClawHire.
func (h *WebhookHandler) NotifyClawHireTaskStarted(ctx context.Context, task *model.TaskDetail) {
	if task.ExternalRef == nil || task.ExternalRef.Platform != "clawhire" {
		return
	}
	ref := task.ExternalRef
	h.publish(ctx, ref.PlatformNodeID, "clawhire.task.started", protocol.ClawHireTaskStartedPayload{
		TaskID:     ref.ExternalTaskID,
		ContractID: ref.ContractID,
		StartedAt:  time.Now().UTC().Format(time.RFC3339),
	}, ref.ExternalTaskID)
}

// NotifyClawHireProgress publishes clawhire.progress.reported when a todo reports
// progress on a task that originated from ClawHire.
func (h *WebhookHandler) NotifyClawHireProgress(ctx context.Context, task *model.TaskDetail, message string) {
	if task.ExternalRef == nil || task.ExternalRef.Platform != "clawhire" {
		return
	}
	ref := task.ExternalRef
	h.publish(ctx, ref.PlatformNodeID, "clawhire.progress.reported", protocol.ClawHireProgressReportedPayload{
		TaskID:     ref.ExternalTaskID,
		ContractID: ref.ContractID,
		Summary:    message,
		ReportedAt: time.Now().UTC().Format(time.RFC3339),
	}, ref.ExternalTaskID)
}

// ── Helpers ────────────────────────────────────────────────────────────────

func (h *WebhookHandler) publishClawHirePMTaskMessage(ctx context.Context, userID string, task *model.TaskDetail) {
	if h.client == nil {
		return
	}
	project, appErr := h.store.GetProject(userID, task.ProjectID)
	if appErr != nil {
		if h.log != nil {
			h.log.Warn("clawhire task.awarded: cannot load project for PM message",
				zap.String("project_id", task.ProjectID), zap.Error(appErr))
		}
		return
	}

	candidates := buildCandidateAgentsFromStore(project.PMAgent.ID, h.store.ListAgents(userID), true)
	payload := protocol.PMTaskMessage{
		SchemaVersion: "1.0",
		TaskID:        task.ID,
		ProjectID:     task.ProjectID,
		Content:       "请使用 /tm-task-plan skill 处理本次需求。",
		UserContent:   task.Description,
		IsInitial:     true,
		Project: &protocol.PMTaskProject{
			Name:        project.Name,
			Description: project.Description,
		},
		CandidateAgents: candidates,
	}

	if _, err := h.client.Publish(ctx, task.PMAgent.NodeID, "task.message", payload, task.ID, nil); err != nil && h.log != nil {
		h.log.Warn("clawhire task.awarded: publish task.message failed",
			zap.String("task_id", task.ID), zap.Error(err))
	}
}

func buildCandidateAgentsFromStore(pmAgentID string, agents []model.Agent, includePMSelf bool) []protocol.PMTaskAgent {
	out := make([]protocol.PMTaskAgent, 0, len(agents))
	for _, agent := range agents {
		if agent.ID == pmAgentID && !includePMSelf {
			continue
		}
		out = append(out, protocol.PMTaskAgent{
			ID:           agent.ID,
			Name:         agent.Name,
			NodeID:       agent.NodeID,
			Role:         agent.Role,
			Status:       agent.Status,
			Capabilities: append([]string(nil), agent.Capabilities...),
		})
	}
	return out
}

func remoteUserIDFromMetadata(metadata map[string]any) string {
	if metadata == nil {
		return ""
	}
	for _, key := range []string{"remoteUserId", "remote_user_id"} {
		if v, ok := metadata[key].(string); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
