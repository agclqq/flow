package executor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// LocalExecutor 在本地环境中执行 shell 脚本。
type LocalExecutor struct{}

// NewLocalExecutor 构造本地执行器。
func NewLocalExecutor() *LocalExecutor {
	return &LocalExecutor{}
}

// Type 返回执行器类型。
func (e *LocalExecutor) Type() string {
	return "local"
}

// Execute 在本地执行 shell 脚本，注入环境变量，返回退出码和输出。
// 若 spec.Env 中包含 FLOW_ENV_OUT，脚本可通过向该文件追加 KEY=VALUE 导出新变量。
func (e *LocalExecutor) Execute(ctx context.Context, spec ExecuteSpec) (ExecuteResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	envOutPath := spec.Env[EnvOutFileKey]
	if envOutPath != "" {
		if err := os.MkdirAll(filepath.Dir(envOutPath), 0o755); err != nil {
			return ExecuteResult{}, fmt.Errorf("create env out dir: %w", err)
		}
		if _, err := os.Stat(envOutPath); os.IsNotExist(err) {
			if err := os.WriteFile(envOutPath, nil, 0o644); err != nil {
				return ExecuteResult{}, fmt.Errorf("init env out file: %w", err)
			}
		}
	}

	cmd := exec.CommandContext(ctx, "bash", "-c", spec.Command)

	env := make([]string, 0, len(spec.Env))
	for k, v := range spec.Env {
		env = append(env, k+"="+v)
	}
	cmd.Env = env

	if spec.WorkingDir != "" {
		cmd.Dir = spec.WorkingDir
	}

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		} else {
			return ExecuteResult{}, err
		}
	}

	exportedEnv, readErr := ReadEnvFile(envOutPath)
	if readErr != nil {
		return ExecuteResult{}, readErr
	}

	return ExecuteResult{
		ExitCode:    exitCode,
		Output:      output.String(),
		ExportedEnv: exportedEnv,
	}, nil
}
