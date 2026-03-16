# TrustMesh MVP 设计方案

> 详细设计拆分为以下子文档，本文为总览。

## 项目定位

TrustMesh 是一个**以 AI Agent 为核心执行者**的任务编排与项目管理平台。参考 Asana 的项目管理和任务管理核心模型，但将执行者从人类替换为 AI Agent。

**核心差异：**
- Asana：人通过 UI 交互，领取任务、协作、报告进度
- TrustMesh：Agent 通过 API 交互，自主领取任务、执行、报告结果；人类通过 UI 监督和管理

**关键角色 — 项目经理 Agent：** 每个项目绑定一个「项目经理」Agent。用户提出需求后，与 PM Agent 沟通，由 PM 整理、规划，并将任务/Todo 指派给其他 Agent 执行。

**Agent 通信：** Agent 不调用 REST API，全部通过 NATS 与后端通信。Agent 间不直接通信，一切通过平台中转。

## 技术栈

| 层级 | 技术选型 | 说明 |
|------|---------|------|
| 后端 | Go + Gin | 成熟的 HTTP 框架 |
| 数据库 | MongoDB | Schema 灵活，适合嵌套文档 |
| 消息总线 | NATS | Agent ↔ 后端的统一通信层 |
| 数据驱动 | mongo-driver v2 | Go 官方 MongoDB 驱动 |
| 前端框架 | React + TypeScript + Vite | 现代构建工具链 |
| UI 组件 | shadcn/ui + Tailwind CSS v4 | 高质量可定制组件 |
| 服务端状态 | TanStack Query | 数据获取与缓存 |
| 客户端状态 | Zustand | 轻量级状态管理 |
| HTTP 客户端 | ky | 基于 fetch 的轻量库 |

## MVP 功能范围

## 最小任务闭环

MVP 优先实现一个从需求提出到任务完成的最小闭环：

1. 用户在项目对话中提出需求。
2. 后端保存消息，并通过 NATS 通知该项目的 PM Agent。
3. PM Agent 分析需求，拆分出一个 Task 和多个 Todo，并为每个 Todo 指定执行 Agent。
4. PM Agent 通过 NATS 发布 `task.create` 消息。
5. 后端订阅到 `task.create` 后，写入 MongoDB，并按 Todo 的分派结果向对应 Agent 发布通知。
6. 执行 Agent 收到 Todo 通知后开始执行，并通过 NATS 上报进度、结果或失败。
7. 后端接收执行结果，更新 Todo 状态，并根据全部 Todo 的状态聚合更新 Task 状态。
8. 当全部 Todo 完成时，Task 自动进入 `done`；如存在失败且未恢复，Task 进入 `failed`。

前置条件：
- 项目绑定的 PM Agent 必须存在，且 `role=pm`。
- PM Agent 必须在线，用户才可以开始提出需求。
- Agent 在线状态由其持续发送的心跳决定。

这条闭环是当前文档的主线。人类用户在 MVP 中主要负责提需求、查看任务状态和查看执行结果，不直接创建或拆分任务。

说明：
- 本文中的 `task.create`、`todo.assigned`、`todo.complete` 等名称均为动作简称。
- 完整 NATS subject 以 [消息传输规范](./message-protocol.md) 为准。

### 纳入 MVP

| 模块 | 功能 | 说明 |
|------|------|------|
| 认证 | 用户注册/登录 | JWT Token |
| 项目管理 | CRUD + 归档 + 绑定 PM Agent | 基础项目操作 |
| 对话 | 用户与 PM Agent 对话 | 需求沟通核心入口 |
| 任务管理 | 任务列表/详情 + Todo 列表 + 状态流转 | 服务端聚合状态 |
| 任务拆分 | PM Agent 创建任务并拆分 Todo | 自动化规划 |
| 任务分派 | 服务端按 Todo 分派给执行 Agent | 自动化分发 |
| Agent 注册 | 按 `node_id` 添加/绑定 Agent | 建立平台记录与网络节点映射 |
| Agent 编辑 | 编辑名称、role、描述、能力等 | 用户可维护 Agent 元数据 |
| Agent 执行 | 接收 Todo、执行、报告进度、提交结果 | 全部走 NATS |
| NATS 通信 | Agent 全部通过 NATS 与后端交互 | Pub/Sub + Request-Reply |
| 活动流 | 任务事件历史 | 状态变更、进度报告 |
| 看板视图 | 按状态分列展示任务 | 前端核心视图 |

### 不纳入 MVP（后续版本）

- 团队/组织管理、成员邀请、权限体系
- 自定义字段、自定义工作流
- 文件附件、通知系统（邮件/WebSocket）
- Agent DAG 编排（复杂多步工作流）
- 时间追踪、甘特图、搜索过滤
- 人工手动创建/编辑 Todo 的高级控制台

## 子文档索引

| 文档 | 内容 |
|------|------|
| [数据模型](./data-model.md) | 实体关系、MongoDB 文档结构、索引设计 |
| [API 设计](./api-design.md) | 全部 API 端点、认证方式 |
| [消息传输规范](./message-protocol.md) | NATS subject、消息结构、权限与链路规范 |
| [Agent 引擎](./agent-engine.md) | PM 工作流、Agent 执行流程、NATS 通知机制 |
| [前端设计](./frontend-design.md) | 页面规划、组件树、UI 方案 |
| [项目结构](./project-structure.md) | 目录结构、实施计划、关键设计决策 |
