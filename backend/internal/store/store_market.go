package store

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/transport"
)

// 目录前缀 → 部门 ID（优先匹配双段前缀）
var deptPrefixMap = map[string]string{
	// 双段前缀（优先）
	"paid-media":          "sales-marketing",
	"supply-chain":        "supply-chain",
	"project-management":  "project-management",
	"project-manager":     "project-management",
	"technical-artist":    "creative-tech",
	// 单段前缀
	"academic":    "academic",
	"engineering": "engineering",
	"design":      "design",
	"marketing":   "marketing",
	"product":     "product",
	"testing":     "testing",
	"support":     "support",
	"specialized": "specialized",
	"agents":      "specialized",
	"prompt":      "specialized",
	"game":        "creative-tech",
	"godot":       "creative-tech",
	"unity":       "creative-tech",
	"unreal":      "creative-tech",
	"roblox":      "creative-tech",
	"blender":     "creative-tech",
	"xr":          "creative-tech",
	"visionos":    "creative-tech",
	"level":       "creative-tech",
	"narrative":   "creative-tech",
	"finance":     "finance",
	"hr":          "hr",
	"legal":       "legal",
	"sales":       "sales-marketing",
}

// 部门 ID → 中文名称
var deptNameMap = map[string]string{
	"engineering":        "工程部",
	"marketing":          "营销部",
	"design":             "设计部",
	"product":            "产品部",
	"project-management": "项目管理部",
	"testing":            "测试部",
	"support":            "支持部",
	"specialized":        "专项部",
	"creative-tech":      "创意技术部",
	"finance":            "金融部",
	"hr":                 "HR部",
	"legal":              "法务部",
	"sales-marketing":    "销售与营销部",
	"supply-chain":       "供应链部",
	"academic":           "学术部",
	"other":              "其他",
}

// 部门显示顺序
var deptOrder = []string{
	"engineering",
	"marketing",
	"design",
	"product",
	"project-management",
	"testing",
	"support",
	"specialized",
	"creative-tech",
	"finance",
	"hr",
	"legal",
	"sales-marketing",
	"supply-chain",
	"academic",
	"other",
}

// roleEntry 内存中的角色条目（含部门信息）
type roleEntry struct {
	role     *model.MarketRole
	deptID   string
	deptName string
}

// MarketStore 工作岗位市场的数据层，独立于主 Store
type MarketStore struct {
	mu      sync.RWMutex
	dataDir string               // 指向 backend/data/ 目录
	index   *model.RolesIndex    // 完整两层结构（直接对应 JSON）
	byID    map[string]roleEntry // role id → entry（含部门信息，O(1) 查找）
}

// NewMarketStore 创建 MarketStore，优先从 roles_index.json 加载
func NewMarketStore(dataDir string) (*MarketStore, error) {
	ms := &MarketStore{
		dataDir: dataDir,
		byID:    make(map[string]roleEntry),
	}

	indexPath := filepath.Join(dataDir, "roles_index.json")
	if data, err := os.ReadFile(indexPath); err == nil {
		var idx model.RolesIndex
		if err := json.Unmarshal(data, &idx); err == nil {
			ms.index = &idx
			ms.buildByID()
			return ms, nil
		}
	}

	// JSON 不存在或损坏，重新扫描目录
	if err := ms.rebuild(); err != nil {
		return nil, fmt.Errorf("market store init: %w", err)
	}
	return ms, nil
}

// RebuildIndex 重新扫描 roles 目录，更新内存索引并写入 JSON 文件
func (ms *MarketStore) RebuildIndex() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.rebuild()
}

// rebuild 扫描 roles 目录，构建内存索引并持久化到 JSON（调用方持有写锁或初始化时调用）
func (ms *MarketStore) rebuild() error {
	rolesDir := filepath.Join(ms.dataDir, "roles")
	entries, err := os.ReadDir(rolesDir)
	if err != nil {
		return fmt.Errorf("read roles dir %q: %w", rolesDir, err)
	}

	// 按部门 ID 收集角色
	deptRoles := make(map[string][]model.MarketRole)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirName := entry.Name()
		deptID := resolveDeptID(dirName)

		identityPath := filepath.Join(rolesDir, dirName, "IDENTITY.md")
		name, desc := parseIdentityFile(identityPath)
		if name == "" {
			name = dirName // 解析失败时回退到目录名
		}

		role := model.MarketRole{
			ID:          dirName,
			Name:        name,
			Description: desc,
			Files: model.MarketRoleFiles{
				Identity: filepath.Join("data", "roles", dirName, "IDENTITY.md"),
				Soul:     filepath.Join("data", "roles", dirName, "SOUL.md"),
				Agents:   filepath.Join("data", "roles", dirName, "AGENTS.md"),
			},
		}
		deptRoles[deptID] = append(deptRoles[deptID], role)
	}

	// 按 deptOrder 组装有序部门列表
	var departments []model.MarketDepartment
	for _, deptID := range deptOrder {
		roles, ok := deptRoles[deptID]
		if !ok {
			continue
		}
		sort.Slice(roles, func(i, j int) bool { return roles[i].Name < roles[j].Name })
		departments = append(departments, model.MarketDepartment{
			ID:    deptID,
			Name:  deptNameMap[deptID],
			Roles: roles,
		})
		delete(deptRoles, deptID)
	}

	// 剩余未匹配的归入 other
	var otherRoles []model.MarketRole
	for _, roles := range deptRoles {
		otherRoles = append(otherRoles, roles...)
	}
	if len(otherRoles) > 0 {
		sort.Slice(otherRoles, func(i, j int) bool { return otherRoles[i].Name < otherRoles[j].Name })
		departments = append(departments, model.MarketDepartment{
			ID:    "other",
			Name:  deptNameMap["other"],
			Roles: otherRoles,
		})
	}

	idx := &model.RolesIndex{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Departments: departments,
	}

	// 持久化到 JSON
	indexPath := filepath.Join(ms.dataDir, "roles_index.json")
	if data, err := json.MarshalIndent(idx, "", "  "); err == nil {
		_ = os.WriteFile(indexPath, data, 0644)
	}

	ms.index = idx
	ms.buildByID()
	return nil
}

