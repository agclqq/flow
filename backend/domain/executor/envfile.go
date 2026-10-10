package executor

import (
	"bufio"
	"bytes"
	"os"
	"strings"
)

// EnvOutFileKey 是注入脚本环境、指向导出文件路径的变量名。
const EnvOutFileKey = "FLOW_ENV_OUT"

// ShellInitEnvOutSnippet 在脚本开头确保 FLOW_ENV_OUT 路径存在（不截断已有内容，便于多行 append）。
func ShellInitEnvOutSnippet() string {
	return `if [ -n "$FLOW_ENV_OUT" ]; then mkdir -p "$(dirname "$FLOW_ENV_OUT")" && touch "$FLOW_ENV_OUT"; fi`
}

// ParseEnvFile 解析 KEY=VALUE 格式的环境变量导出文件。
// 支持空行、# 注释，以及 export KEY=VALUE 前缀。
func ParseEnvFile(data []byte) map[string]string {
	env := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok || key == "" {
			continue
		}
		env[strings.TrimSpace(key)] = strings.TrimSpace(val)
	}
	return env
}

// ReadEnvFile 读取导出文件并解析为 map。
func ReadEnvFile(path string) (map[string]string, error) {
	if path == "" {
		return map[string]string{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	return ParseEnvFile(data), nil
}
