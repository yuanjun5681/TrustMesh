# 外部平台连接设计

本文说明外部平台（如 ClawHire）如何与 TrustMesh 建立账号绑定，以及连接建立后的双向消息交互机制。

## 一、设计目标

1. 用户在外部平台点击"连接 TrustMesh"，自动跳转至 TrustMesh 授权页完成绑定，无需手动填写参数。
2. 连接建立/断开后，TrustMesh 主动通过 ClawSynapse 网络通知外部平台，双方保持对称的绑定记录。
3. 连接关系决定外部任务的路由：收到入站消息时，通过 `(platformNodeID, remoteUserID)` 二元组定位到本地绑定，进而找到负责的 PM Agent。

## 二、核心数据模型

### TrustMesh 侧（`platform_connections` 集合）

```json
{
  "id": "conn_abc",
  "user_id": "usr_123",
  "platform": "clawhire",
  "platform_node_id": "<ClawHire 节点的 ClawSynapse nodeId>",
  "remote_user_id": "<用户在 ClawHire 的账号 ID>",
  "pm_agent_id": "agent_pm_001",
  "linked_at": "2026-04-26T10:00:00Z"
}
```

### 外部平台侧（对称结构，以 ClawHire 为例）

```json
{
  "platform": "trustmesh",
  "platform_node_id": "<TrustMesh 节点的 ClawSynapse nodeId>",
  "local_user_id": "<用户在 ClawHire 的账号 ID>",
  "linked_at": "2026-04-26T10:00:00Z"
}
```

查询 key：`(platform_node_id, local_user_id)`，与 TrustMesh 侧一致。

## 三、连接建立流程

### 3.1 外部平台构造连接链接

外部平台为用户生成如下格式的 URL：

```
https://{trustmesh-host}/connect
  ?platform=clawhire
  &platform_node_id={CLAWHIRE_NODE_ID}
  &remote_user_id={CLAWHIRE_USER_ID}
```

| 参数 | 含义 |
|---|---|
| `platform` | 平台标识，固定为 `clawhire` |
| `platform_node_id` | ClawHire 节点在 ClawSynapse 网络中的 nodeId |
| `remote_user_id` | 发起连接的用户在 ClawHire 的账号 ID |

### 3.2 用户侧操作

1. 用户点击链接，若未登录 TrustMesh，自动跳转至登录页，登录后返回授权页。
2. TrustMesh 授权页 (`/connect`) 展示连接信息，并自动预选第一个可用 PM Agent。
3. 用户确认后点击"授权连接"。

### 3.3 TrustMesh 处理逻辑

```
POST /api/v1/platform-connections
{
  "platform": "clawhire",
  "platform_node_id": "...",
  "remote_user_id": "...",
  "pm_agent_id": "..."
}
```

1. 写入（或更新）`platform_connections` 记录，key 为 `(platform, platform_node_id, user_id)`。
2. 通过 ClawSynapse 向 `platform_node_id` 发送 `clawhire.connection.established`（异步，不阻断响应）。

### 3.4 外部平台处理响应

ClawHire 的 WebhookAdapter 收到 `clawhire.connection.established` 后：

1. 存储对称绑定记录（见第二节）。
2. 更新 UI，将该用户标记为"已连接 TrustMesh"。

### 3.5 完整时序

```
用户点击 ClawHire 中的"连接 TrustMesh"按钮
  ↓
打开浏览器: https://trustmesh.example.com/connect?platform=clawhire&...
  ↓
TrustMesh /connect 页面（需登录）
  ↓ 用户点击"授权连接"
POST /api/v1/platform-connections → 写入本地绑定
  ↓ 异步
ClawSynapse: TrustMesh → ClawHire 节点
  type: clawhire.connection.established
  ↓
ClawHire webhook 处理 → 写入对称绑定
  ↓
ClawHire UI 显示"已连接"
```

## 四、连接断开流程

断开可由 TrustMesh 侧发起（用户在 `/connect` 页或 `/settings` 页点击"断开连接"）：

```
DELETE /api/v1/platform-connections/{platform}/{platformNodeId}
```

1. TrustMesh 删除本地绑定记录。
2. 异步向 `platformNodeId` 发送 `clawhire.connection.removed`。
3. ClawHire 收到后删除对称绑定记录，UI 更新为"未连接"。

> 若需支持由 ClawHire 侧主动断开，ClawHire 可向 TrustMesh 发送自定义消息，TrustMesh 的 webhook handler 收到后执行同等删除逻辑，并无需再发回通知（避免循环）。

## 五、消息协议

