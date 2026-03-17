package handler

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"trustmesh/backend/internal/clawsynapse"
	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/store"
	"trustmesh/backend/internal/transport"
)

type ConversationHandler struct {
	store     *store.Store
	publisher *clawsynapse.Client
	log       *zap.Logger
}

func NewConversationHandler(s *store.Store, publisher *clawsynapse.Client, log *zap.Logger) *ConversationHandler {
	return &ConversationHandler{store: s, publisher: publisher, log: log}
}

type createConversationRequest struct {
	Content string `json:"content"`
}

type pmConversationMessage struct {
	ConversationID  string                 `json:"conversation_id"`
	ProjectID       string                 `json:"project_id"`
	Content         string                 `json:"content"`
	UserContent     string                 `json:"user_content"`
	IsInitial       bool                   `json:"is_initial_message"`
	Project         *pmConversationProject `json:"project,omitempty"`
	PMBrief         *pmConversationBrief   `json:"pm_brief,omitempty"`
	CandidateAgents []pmConversationAgent  `json:"candidate_agents,omitempty"`
}

type pmConversationProject struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type pmConversationBrief struct {
	Objective                   string `json:"objective"`
	MustClarifyBeforeTaskCreate bool   `json:"must_clarify_before_task_create"`
	MustUseSkill                string `json:"must_use_skill"`
}

type pmConversationAgent struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	NodeID       string   `json:"node_id"`
	Role         string   `json:"role"`
	Status       string   `json:"status"`
	Capabilities []string `json:"capabilities"`
}

func (h *ConversationHandler) Create(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req createConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_REQUEST", "invalid json body"))
		return
	}
	detail, appErr := h.store.CreateConversation(userID, c.Param("projectId"), req.Content)
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	h.notifyPM(userID, detail.ProjectID, detail.ID, req.Content, true)
	transport.WriteData(c, http.StatusCreated, detail)
}

func (h *ConversationHandler) ListByProject(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	items, appErr := h.store.ListConversations(userID, c.Param("projectId"))
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteList(c, items, len(items))
}

func (h *ConversationHandler) Get(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	detail, appErr := h.store.GetConversation(userID, c.Param("id"))
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteData(c, http.StatusOK, detail)
}

func (h *ConversationHandler) AppendMessage(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req createConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_REQUEST", "invalid json body"))
		return
	}
	detail, appErr := h.store.AppendConversationMessage(userID, c.Param("id"), req.Content)
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	h.notifyPM(userID, detail.ProjectID, detail.ID, req.Content, false)
	transport.WriteData(c, http.StatusOK, detail)
}

func (h *ConversationHandler) notifyPM(userID, projectID, conversationID, content string, initial bool) {
	if h.publisher == nil || h.log == nil {
		return
	}
	pmNodeID, appErr := h.store.GetProjectPMNode(userID, projectID)
	if appErr != nil {
		h.log.Warn("skip notify conversation.message", zap.String("project_id", projectID), zap.String("code", appErr.Code), zap.String("message", appErr.Message))
		return
	}
	payload := h.buildPMConversationMessage(userID, projectID, conversationID, content, initial)
	if _, err := h.publisher.Publish(context.Background(), pmNodeID, "conversation.message", payload, conversationID, nil); err != nil {
		h.log.Warn("notify conversation.message failed", zap.String("project_id", projectID), zap.String("conversation_id", conversationID), zap.Error(err))
	}
}

func (h *ConversationHandler) buildPMConversationMessage(userID, projectID, conversationID, content string, initial bool) pmConversationMessage {
	payload := pmConversationMessage{
		ConversationID: conversationID,
		ProjectID:      projectID,
		Content:        content,
		UserContent:    content,
		IsInitial:      initial,
	}
	if !initial {
		return payload
	}

	project, appErr := h.store.GetProject(userID, projectID)
	if appErr != nil {
		if h.log != nil {
			h.log.Warn("build initial pm prompt missing project context", zap.String("project_id", projectID), zap.String("code", appErr.Code), zap.String("message", appErr.Message))
		}
		return payload
	}

	candidates := buildCandidateAgents(project.PMAgent.ID, h.store.ListAgents(userID))
	payload.Project = &pmConversationProject{
		Name:        project.Name,
		Description: project.Description,
	}
	payload.PMBrief = defaultPMBrief()
	payload.CandidateAgents = candidates
	payload.Content = buildInitialPMContent(project, content, candidates)
	return payload
}

