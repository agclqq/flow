package executor

import "regexp"

var scriptPlaceholderPattern = regexp.MustCompile(`\{\{([A-Za-z_][A-Za-z0-9_]*)\}\}`)

// ExpandScriptPlaceholders 将脚本中的 {{KEY}} 替换为 env 中对应值（与编辑器补全语法一致）。
// 未找到的键保持原样。
func ExpandScriptPlaceholders(script string, env map[string]string) string {
	if script == "" || len(env) == 0 {
		return script
	}
	return scriptPlaceholderPattern.ReplaceAllStringFunc(script, func(match string) string {
		sub := scriptPlaceholderPattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		if v, ok := env[sub[1]]; ok {
			return v
		}
		return match
	})
}
