#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080/api/v1}"
WEBHOOK_URL="${WEBHOOK_URL:-http://127.0.0.1:8080/webhook/clawsynapse}"
CLAWSYNAPSE_NODE_ID="${CLAWSYNAPSE_NODE_ID:-trustmesh-server}"
SUFFIX="$(date +%s)"

require() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

api() {
  local method="$1"
  local path="$2"
  local token="${3:-}"
  local payload="${4:-}"
  local cmd=(curl -sS -X "$method" "$BASE_URL$path")
  if [[ -n "$token" ]]; then
    cmd+=(-H "Authorization: Bearer $token")
  fi
  if [[ -n "$payload" ]]; then
    cmd+=(-H "Content-Type: application/json" -d "$payload")
  fi
  "${cmd[@]}"
}

publish() {
  local subject="$1"
  local node_id="$2"
  local message_id="$3"
  local payload="$4"
  IFS='.' read -r _ _ domain action <<<"$subject"
  local msg_type="$domain.$action"
  local webhook_payload
  webhook_payload="$(jq -nc \
    --arg nodeId "$CLAWSYNAPSE_NODE_ID" \
    --arg type "$msg_type" \
    --arg from "$node_id" \
    --arg message "$payload" \
    --arg messageId "$message_id" \
    '{nodeId:$nodeId,type:$type,from:$from,message:$message,metadata:{messageId:$messageId}}')"
  curl -sS -X POST "$WEBHOOK_URL" -H "Content-Type: application/json" -d "$webhook_payload" >/dev/null
}

assert_json() {
  local raw="$1"
  local expr="$2"
  local message="$3"
  if ! printf '%s' "$raw" | jq -e "$expr" >/dev/null; then
    echo "assertion failed: $message" >&2
    printf '%s\n' "$raw" | jq .
    exit 1
  fi
}

require curl
require jq

register_payload="$(jq -nc --arg email "smoke-progress-fail-$SUFFIX@example.com" '{email:$email,name:"Smoke User",password:"secret123"}')"
register_resp="$(api POST /auth/register "" "$register_payload")"
token="$(printf '%s' "$register_resp" | jq -r '.data.token')"

pm_node_id="node-pm-progress-fail-$SUFFIX"
dev_node_id="node-dev-progress-fail-$SUFFIX"

pm_payload="$(jq -nc --arg node_id "$pm_node_id" '{node_id:$node_id,name:"PM Smoke",role:"pm",description:"pm",capabilities:["plan"]}')"
pm_resp="$(api POST /agents "$token" "$pm_payload")"
pm_id="$(printf '%s' "$pm_resp" | jq -r '.data.id')"

dev_payload="$(jq -nc --arg node_id "$dev_node_id" '{node_id:$node_id,name:"Dev Smoke",role:"developer",description:"dev",capabilities:["backend"]}')"
dev_resp="$(api POST /agents "$token" "$dev_payload")"
dev_id="$(printf '%s' "$dev_resp" | jq -r '.data.id')"

project_payload="$(jq -nc --arg pm_agent_id "$pm_id" '{name:"Smoke Project",description:"smoke",pm_agent_id:$pm_agent_id}')"
project_resp="$(api POST /projects "$token" "$project_payload")"
project_id="$(printf '%s' "$project_resp" | jq -r '.data.id')"

conversation_resp="$(api POST "/projects/$project_id/conversations" "$token" '{"content":"Need login flow"}')"
conversation_id="$(printf '%s' "$conversation_resp" | jq -r '.data.id')"

task_create_payload="$(jq -nc \
  --arg project_id "$project_id" \
  --arg conversation_id "$conversation_id" \
  --arg assignee_node_id "$dev_node_id" \
  '{
    project_id:$project_id,
    conversation_id:$conversation_id,
    title:"Implement login",
    description:"Support email and password login",
    todos:[
      {
        id:"todo-1",
        title:"Build backend API",
        description:"Implement auth endpoints",
        assignee_node_id:$assignee_node_id
      }
    ]
  }')"
publish "agent.$pm_node_id.task.create" "$pm_node_id" "smoke-task-create-$SUFFIX" "$task_create_payload"
sleep 1

tasks_resp="$(api GET "/projects/$project_id/tasks" "$token")"
task_id="$(printf '%s' "$tasks_resp" | jq -r '.data.items[0].id')"

todo_progress_payload="$(jq -nc --arg task_id "$task_id" '{
  task_id:$task_id,
  todo_id:"todo-1",
  message:"JWT validation done, OAuth blocked by missing credentials"
}')"
publish "agent.$dev_node_id.todo.progress" "$dev_node_id" "smoke-todo-progress-$SUFFIX" "$todo_progress_payload"
sleep 1

progress_task_resp="$(api GET "/tasks/$task_id" "$token")"
assert_json "$progress_task_resp" '.data.status == "in_progress"' 'task should be in_progress after todo.progress'
assert_json "$progress_task_resp" '.data.todos[0].status == "in_progress"' 'todo should be in_progress after todo.progress'

todo_fail_payload="$(jq -nc --arg task_id "$task_id" '{
  task_id:$task_id,
  todo_id:"todo-1",
  error:"Google OAuth credentials missing"
}')"
publish "agent.$dev_node_id.todo.fail" "$dev_node_id" "smoke-todo-fail-$SUFFIX" "$todo_fail_payload"
sleep 1

failed_task_resp="$(api GET "/tasks/$task_id" "$token")"
events_resp="$(api GET "/tasks/$task_id/events" "$token")"

assert_json "$failed_task_resp" '.data.status == "failed"' 'task should be failed after todo.fail'
assert_json "$failed_task_resp" '.data.todos[0].status == "failed"' 'todo should be failed after todo.fail'
assert_json "$failed_task_resp" '.data.todos[0].error == "Google OAuth credentials missing"' 'todo error should be persisted'
assert_json "$failed_task_resp" '.data.result.metadata.failed_todo_count == 1' 'failed todo count should be aggregated'
assert_json "$events_resp" '[.data.items[].event_type] | index("todo_started") != null' 'todo_started event should exist'
assert_json "$events_resp" '[.data.items[].event_type] | index("todo_progress") != null' 'todo_progress event should exist'
assert_json "$events_resp" '[.data.items[].event_type] | index("todo_failed") != null' 'todo_failed event should exist'

echo "smoke progress/fail flow passed"
echo "user=$(printf '%s' "$register_resp" | jq -c '{id:.data.user.id,email:.data.user.email}')"
echo "pm=$(printf '%s' "$pm_resp" | jq -c '{id:.data.id,node_id:.data.node_id,status:.data.status}')"
echo "dev=$(printf '%s' "$dev_resp" | jq -c '{id:.data.id,node_id:.data.node_id,status:.data.status}')"
echo "project=$(printf '%s' "$project_resp" | jq -c '{id:.data.id,status:.data.status}')"
echo "conversation=$(printf '%s' "$conversation_resp" | jq -c '{id:.data.id,status:.data.status,message_count:(.data.messages|length)}')"
echo "task=$(printf '%s' "$failed_task_resp" | jq -c '{id:.data.id,status:.data.status,todo_status:.data.todos[0].status,error:.data.todos[0].error,result_summary:.data.result.summary}')"
echo "events=$(printf '%s' "$events_resp" | jq -c '[.data.items[].event_type]')"
