#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080/api/v1}"
WEBHOOK_URL="${WEBHOOK_URL:-http://127.0.0.1:8080/webhook/clawsynapse}"
CLAWSYNAPSE_API_URL="${CLAWSYNAPSE_API_URL:-http://127.0.0.1:18080}"
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
  else
    :
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
    --arg nodeId "$LOCAL_NODE_ID" \
    --arg type "$msg_type" \
    --arg from "$node_id" \
    --arg message "$payload" \
    --arg messageId "$message_id" \
    '{nodeId:$nodeId,type:$type,from:$from,message:$message,metadata:{messageId:$messageId}}')"
  curl -sS -X POST "$WEBHOOK_URL" -H "Content-Type: application/json" -d "$webhook_payload" >/dev/null
}

require curl
require jq

LOCAL_NODE_ID="$(curl -sS "$CLAWSYNAPSE_API_URL/v1/health" | jq -er '.data.self.nodeId')"

register_payload="$(jq -nc --arg email "smoke-$SUFFIX@example.com" '{email:$email,name:"Smoke User",password:"secret123"}')"
register_resp="$(api POST /auth/register "" "$register_payload")"
token="$(printf '%s' "$register_resp" | jq -r '.data.access_token')"

pm_node_id="node-pm-smoke-$SUFFIX"
dev_node_id="node-dev-smoke-$SUFFIX"

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

todo_complete_payload="$(jq -nc --arg task_id "$task_id" '{
  task_id:$task_id,
  todo_id:"todo-1",
  result:{
    summary:"Auth API ready",
    output:"Implemented register/login endpoints",
    artifact_refs:[
      {
        artifact_id:"artifact-login-api",
        kind:"report",
        label:"Login API report"
      }
    ],
    metadata:{
      duration_ms:1200
    }
  }
}')"
publish "agent.$dev_node_id.todo.complete" "$dev_node_id" "smoke-todo-complete-$SUFFIX" "$todo_complete_payload"
sleep 1

task_resp="$(api GET "/tasks/$task_id" "$token")"

echo "smoke flow passed"
echo "user=$(printf '%s' "$register_resp" | jq -c '{id:.data.user.id,email:.data.user.email}')"
echo "pm=$(printf '%s' "$pm_resp" | jq -c '{id:.data.id,node_id:.data.node_id,status:.data.status}')"
echo "dev=$(printf '%s' "$dev_resp" | jq -c '{id:.data.id,node_id:.data.node_id,status:.data.status}')"
echo "project=$(printf '%s' "$project_resp" | jq -c '{id:.data.id,status:.data.status}')"
echo "conversation=$(printf '%s' "$conversation_resp" | jq -c '{id:.data.id,status:.data.status,message_count:(.data.messages|length)}')"
echo "task=$(printf '%s' "$task_resp" | jq -c '{id:.data.id,status:.data.status,todo_status:.data.todos[0].status,artifact_count:(.data.artifacts|length),result_summary:.data.result.summary}')"
