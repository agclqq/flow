package executor

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

const (
	envBeginMarker = "__FLOW_ENV_BEGIN__"
	envEndMarker   = "__FLOW_ENV_END__"
)

// SSHExecutor 通过 SSH 在远程主机上执行 shell 脚本。
type SSHExecutor struct {
	cfg SSHConfig
}

// NewSSHExecutor 构造 SSH 执行器并校验配置。
func NewSSHExecutor(cfg SSHConfig) (*SSHExecutor, error) {
	if cfg.Host == "" || cfg.Username == "" {
		return nil, fmt.Errorf("ssh resource requires host and username")
	}
	if cfg.Port <= 0 {
		cfg.Port = 22
	}
	if cfg.AuthType == "" {
		cfg.AuthType = "password"
	}
	return &SSHExecutor{cfg: cfg}, nil
}

// Type 返回执行器类型。
func (e *SSHExecutor) Type() string {
	return RuntimeKindSSH
}

func (e *SSHExecutor) clientConfig() (*gossh.ClientConfig, error) {
	var auth []gossh.AuthMethod
	switch e.cfg.AuthType {
	case "key":
		if e.cfg.PrivateKey == "" {
			return nil, fmt.Errorf("ssh key auth requires private_key")
		}
		signer, err := gossh.ParsePrivateKey([]byte(e.cfg.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		auth = append(auth, gossh.PublicKeys(signer))
	default:
		if e.cfg.Password == "" {
			return nil, fmt.Errorf("ssh password auth requires password")
		}
		auth = append(auth, gossh.Password(e.cfg.Password))
	}
	return &gossh.ClientConfig{
		User:            e.cfg.Username,
		Auth:            auth,
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}, nil
}

func (e *SSHExecutor) dial(ctx context.Context) (*gossh.Client, error) {
	cfg, err := e.clientConfig()
	if err != nil {
		return nil, err
	}
	addr := net.JoinHostPort(e.cfg.Host, fmt.Sprintf("%d", e.cfg.Port))
	dialer := &net.Dialer{Timeout: cfg.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial ssh %s: %w", addr, err)
	}
	sshConn, chans, reqs, err := gossh.NewClientConn(conn, addr, cfg)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ssh handshake %s: %w", addr, err)
	}
	return gossh.NewClient(sshConn, chans, reqs), nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Execute 在远端以 bash 执行脚本：注入环境变量，收集输出与退出码，
// 并通过 FLOW_ENV_OUT 文件捕获脚本导出的环境变量（读取后清理）。
func (e *SSHExecutor) Execute(ctx context.Context, spec ExecuteSpec) (ExecuteResult, error) {
	client, err := e.dial(ctx)
	if err != nil {
		return ExecuteResult{}, err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return ExecuteResult{}, fmt.Errorf("new ssh session: %w", err)
	}
	defer session.Close()

	envOut := spec.Env[EnvOutFileKey]
	if envOut == "" {
		envOut = fmt.Sprintf("/tmp/flow_env_out_%d", time.Now().UnixNano())
	}

	var b strings.Builder
	b.WriteString("set -u\n")
	if e.cfg.WorkDir != "" || spec.WorkingDir != "" {
		dir := spec.WorkingDir
		if dir == "" {
			dir = e.cfg.WorkDir
		}
		fmt.Fprintf(&b, "cd %s || exit 90\n", shellQuote(dir))
	}
	for k, v := range spec.Env {
		if k == EnvOutFileKey {
			continue
		}
		fmt.Fprintf(&b, "export %s=%s\n", k, shellQuote(v))
	}
	fmt.Fprintf(&b, "export %s=%s\n", EnvOutFileKey, shellQuote(envOut))
	b.WriteString(spec.Command)
	b.WriteString("\n")
	fmt.Fprintf(&b, "__flow_rc=$?\n")
	fmt.Fprintf(&b, "echo %s\n", envBeginMarker)
	fmt.Fprintf(&b, "[ -f \"$%s\" ] && cat \"$%s\"; rm -f \"$%s\"\n", EnvOutFileKey, EnvOutFileKey, EnvOutFileKey)
	fmt.Fprintf(&b, "echo %s\n", envEndMarker)
	fmt.Fprintf(&b, "exit $__flow_rc\n")

	var output strings.Builder
	session.Stdout = &output
	session.Stderr = &output

	done := make(chan error, 1)
	go func() {
		done <- session.Run(b.String())
	}()
	select {
	case <-ctx.Done():
		_ = session.Signal(gossh.SIGKILL)
		return ExecuteResult{}, ctx.Err()
	case runErr := <-done:
		err = runErr
	}

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*gossh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
		} else {
			return ExecuteResult{}, err
		}
	}

	exported, cleanOutput := extractEnvBlock(output.String())
	return ExecuteResult{
		ExitCode:    exitCode,
		Output:      cleanOutput,
		ExportedEnv: exported,
	}, nil
}

// extractEnvBlock 从合并输出中截取导出环境变量块并解析，返回解析结果与剥离后的输出。
func extractEnvBlock(output string) (map[string]string, string) {
	begin := strings.Index(output, envBeginMarker)
	if begin < 0 {
		return map[string]string{}, strings.TrimSuffix(output, "\n")
	}
	rest := output[begin+len(envBeginMarker):]
	end := strings.Index(rest, envEndMarker)
	if end < 0 {
		return map[string]string{}, strings.TrimSuffix(output, "\n")
	}
	block := rest[:end]
	tail := rest[end+len(envEndMarker):]
	head := strings.TrimSuffix(output[:begin], "\n")
	clean := strings.TrimLeft(strings.TrimSuffix(head+tail, "\n"), "\n")
	return ParseEnvFile([]byte(block)), clean
}