// buildByID 从 index 构建 byID 索引（调用方持有锁）
func (ms *MarketStore) buildByID() {
	ms.byID = make(map[string]roleEntry)
	for i := range ms.index.Departments {
		dept := &ms.index.Departments[i]
		for j := range dept.Roles {
			role := &dept.Roles[j]
			ms.byID[role.ID] = roleEntry{
				role:     role,
				deptID:   dept.ID,
				deptName: dept.Name,
			}
		}
	}
}

// ListDepts 返回所有有角色的部门摘要列表
func (ms *MarketStore) ListDepts() []model.MarketDeptSummary {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	result := make([]model.MarketDeptSummary, 0, len(ms.index.Departments))
	for _, dept := range ms.index.Departments {
		result = append(result, model.MarketDeptSummary{
			ID:    dept.ID,
			Name:  dept.Name,
			Count: len(dept.Roles),
		})
	}
	return result
}

// ListRoles 返回过滤后的角色列表（支持部门筛选和关键词搜索）
func (ms *MarketStore) ListRoles(filter model.MarketRoleFilter) []model.MarketRoleListItem {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	q := strings.ToLower(filter.Query)

	var result []model.MarketRoleListItem
	for _, dept := range ms.index.Departments {
		if filter.DeptID != "" && dept.ID != filter.DeptID {
			continue
		}
		for _, role := range dept.Roles {
			if q != "" {
				if !strings.Contains(strings.ToLower(role.Name), q) &&
					!strings.Contains(strings.ToLower(role.Description), q) {
					continue
				}
			}
			result = append(result, model.MarketRoleListItem{
				ID:          role.ID,
				Name:        role.Name,
				Description: role.Description,
				DeptID:      dept.ID,
				DeptName:    dept.Name,
			})
		}
	}
	return result
}

// GetRole 返回角色详情（含三个文件内容）
func (ms *MarketStore) GetRole(id string) (*model.MarketRoleDetail, *transport.AppError) {
	ms.mu.RLock()
	entry, ok := ms.byID[id]
	ms.mu.RUnlock()

	if !ok {
		return nil, transport.NotFound("role not found")
	}

	identity, err := os.ReadFile(entry.role.Files.Identity)
	if err != nil {
		return nil, transport.NotFound("role files not found")
	}
	soul, err := os.ReadFile(entry.role.Files.Soul)
	if err != nil {
		soul = []byte("")
	}
	agents, err := os.ReadFile(entry.role.Files.Agents)
	if err != nil {
		agents = []byte("")
	}

	return &model.MarketRoleDetail{
		MarketRoleListItem: model.MarketRoleListItem{
			ID:          entry.role.ID,
			Name:        entry.role.Name,
			Description: entry.role.Description,
			DeptID:      entry.deptID,
			DeptName:    entry.deptName,
		},
		IdentityContent: string(identity),
		SoulContent:     string(soul),
		AgentsContent:   string(agents),
	}, nil
}

// WriteRoleZip 将角色包三个文件打包为 zip 并写入 http.ResponseWriter（流式，无临时文件）
func (ms *MarketStore) WriteRoleZip(w http.ResponseWriter, id string) *transport.AppError {
	ms.mu.RLock()
	entry, ok := ms.byID[id]
	ms.mu.RUnlock()

	if !ok {
		return transport.NotFound("role not found")
	}

	files := map[string]string{
		"IDENTITY.md": entry.role.Files.Identity,
		"SOUL.md":     entry.role.Files.Soul,
		"AGENTS.md":   entry.role.Files.Agents,
	}

	// 响应头必须在 zip.NewWriter 之前设置
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, id))
	w.WriteHeader(http.StatusOK)

	zw := zip.NewWriter(w)
	defer zw.Close()

	for name, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			continue // 文件不存在则跳过
		}
		f, err := zw.Create(filepath.Join(id, name))
		if err != nil {
			continue
		}
		_, _ = f.Write(data)
	}
	return nil
}

// resolveDeptID 从目录名解析部门 ID，优先匹配双段前缀
func resolveDeptID(dirName string) string {
	parts := strings.SplitN(dirName, "-", 3)
	if len(parts) >= 2 {
		twoSeg := parts[0] + "-" + parts[1]
		if id, ok := deptPrefixMap[twoSeg]; ok {
			return id
		}
	}
	if id, ok := deptPrefixMap[parts[0]]; ok {
		return id
	}
	return "other"
}

// parseIdentityFile 解析 IDENTITY.md，返回角色名称和描述
// 格式：第一行 "# 名称"，第二行一句话描述
func parseIdentityFile(path string) (name, desc string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if name == "" {
			name = strings.TrimPrefix(line, "# ")
			name = strings.TrimSpace(name)
			continue
		}
		desc = line
		break
	}
	return name, desc
}
