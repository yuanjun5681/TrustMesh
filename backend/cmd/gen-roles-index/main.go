// gen-roles-index 扫描 backend/data/roles/ 目录，生成 backend/data/roles_index.json。
//
// 用法（从 backend/ 目录执行）：
//
//	go run ./cmd/gen-roles-index
//	go run ./cmd/gen-roles-index -roles data/roles -output data/roles_index.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"trustmesh/backend/internal/store"
)

func main() {
	rolesDir := flag.String("roles", "data/roles", "角色包目录路径")
	outputPath := flag.String("output", "data/roles_index.json", "输出 JSON 文件路径")
	filePrefix := flag.String("prefix", "data/roles", "JSON 中 files 字段的路径前缀（相对于后端工作目录）")
	flag.Parse()

	fmt.Printf("扫描角色目录: %s\n", *rolesDir)

	idx, err := store.BuildRolesIndex(*rolesDir, *filePrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON 序列化失败: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outputPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "写入文件失败: %v\n", err)
		os.Exit(1)
	}

	totalRoles := 0
	for _, dept := range idx.Departments {
		totalRoles += len(dept.Roles)
		fmt.Printf("  %-12s (%s): %d 个角色\n", dept.Name, dept.ID, len(dept.Roles))
	}
	fmt.Printf("\n共 %d 个部门，%d 个角色\n", len(idx.Departments), totalRoles)
	fmt.Printf("已写入: %s (%d bytes)\n", *outputPath, len(data))
}
