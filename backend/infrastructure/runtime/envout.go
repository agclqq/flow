package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveEnvOutInstanceDir 解析实例级 FLOW_ENV_OUT 目录。
// configured 为配置文件中的根目录；有效时使用 {configured}/{templateID}/{instanceID}，
// 否则回退 /tmp/flow/env/{templateID}/{instanceID}。
func ResolveEnvOutInstanceDir(configured string, templateID, instanceID int64) string {
	fallback := filepath.Join("/tmp", "flow", "env", fmt.Sprintf("%d", templateID), fmt.Sprintf("%d", instanceID))
	root := strings.TrimSpace(configured)
	if root == "" {
		return fallback
	}
	candidate := filepath.Join(root, fmt.Sprintf("%d", templateID), fmt.Sprintf("%d", instanceID))
	if dirWritable(candidate) {
		return candidate
	}
	return fallback
}

func dirWritable(dir string) bool {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false
	}
	test := filepath.Join(dir, ".flow_write_test")
	if err := os.WriteFile(test, []byte("ok"), 0o644); err != nil {
		return false
	}
	_ = os.Remove(test)
	return true
}

// EnvOutFilePath 返回节点 env 导出文件的完整路径。
func EnvOutFilePath(instanceDir string, nodeID int64) string {
	return filepath.Join(instanceDir, fmt.Sprintf("%d.env.out", nodeID))
}

// EnsureEnvOutFile 创建空导出文件（实例启动时预建，供脚本 append）。
func EnsureEnvOutFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}
