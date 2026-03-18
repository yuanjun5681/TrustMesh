// ─── API 通用类型 ───

export interface ApiError {
  code: string
  message: string
  details: Record<string, unknown>
}

export interface ApiResponse<T> {
  data: T
}

export interface ApiListMeta {
  count: number
}

export interface ApiListResponse<T> {
  data: {
    items: T[]
  }
  meta: ApiListMeta
}

// ─── 领域模型 ───

export interface User {
  id: string
  email: string
  name: string
  created_at: string
  updated_at: string
}

export interface PMAgentSummary {
  id: string
  name: string
  node_id: string
  status: 'online' | 'offline' | 'busy'
}

export interface Project {
  id: string
  name: string
  description: string
  status: 'active' | 'archived'
  pm_agent: PMAgentSummary
  created_at: string
  updated_at: string
}

export interface ConversationMessage {
  id: string
  role: 'user' | 'pm_agent'
  content: string
  created_at: string
}

export interface TaskSummary {
  id: string
  title: string
  status: TaskStatus
  priority: TaskPriority
  todo_count: number
  completed_todo_count: number
  created_at: string
  updated_at: string
}

export interface ConversationListItem {
  id: string
  project_id: string
  status: 'active' | 'resolved'
  last_message: ConversationMessage
  linked_task: TaskSummary | null
  created_at: string
  updated_at: string
}

export interface ConversationDetail {
  id: string
  project_id: string
  status: 'active' | 'resolved'
  messages: ConversationMessage[]
  linked_task: TaskSummary | null
  created_at: string
  updated_at: string
}

export interface ConversationStreamSnapshot {
  conversation: ConversationDetail
}

export type AgentRole = 'pm' | 'developer' | 'reviewer' | 'custom'
export type AgentStatus = 'online' | 'offline' | 'busy'

export interface AgentUsage {
  project_count: number
  task_count: number
  todo_count: number
  total_count: number
  in_use: boolean
}

export interface Agent {
  id: string
  name: string
  description: string
  role: AgentRole
  capabilities: string[]
  node_id: string
  status: AgentStatus
  last_seen_at: string | null
  usage: AgentUsage
  created_at: string
  updated_at: string
}

export interface TodoAssignee {
  agent_id: string
  name: string
  node_id: string
}

export interface TodoResultArtifactRef {
  artifact_id: string
  kind: string
  label: string
}

export interface TodoResult {
  summary: string
  output: string
  artifact_refs: TodoResultArtifactRef[]
  metadata: Record<string, unknown>
}

export type TaskStatus = 'pending' | 'in_progress' | 'done' | 'failed'
export type TaskPriority = 'low' | 'medium' | 'high' | 'urgent'

export interface Todo {
  id: string
  title: string
  description: string
  status: TaskStatus
  assignee: TodoAssignee
  started_at: string | null
  completed_at: string | null
  failed_at: string | null
  error: string | null
  result: TodoResult
  created_at: string
}

export interface TaskArtifact {
  id: string
  source_todo_id: string | null
  kind: string
  title: string
  uri: string
  mime_type: string | null
  metadata: Record<string, unknown>
}

export interface TaskResult {
  summary: string
  final_output: string
  metadata: Record<string, unknown>
}

export interface TaskListItem {
  id: string
  project_id: string
  conversation_id: string
  title: string
  description: string
  status: TaskStatus
  priority: TaskPriority
  pm_agent: PMAgentSummary
  todo_count: number
  completed_todo_count: number
  failed_todo_count: number
  created_at: string
  updated_at: string
}

export interface TaskDetail {
  id: string
  project_id: string
  conversation_id: string
  title: string
  description: string
  status: TaskStatus
  priority: TaskPriority
  pm_agent: PMAgentSummary
  todos: Todo[]
  artifacts: TaskArtifact[]
  result: TaskResult
  version: number
  created_at: string
  updated_at: string
}

export type TaskEventType =
  | 'task_created'
  | 'task_status_changed'
  | 'todo_assigned'
  | 'todo_started'
  | 'todo_progress'
  | 'todo_completed'
  | 'todo_failed'
  | 'task_comment'

export interface TaskEvent {
  id: string
  task_id: string
  actor_type: 'user' | 'agent' | 'system'
  actor_id: string
  event_type: TaskEventType
  content: string | null
  metadata: Record<string, unknown>
  created_at: string
}

export interface TaskStreamSnapshot {
  task: TaskDetail
  events: TaskEvent[]
}

// ─── 请求类型 ───

export interface AuthRegisterRequest {
  email: string
  name: string
  password: string
}

export interface AuthLoginRequest {
  email: string
  password: string
}

export interface AuthSuccessData {
  token: string
  user: User
}

export interface CreateProjectRequest {
  name: string
  description: string
  pm_agent_id: string
}

export interface UpdateProjectRequest {
  name?: string
  description?: string
}

export interface CreateConversationRequest {
  content: string
}

export interface AppendConversationMessageRequest {
  content: string
}

export interface ListProjectTasksQuery {
  status?: TaskStatus
}

export interface CreateAgentRequest {
  node_id: string
  name: string
  role: AgentRole
  description: string
  capabilities: string[]
}

export interface UpdateAgentRequest {
  name?: string
  role?: AgentRole
  description?: string
  capabilities?: string[]
}