所有连接相关消息均使用 `clawhire.*` 前缀，与任务类消息同属一个命名空间，方向由 `From` 字段区分。

### 5.1 `clawhire.connection.established`（出站：TrustMesh → ClawHire）

```json
{
  "trustMeshNodeId": "<TrustMesh 节点的 ClawSynapse nodeId>",
  "remoteUserId": "<用户在 ClawHire 的账号 ID>",
  "linkedAt": "2026-04-26T10:00:00Z"
}
```

ClawHire 应将 `trustMeshNodeId` 存为 `platform_node_id`，`remoteUserId` 存为 `local_user_id`，用于后续任务消息的路由校验。

### 5.2 `clawhire.connection.removed`（出站：TrustMesh → ClawHire）

```json
{
  "trustMeshNodeId": "<TrustMesh 节点的 ClawSynapse nodeId>",
  "remoteUserId": "<用户在 ClawHire 的账号 ID>",
  "removedAt": "2026-04-26T11:00:00Z"
}
```

### 5.3 消息方向总览

| 消息类型 | 方向 | 触发时机 |
|---|---|---|
| `clawhire.connection.established` | TrustMesh → ClawHire | 用户完成授权绑定 |
| `clawhire.connection.removed` | TrustMesh → ClawHire | 用户断开连接 |
| `clawhire.task.awarded` | ClawHire → TrustMesh | ClawHire 将任务分配给该用户 |
| `clawhire.task.started` | TrustMesh → ClawHire | TrustMesh 开始执行任务 |
| `clawhire.progress.reported` | TrustMesh → ClawHire | 执行过程中进度更新 |
| `clawhire.submission.created` | TrustMesh → ClawHire | 任务完成，提交交付物 |
| `clawhire.submission.accepted` | ClawHire → TrustMesh | ClawHire 验收通过 |
| `clawhire.submission.rejected` | ClawHire → TrustMesh | ClawHire 验收拒绝 |

## 六、路由查找机制

TrustMesh 收到 ClawHire 入站消息时，通过以下二元组查找本地绑定：

```
key = (webhook.From, metadata.remoteUserId)
      │                └── ClawHire 用户 ID，从消息 metadata 中取
      └── ClawHire 节点 nodeId，即 webhook payload 的 From 字段
```

对应内存索引：`platformConnByNodeUser[platformNodeID + ":" + remoteUserID]`

**ClawHire 发送任务消息时必须在 metadata 中携带：**

```json
{
  "metadata": {
    "remoteUserId": "<用户在 ClawHire 的账号 ID>"
  }
}
```

缺少此字段将导致 TrustMesh 无法找到绑定，消息被静默丢弃（返回 `status: skipped`）。

## 七、外部平台实现清单

ClawHire（或其他平台）接入 TrustMesh 需完成以下工作：

### 7.1 连接发起

- [ ] 生成连接链接：`https://{trustmesh-host}/connect?platform=clawhire&platform_node_id={NODE_ID}&remote_user_id={USER_ID}`
- [ ] 在平台 UI 提供"连接 TrustMesh"入口，点击后在新标签打开链接

### 7.2 Webhook 处理（接收 TrustMesh 通知）

- [ ] `clawhire.connection.established`：写入平台侧绑定记录，更新用户连接状态
- [ ] `clawhire.connection.removed`：删除平台侧绑定记录，更新用户连接状态

### 7.3 任务消息发送

- [ ] 发送 `clawhire.task.awarded` 时，在 `metadata` 中携带 `remoteUserId`
- [ ] 发送 `clawhire.submission.accepted` / `clawhire.submission.rejected` 时同样携带

### 7.4 Webhook 处理（接收任务状态）

- [ ] `clawhire.task.started`：标记任务已开始执行
- [ ] `clawhire.progress.reported`：更新任务进度
- [ ] `clawhire.submission.created`：收到交付物，触发验收流程

## 八、注意事项

- **连接通知为尽力而为**：`clawhire.connection.established` 异步发送，若 ClawSynapse 暂时不可达，消息会丢失。外部平台可通过轮询 TrustMesh REST API 来校验连接状态（无需认证的公开端点待规划）。
- **一个用户可绑定多个外部平台账号**：`platform_connections` 按 `(platform, platform_node_id)` 去重，同一用户可同时绑定不同平台。
- **PM Agent 必须在线**：绑定时选择的 PM Agent 必须处于在线状态，否则外部任务将无法分发。
- **连接关系不随 PM Agent 变更自动迁移**：修改绑定使用的 PM Agent（重新提交表单）不会影响已创建的任务。
