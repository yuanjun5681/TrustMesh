# TrustMesh 工作岗位市场 — 技术实现方案

> 版本：v1.0  
> 日期：2026-04-08  
> 基于设计文档：`TrustMesh工作市场模块设计文档.md`

---

## 一、关键设计决策

| 决策 | 方案 | 理由 |
|------|------|------|
| 角色数据存储 | 内存（启动时加载） | 191 个静态角色，约 200KB，无需 DB |
| 索引缓存 | `backend/data/roles_index.json` | 预计算加速启动，便于调试和查阅 |
| MarketStore | 独立结构体，不挂主 Store | 职责隔离，未来可独立替换为 DB |
| 下载端点 | 流式构造 zip | 无临时文件，减少磁盘 I/O |
| 鉴权 | 挂在 `authed` 分组 | 与平台其他接口一致 |
| 安装方式 | 详情页展示手动安装说明 | 第一阶段，不做 OpenClaw 自动安装 |

---

## 二、部门分类映射

基于 `agency-agents-zh` 角色目录命名前缀划分（优先匹配双段前缀）：

| 目录前缀（一段或两段） | DeptID | 中文名 |
|----------------------|--------|-------|
| `academic` | academic | 学术部 |
| `engineering` | engineering | 工程部 |
| `design` | design | 设计部 |
| `marketing` | marketing | 营销部 |
| `product` | product | 产品部 |
| `project-management`, `project-manager` | project-management | 项目管理部 |
| `testing` | testing | 测试部 |
| `support` | support | 支持部 |
| `specialized`, `agents`, `prompt` | specialized | 专项部 |
| `game`, `godot`, `unity`, `unreal`, `roblox`, `blender`, `xr`, `visionos`, `level`, `narrative` | creative-tech | 创意技术部 |
| `finance` | finance | 金融部 |
| `hr` | hr | HR部 |
| `legal` | legal | 法务部 |
| `sales`, `paid-media` | sales-marketing | 销售与营销部 |
| `supply-chain` | supply-chain | 供应链部 |
| 其余所有前缀 | other | 其他 |

**前缀解析规则**：
1. 先尝试目录名前两段作为前缀（如 `paid-media`、`supply-chain`、`project-management`）
2. 找不到再回退单段前缀（如 `engineering`）
3. 仍找不到则归入 `other`

---

## 三、JSON 索引文件格式

文件路径：`backend/data/roles_index.json`

```json
{
  "generated_at": "2026-04-08T00:00:00Z",
  "departments": [
    {
      "id": "engineering",
      "name": "工程部",
      "roles": [
        {
          "id": "engineering-backend-architect",
          "name": "后端架构师",
          "description": "资深后端架构师，专精可扩展系统设计、数据库架构、API 开发和云基础设施。",
          "files": {
            "identity": "data/roles/engineering-backend-architect/IDENTITY.md",
            "soul":     "data/roles/engineering-backend-architect/SOUL.md",
            "agents":   "data/roles/engineering-backend-architect/AGENTS.md"
          }
        }
      ]
    },
    {
      "id": "marketing",
      "name": "营销部",
      "roles": [ ... ]
    }
  ]
}
```

**加载策略**：
- 启动时优先读取 `roles_index.json`（快速路径）
- 若文件不存在或损坏，自动扫描 `data/roles/` 目录重新生成
- 提供 `RebuildIndex()` 方法供手动触发（可通过管理接口暴露）

---

## 四、后端实现

### 4.1 新增文件

#### `backend/internal/model/market.go`

Go 结构体直接镜像 JSON 文件格式，`json.Unmarshal` 即可完成加载，不需要额外解析逻辑：

```go
package model

// ── JSON 文件结构（与 roles_index.json 完全一致）──

// RolesIndex 对应 roles_index.json 根节点
type RolesIndex struct {
    GeneratedAt string             `json:"generated_at"`
    Departments []MarketDepartment `json:"departments"`
}

// MarketDepartment 对应 JSON 第一层：部门及其角色列表
type MarketDepartment struct {
    ID    string       `json:"id"`    // 如 "engineering"
    Name  string       `json:"name"`  // 如 "工程部"
    Roles []MarketRole `json:"roles"`
}

// MarketRole 对应 JSON 第二层：角色基本信息（含文件路径）
type MarketRole struct {
    ID          string          `json:"id"`
    Name        string          `json:"name"`
    Description string          `json:"description"`
    Files       MarketRoleFiles `json:"files"`
}

// MarketRoleFiles 角色包的三个文件路径
type MarketRoleFiles struct {
    Identity string `json:"identity"`
    Soul     string `json:"soul"`
    Agents   string `json:"agents"`
}

// ── API 响应类型（在 JSON 结构基础上补充 dept 信息和文件内容）──

// MarketRoleListItem API 列表响应项：角色基本信息 + 部门信息（从父节点提升）
type MarketRoleListItem struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
    DeptID      string `json:"dept_id"`
    DeptName    string `json:"dept_name"`
}

// MarketRoleDetail API 详情响应：基本信息 + 部门信息 + 三个文件内容
type MarketRoleDetail struct {
    MarketRoleListItem
    IdentityContent string `json:"identity_content"`
    SoulContent     string `json:"soul_content"`
    AgentsContent   string `json:"agents_content"`
}

// MarketDeptSummary API 部门列表响应（含角色数量）
type MarketDeptSummary struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Count int    `json:"count"`
}

// MarketRoleFilter 列表过滤参数
type MarketRoleFilter struct {
    DeptID string // 按部门过滤，空表示全部
    Query  string // 关键词（匹配 name 和 description，大小写不敏感）
}
```