func defaultPMBrief() *pmConversationBrief {
	return &pmConversationBrief{
		Objective:                   "明确任务目标和业务目的；在需求清晰前持续澄清；仅在需求满足执行条件后拆解任务并派发给其他 Agent。",
		MustClarifyBeforeTaskCreate: true,
		MustUseSkill:                "trustmesh",
	}
}

func buildCandidateAgents(pmAgentID string, agents []model.Agent) []pmConversationAgent {
	out := make([]pmConversationAgent, 0, len(agents))
	for _, agent := range agents {
		if agent.ID == pmAgentID {
			continue
		}
		out = append(out, pmConversationAgent{
			ID:           agent.ID,
			Name:         agent.Name,
			NodeID:       agent.NodeID,
			Role:         agent.Role,
			Status:       agent.Status,
			Capabilities: append([]string(nil), agent.Capabilities...),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		left := agentStatusRank(out[i].Status)
		right := agentStatusRank(out[j].Status)
		if left != right {
			return left < right
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func agentStatusRank(status string) int {
	switch status {
	case "online":
		return 0
	case "busy":
		return 1
	default:
		return 2
	}
}

func buildInitialPMContent(project *model.Project, content string, candidates []pmConversationAgent) string {
	var sb strings.Builder
	sb.WriteString("你正在处理一次新的需求沟通。\n\n")
	sb.WriteString("任务：")
	sb.WriteString(content)
	sb.WriteString("\n")
	sb.WriteString("目标：先明确任务目标、业务目的、范围、约束和验收标准；如果有不明确之处，先向用户提问澄清。只有在需求足够明确后，才创建任务并拆解 Todo。\n\n")
	sb.WriteString("执行要求：\n")
	sb.WriteString("1. 先复述你的理解，并指出缺失信息、歧义或风险。\n")
	sb.WriteString("2. 未明确前，不要创建 task，也不要拆分 todo。\n")
	sb.WriteString("3. 明确后，使用 `trustmesh` skill 创建 1 个 Task，并拆成边界清晰、可独立验收的 Todos。\n")
	sb.WriteString("4. 使用 `trustmesh` skill 发送 `conversation.reply` 回复用户。\n")
	sb.WriteString("5. 结合候选 Agent 的 role、status、capabilities 进行分派，不要虚构能力。\n\n")
	sb.WriteString("项目上下文：\n")
	sb.WriteString("- 项目名称：")
	sb.WriteString(project.Name)
	sb.WriteString("\n")
	if strings.TrimSpace(project.Description) != "" {
		sb.WriteString("- 项目描述：")
		sb.WriteString(project.Description)
		sb.WriteString("\n")
	}
	sb.WriteString("- 当前项目 PM：")
	sb.WriteString(project.PMAgent.Name)
	sb.WriteString(" (")
	sb.WriteString(project.PMAgent.NodeID)
	sb.WriteString(")\n")

	sb.WriteString("候选执行 Agent：\n")
	if len(candidates) == 0 {
		sb.WriteString("- 暂无可用于派发的其他 Agent。如需执行，请先说明阻塞。\n")
	} else {
		for _, agent := range candidates {
			sb.WriteString("- ")
			sb.WriteString(agent.Name)
			sb.WriteString(" | node_id=")
			sb.WriteString(agent.NodeID)
			sb.WriteString(" | role=")
			sb.WriteString(agent.Role)
			sb.WriteString(" | status=")
			sb.WriteString(agent.Status)
			if len(agent.Capabilities) > 0 {
				sb.WriteString(" | capabilities=")
				sb.WriteString(strings.Join(agent.Capabilities, ", "))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n用户原始需求：\n")
	sb.WriteString(content)
	return sb.String()
}
