package model

// ── JSON 文件结构（与 roles_index.json 完全对应）──

// RolesIndex 对应 roles_index.json 根节点
type RolesIndex struct {
	GeneratedAt string             `json:"generated_at"`
	Departments []MarketDepartment `json:"departments"`
}

// MarketDepartment 对应 JSON 第一层：部门及其角色列表
type MarketDepartment struct {
	ID    string       `json:"id"`
	Name  string       `json:"name"`
	Roles []MarketRole `json:"roles"`
}

// MarketRole 对应 JSON 第二层：角色基本信息（含文件路径）
type MarketRole struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Files       MarketRoleFiles `json:"files"`
}

// MarketRoleFiles 角色包的三个文件路径（相对于后端工作目录）
type MarketRoleFiles struct {
	Identity string `json:"identity"`
	Soul     string `json:"soul"`
	Agents   string `json:"agents"`
}

// ── API 响应类型 ──

// MarketRoleListItem API 列表响应：基本信息 + 部门信息（不含文件路径）
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

// MarketDeptSummary API 部门列表项（含角色数量）
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