---

#### `backend/internal/store/store_market.go`

核心结构：

```go
type MarketStore struct {
    mu    sync.RWMutex
    dataDir string                       // 指向 backend/data/
    index *model.RolesIndex              // 完整 JSON 索引（两层结构，json.Unmarshal 直接填充）
    roles map[string]*model.MarketRole   // id → 角色（含文件路径，方便 O(1) 查找）
    depts map[string]*model.MarketDepartment // deptID → 部门（含角色列表）
}
```

主要方法：

| 方法 | 说明 |
|------|------|
| `NewMarketStore(dataDir)` | 优先加载 JSON 索引，否则扫描目录 |
| `RebuildIndex()` | 重新扫描目录并写入 `roles_index.json` |
| `ListCategories()` | 返回有角色的部门 |
| `ListRoles(filter)` | 按部门+关键词过滤 |
| `GetRole(id)` | 读取文件内容返回 Detail |
| `WriteRoleZip(w, id)` | 流式打包 zip |

关键实现细节：

```go
// WriteRoleZip：响应头必须在 zip.NewWriter 之前设置
func (ms *MarketStore) WriteRoleZip(w http.ResponseWriter, id string) *transport.AppError {
    entry, ok := ms.roles[id]
    if !ok {
        return transport.NotFound("ROLE_NOT_FOUND", "role not found")
    }
    w.Header().Set("Content-Type", "application/zip")
    w.Header().Set("Content-Disposition", `attachment; filename="`+id+`.zip"`)
    w.WriteHeader(http.StatusOK)
    
    zw := zip.NewWriter(w)
    defer zw.Close()
    
    for _, filePath := range []string{entry.Files.Identity, entry.Files.Soul, entry.Files.Agents} {
        // 写入 zip 条目...
    }
    return nil
}
```

---

#### `backend/internal/handler/market.go`

```go
type MarketHandler struct {
    market *store.MarketStore
}

// GET /api/v1/market/categories
func (h *MarketHandler) ListCategories(c *gin.Context)

// GET /api/v1/market/roles?category=engineering&q=后端
func (h *MarketHandler) ListRoles(c *gin.Context)

// GET /api/v1/market/roles/:id
func (h *MarketHandler) GetRole(c *gin.Context)

// GET /api/v1/market/roles/:id/download
func (h *MarketHandler) DownloadRole(c *gin.Context)
```

---

### 4.2 修改文件

**`backend/internal/config/config.go`**
- 新增：`RolesDataPath string`，默认 `"data"`（相对于进程工作目录）

**`backend/internal/app/router.go`**
```go
// 初始化 MarketStore
marketStore, err := store.NewMarketStore(cfg.RolesDataPath)
if err != nil {
    log.Warn("market store init failed", zap.Error(err))
    // 非致命，继续启动（市场功能不可用，但其他功能正常）
}

// 注册路由（authed 分组下）
if marketStore != nil {
    marketHandler := handler.NewMarketHandler(marketStore)
    market := authed.Group("/market")
    market.GET("/categories", marketHandler.ListCategories)
    market.GET("/roles", marketHandler.ListRoles)
    market.GET("/roles/:id", marketHandler.GetRole)
    market.GET("/roles/:id/download", marketHandler.DownloadRole)
}
```

---

## 五、前端实现

### 5.1 新增文件

**`frontend/src/api/market.ts`**

| 函数 | 方法 | 路径 |
|------|------|------|
| `listCategories()` | GET | `market/categories` |
| `listRoles(params?)` | GET | `market/roles?category=&q=` |
| `getRole(id)` | GET | `market/roles/:id` |
| `downloadRole(id)` | GET（blob） | `market/roles/:id/download` |

