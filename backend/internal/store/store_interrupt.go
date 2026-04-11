package store

import (
	"fmt"
	"strings"
	"time"

	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/transport"
)

// InterruptTaskByAgent 标记 task 为 interrupted 状态。
//
// 触发场景：Agent 通过 webhook 回报 task.error / todo.error，表示
// 它无法继续处理我们派发的消息（区别于业务执行失败 todo.fail）。
//
// 行为：
//   - 找到 task 中所有 in_progress todo，标记为 interrupted；
//   - 把 task.Status 显式设为 interrupted；
//   - 累加 InterruptCount，记录 InterruptReason / InterruptedAt；
//   - 写入 task_interrupted（以及 todo_interrupted，如有活跃 todo）event；
//   - 通过 SSE 推送给前端。
//
// 终态保护：task 已是 done / canceled / failed 时不改业务状态，仅写 event。
// 已是 interrupted 时累加计数并刷新 reason，便于 UI 提示反复中断。
func (s *Store) InterruptTaskByAgent(nodeID, taskID, errMsg string) (*model.TaskDetail, *transport.AppError) {
	taskID = strings.TrimSpace(taskID)
	errMsg = strings.TrimSpace(errMsg)
	if taskID == "" {
		return nil, transport.Validation("invalid interrupt payload", map[string]any{"task_id": "required"})
	}
	if errMsg == "" {
		errMsg = "agent reported error without message"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	agent, appErr := s.agentByNodeUnsafe(nodeID)
	if appErr != nil {
		return nil, appErr
	}
	task, ok := s.tasks[taskID]
	if !ok || task.UserID != agent.UserID {
		return nil, transport.NotFound("task not found")
	}

	now := time.Now().UTC()
	s.markAgentSeenUnsafe(agent.ID, now)
	prevStatus := task.Status

	reasonCopy := errMsg
	metadata := map[string]any{
		"task_title":     task.Title,
		"reason":         errMsg,
		"from_node_id":   nodeID,
		"from_agent_id":  agent.ID,
		"task_status":    task.Status,
		"interrupt_kind": classifyInterruptKind(task.Status),
	}

	// 终态：仅记录 event，不动业务状态。
	if isTaskTerminal(task.Status) {
		summary := fmt.Sprintf("agent error after task terminal: %s", errMsg)
		s.addEventUnsafe(task.UserID, task.ProjectID, task.ID, "", "agent", agent.ID, agent.Name, "task_interrupted", &summary, metadata, now)
		if err := s.persistTaskBundleUnsafe(task.ID); err != nil {
			return nil, mongoWriteError(err)
		}
		s.publishTaskUnsafe(task.ID)
		return s.copyTaskWithArtifactsUnsafe(task), nil
	}

	// 找到当前活跃 todo（顺序派发保证最多一个 in_progress）
	for i := range task.Todos {
		todo := &task.Todos[i]
		if todo.Status != "in_progress" {
			continue
		}
		todo.Status = "interrupted"
		todo.InterruptedAt = &now
		todoReason := errMsg
		todo.InterruptReason = &todoReason
		todoSummary := fmt.Sprintf("todo interrupted: %s", todo.Title)
		todoMeta := map[string]any{
			"todo_id":       todo.ID,
			"todo_title":    todo.Title,
			"task_title":    task.Title,
			"reason":        errMsg,
			"from_node_id":  nodeID,
			"from_agent_id": agent.ID,
		}
		s.addEventUnsafe(task.UserID, task.ProjectID, task.ID, todo.ID, "agent", agent.ID, agent.Name, "todo_interrupted", &todoSummary, todoMeta, now)
	}

	// 显式设置 task 状态为 interrupted（覆盖聚合结果）。
	if task.InterruptedFrom == nil && prevStatus != "interrupted" {
		prev := prevStatus
		task.InterruptedFrom = &prev
	}
	task.Status = "interrupted"
	task.InterruptedAt = &now
	task.InterruptReason = &reasonCopy
	task.InterruptCount++

	taskSummary := fmt.Sprintf("task interrupted: %s", errMsg)
	metadata["interrupt_count"] = task.InterruptCount
	s.addEventUnsafe(task.UserID, task.ProjectID, task.ID, "", "agent", agent.ID, agent.Name, "task_interrupted", &taskSummary, metadata, now)

	// 复用 updateTaskStatusUnsafe 完成 Result 聚合 + version 自增 +
	// updated_at 刷新。该函数检测到 task.Status == "interrupted" 时
	// 不会再覆盖状态，符合预期。
	s.updateTaskStatusUnsafe(task, now)

	if err := s.persistTaskBundleUnsafe(task.ID); err != nil {
		return nil, mongoWriteError(err)
	}
	s.publishTaskUnsafe(task.ID)
	return s.copyTaskWithArtifactsUnsafe(task), nil
}

// ResumeTaskByUser 把 interrupted task 恢复为可执行状态。
//
// 行为：
//   - 校验 task.Status == "interrupted"；
//   - 把所有 interrupted todo 改回 pending，清除 InterruptedAt / InterruptReason；
//   - 重新聚合 task 状态（pending 或 in_progress）；
//   - 保留 InterruptCount 用于反复中断的 UI 提示；
//   - 写 task_resumed event 并 SSE 推送。
//
// 派发动作（重新发送 todo.assigned）由 handler 层在调用本方法之后处理。
func (s *Store) ResumeTaskByUser(userID, taskID string) (*model.TaskDetail, *transport.AppError) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, transport.Validation("invalid resume payload", map[string]any{"task_id": "required"})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok || task.UserID != userID {
		return nil, transport.NotFound("task not found")
	}
	if appErr := s.ensureTaskProjectActiveUnsafe(task); appErr != nil {
		return nil, appErr
	}
	if task.Status != "interrupted" {
		return nil, transport.Conflict("TASK_NOT_INTERRUPTED", "task is not interrupted")
	}

	now := time.Now().UTC()
	resumeFromPlanning := task.InterruptedFrom != nil && (*task.InterruptedFrom == "planning" || *task.InterruptedFrom == "review")
	resetCount := 0
	if !resumeFromPlanning {
		for i := range task.Todos {
			todo := &task.Todos[i]
			if todo.Status != "interrupted" {
				continue
			}
			todo.Status = "pending"
			todo.InterruptedAt = nil
			todo.InterruptReason = nil
			todo.StartedAt = nil
			resetCount++
		}
	}

	// 清除 task 级别的中断标记，但 InterruptCount 累计保留。
	task.InterruptedAt = nil
	task.InterruptReason = nil
	if task.InterruptedFrom != nil {
		task.Status = *task.InterruptedFrom
		task.InterruptedFrom = nil
	} else {
		// 兼容旧数据：缺少来源状态时按执行阶段恢复。
		task.Status = "pending"
	}

	resumeMsg := fmt.Sprintf("task resumed: %d todo(s) reset", resetCount)
	s.addEventUnsafe(task.UserID, task.ProjectID, task.ID, "", "user", userID, "User", "task_resumed", &resumeMsg, map[string]any{
		"task_title":      task.Title,
		"todos_reset":     resetCount,
		"interrupt_count": task.InterruptCount,
		"resume_target":   task.Status,
	}, now)

	if resumeFromPlanning {
		task.Result = aggregateTaskResult(task.Todos, task.Status)
		task.UpdatedAt = now
		task.Version++
	} else {
		s.updateTaskStatusUnsafe(task, now)
	}

	if err := s.persistTaskBundleUnsafe(task.ID); err != nil {
		return nil, mongoWriteError(err)
	}
	s.publishTaskUnsafe(task.ID)
	return s.copyTaskWithArtifactsUnsafe(task), nil
}

// isTaskTerminal 判断 task 是否处于终态（不再接受状态变更）。
// interrupted 不是终态，可以被 resume 恢复。
func isTaskTerminal(status string) bool {
	switch status {
	case "done", "failed", "canceled":
		return true
	}
	return false
}

// classifyInterruptKind 根据当前 task 状态分类中断场景，便于前端区分 UI。
//   - planning_or_review：规划阶段中断，前端引导"重新发送需求"
//   - execution：执行阶段中断，前端展示"重新执行"按钮
//   - terminal：task 已结束，仅作提示
func classifyInterruptKind(taskStatus string) string {
	switch taskStatus {
	case "planning", "review":
		return "planning_or_review"
	case "done", "failed", "canceled":
		return "terminal"
	default:
		return "execution"
	}
}
