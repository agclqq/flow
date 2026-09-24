package executor

import (
	"bufio"
	"bytes"
	"os"
	"strings"
)

// EnvOutFileKey 是注入脚本环境、指向导出文件路径的变量名。
const EnvOutFileKey = "FLOW_ENV_OUT"

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