`downloadRole` 实现：
```typescript
export async function downloadRole(id: string) {
  const blob = await api.get(`market/roles/${id}/download`).blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${id}.zip`
  a.click()
  URL.revokeObjectURL(url)
}
```

**`frontend/src/hooks/useMarket.ts`**
- `useMarketCategories()` — staleTime 5 分钟
- `useMarketRoles(params?)` — staleTime 5 分钟
- `useMarketRole(id)` — staleTime 5 分钟，enabled: !!id

**`frontend/src/pages/MarketPage.tsx`**

布局：
```
┌──────────────────────────────────────────────────────┐
│ 岗位市场           [搜索框...]                         │
├─────────┬────────────────────────────────────────────┤
│ 全部(191)│ 工程部 > 25个结果                           │
│ 工程部 25│ ┌────────┐ ┌────────┐ ┌────────┐          │
│ 营销部 40│ │角色卡片│ │角色卡片│ │角色卡片│          │
│ 设计部  7│ └────────┘ └────────┘ └────────┘          │
│ ...     │ ...                                         │
└─────────┴────────────────────────────────────────────┘
```

状态：
- URL search params 持久化 `category` 和 `q`
- 搜索防抖 300ms
- 点击卡片 → `RoleDetailSheet` 打开

**`frontend/src/components/market/RoleCard.tsx`**
- Badge（部门名 + 颜色映射）
- 角色名（粗体）
- 描述（最多 2 行，超出截断）
- "查看详情"按钮

**`frontend/src/components/market/RoleDetailSheet.tsx`**
- shadcn/ui Sheet（从右侧滑入）
- 顶部：角色名 + 部门 Badge + 下载按钮
- Tabs：简介 / 人格 / 规范（react-markdown 渲染，ScrollArea 包裹）
- 底部：`InstallGuide` 折叠面板

**`frontend/src/components/market/InstallGuide.tsx`**

手动安装步骤说明：
```
1. 解压下载的 .zip 文件
2. 将 {role-id}/ 目录复制到 ~/.openclaw/agents/
   cp -r {role-id} ~/.openclaw/agents/
3. 重启 OpenClaw 网关
4. 在 TrustMesh 中将该智能体添加为团队成员
```

---

### 5.2 修改文件

**`frontend/src/types/index.ts`** — 追加：
```typescript
// 对应 MarketRoleListItem（列表项）
export interface MarketRoleListItem {
  id: string; name: string; description: string
  dept_id: string; dept_name: string
}
// 对应 MarketRoleDetail（详情，含文件内容）
export interface MarketRoleDetail extends MarketRoleListItem {
  identity_content: string; soul_content: string; agents_content: string
}
// 对应 MarketDeptSummary（部门列表）
export interface MarketDeptSummary { id: string; name: string; count: number }
```

**`frontend/src/App.tsx`** — 追加：
```tsx
import { MarketPage } from '@/pages/MarketPage'
// ...
<Route path="/market" element={<MarketPage />} />
```

**`frontend/src/components/layout/Sidebar.tsx`** — 在 knowledge 链接后添加：
```tsx
import { Briefcase } from 'lucide-react'
// ...
<Link to="/market" className={cn(..., isActive('/market') && '...')}>
  <Briefcase className="size-4 shrink-0" />
  {!collapsed && <span>岗位市场</span>}
</Link>
```

---

## 六、实现顺序

1. **生成 `roles_index.json`**：写脚本或手动生成初始版
2. **后端**：model → store_market → handler/market → router 路由注册
3. **后端测试**：curl 验证各接口，重点验证 zip 下载
4. **前端 types + api + hooks**
5. **前端组件**：RoleCard → RoleDetailSheet → InstallGuide
6. **前端页面**：MarketPage 组装，Sidebar + App.tsx 路由
7. **联调**：端到端测试，验证搜索、过滤、下载全流程

---

## 七、API 接口文档

### GET /api/v1/market/categories

**响应**
```json
{
  "data": {
    "items": [
      { "id": "engineering", "name": "工程部", "count": 25 }
    ]
  },
  "meta": { "count": 16 }
}
```

### GET /api/v1/market/roles

**查询参数**
- `category`（可选）：部门 ID，如 `engineering`
- `q`（可选）：关键词搜索

**响应**
```json
{
  "data": {
    "items": [
      {
        "id": "engineering-backend-architect",
        "name": "后端架构师",
        "description": "资深后端架构师，专精可扩展系统设计...",
        "category": "engineering",
        "dept_id": "engineering",
        "dept_name": "工程部"
      }
    ]
  },
  "meta": { "count": 25 }
}
```

### GET /api/v1/market/roles/:id

**响应**：在列表数据基础上增加 `identity_content`、`soul_content`、`agents_content` 三个字段（Markdown 文本）。

### GET /api/v1/market/roles/:id/download

**响应**：`Content-Type: application/zip`，zip 包含：
- `{id}/IDENTITY.md`
- `{id}/SOUL.md`
- `{id}/AGENTS.md`

---

## 八、验证方式

```bash
# 后端编译
cd backend && go build ./cmd/server

# 启动服务
cd backend && go run ./cmd/server

# 获取 token（先登录）
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"xxx","password":"xxx"}' | jq -r '.data.access_token')

# 测试各接口
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/market/categories
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/api/v1/market/roles?category=engineering"
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/api/v1/market/roles?q=后端"
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/market/roles/engineering-backend-architect
curl -H "Authorization: Bearer $TOKEN" -o test.zip \
  http://localhost:8080/api/v1/market/roles/engineering-backend-architect/download
unzip -l test.zip  # 应看到三个文件

# 前端构建验证
cd frontend && npm run build   # 无 TS 错误
cd frontend && npm run dev     # 访问 http://localhost:3000/market
```
