package executor

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// K8sExecutor 通过 kubeconfig 直连 Kubernetes API，
// 为每个脚本创建一次性 Pod 运行，完成后读取日志捕获输出与导出环境变量。
type K8sExecutor struct {
	cfg       K8sConfig
	server    string
	namespace string
	image     string
	http      *http.Client
	token     string
	username  string
	password  string
}

// NewK8sExecutor 解析 kubeconfig 并构造 k8s 执行器。
func NewK8sExecutor(cfg K8sConfig) (*K8sExecutor, error) {
	if strings.TrimSpace(cfg.Kubeconfig) == "" {
		return nil, fmt.Errorf("k8s resource requires kubeconfig")
	}
	var kc kubeconfigFile
	if err := yaml.Unmarshal([]byte(cfg.Kubeconfig), &kc); err != nil {
		return nil, fmt.Errorf("parse kubeconfig: %w", err)
	}
	ctxName := kc.CurrentContext
	if ctxName == "" && len(kc.Contexts) > 0 {
		ctxName = kc.Contexts[0].Name
	}
	var clusterRef, userRef string
	for _, c := range kc.Contexts {
		if c.Name == ctxName {
			clusterRef, userRef = c.Context.Cluster, c.Context.User
			break
		}
	}
	var cluster *kubeCluster
	for i := range kc.Clusters {
		if kc.Clusters[i].Name == clusterRef {
			cluster = &kc.Clusters[i].Cluster
			break
		}
	}
	if cluster == nil || cluster.Server == "" {
		return nil, fmt.Errorf("kubeconfig has no usable cluster server")
	}

	e := &K8sExecutor{
		cfg:       cfg,
		server:    strings.TrimSuffix(cluster.Server, "/"),
		namespace: cfg.Namespace,
		image:     cfg.Image,
	}
	if e.namespace == "" {
		e.namespace = "default"
	}
	if e.image == "" {
		e.image = "busybox:1.36"
	}

	tlsCfg := &tls.Config{InsecureSkipVerify: cluster.InsecureSkipTLSVerify}
	if caData := decodeB64(cluster.CertificateAuthorityData); len(caData) > 0 {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caData) {
			return nil, fmt.Errorf("kubeconfig certificate-authority-data is invalid")
		}
		tlsCfg.RootCAs = pool
	}

	var user *kubeUser
	for i := range kc.Users {
		if kc.Users[i].Name == userRef {
			user = &kc.Users[i].User
			break
		}
	}
	if user != nil {
		e.token = user.Token
		e.username, e.password = user.Username, user.Password
		certData, keyData := decodeB64(user.ClientCertificateData), decodeB64(user.ClientKeyData)
		if len(certData) > 0 && len(keyData) > 0 {
			pair, err := tls.X509KeyPair(certData, keyData)
			if err != nil {
				return nil, fmt.Errorf("kubeconfig client certificate is invalid: %w", err)
			}
			tlsCfg.Certificates = []tls.Certificate{pair}
		}
	}
	e.http = &http.Client{Transport: &http.Transport{TLSClientConfig: tlsCfg}, Timeout: 30 * time.Second}
	return e, nil
}

// Type 返回执行器类型。
func (e *K8sExecutor) Type() string { return RuntimeKindK8s }

type kubeCluster struct {
	Server                   string `yaml:"server"`
	CertificateAuthorityData string `yaml:"certificate-authority-data"`
	InsecureSkipTLSVerify    bool   `yaml:"insecure-skip-tls-verify"`
}

type kubeUser struct {
	Token                 string `yaml:"token"`
	ClientCertificateData string `yaml:"client-certificate-data"`
	ClientKeyData         string `yaml:"client-key-data"`
	Username              string `yaml:"username"`
	Password              string `yaml:"password"`
}

type kubeconfigFile struct {
	CurrentContext string `yaml:"current-context"`
	Clusters       []struct {
		Name    string      `yaml:"name"`
		Cluster kubeCluster `yaml:"cluster"`
	} `yaml:"clusters"`
	Users []struct {
		Name string   `yaml:"name"`
		User kubeUser `yaml:"user"`
	} `yaml:"users"`
	Contexts []struct {
		Name    string `yaml:"name"`
		Context struct {
			Cluster string `yaml:"cluster"`
			User    string `yaml:"user"`
		} `yaml:"context"`
	} `yaml:"contexts"`
}

type kubePod struct {
	Status struct {
		Phase string `json:"phase"`
		ContainerStatuses []struct {
			Name         string `json:"name"`
			State        struct {
				Terminated *struct {
					ExitCode int `json:"exitCode"`
				} `json:"terminated"`
			} `json:"state"`
		} `json:"containerStatuses"`
	} `json:"status"`
}

func decodeB64(s string) []byte {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if data, err := base64.StdEncoding.DecodeString(s); err == nil {
		return data
	}
	return []byte(s)
}

func (e *K8sExecutor) do(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, e.server+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if e.token != "" {
		req.Header.Set("Authorization", "Bearer "+e.token)
	} else if e.username != "" {
		req.SetBasicAuth(e.username, e.password)
	}
	resp, err := e.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("k8s api %s %s: status %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}

// Execute 创建一次性 Pod 运行脚本，轮询至结束并读取日志。
func (e *K8sExecutor) Execute(ctx context.Context, spec ExecuteSpec) (ExecuteResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	podName := fmt.Sprintf("flow-%d-%d-%d", time.Now().Unix(), spec.InstanceID%100000, time.Now().Nanosecond()%100000)
	image := spec.Image
	if image == "" {
		image = e.image
	}

	envList := make([]map[string]string, 0, len(spec.Env))
	for k, v := range spec.Env {
		envList = append(envList, map[string]string{"name": k, "value": v})
	}

	wrapper := strings.Join([]string{
		spec.Command,
		"__flow_rc=$?",
		"echo " + envBeginMarker,
		"cat \"$FLOW_ENV_OUT\" 2>/dev/null; rm -f \"$FLOW_ENV_OUT\"",
		"echo " + envEndMarker,
		"exit $__flow_rc",
	}, "\n")

	pod := map[string]any{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]any{
			"name":   podName,
			"labels": map[string]string{"app": "flow-runner"},
		},
		"spec": map[string]any{
			"restartPolicy": "Never",
			"containers": []map[string]any{{
				"name":            "main",
				"image":           image,
				"command":         []string{"sh", "-c", wrapper},
				"env":             envList,
				"imagePullPolicy": "IfNotPresent",
			}},
		},
	}

	if _, err := e.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/namespaces/%s/pods", e.namespace), pod); err != nil {
		return ExecuteResult{}, fmt.Errorf("create pod: %w", err)
	}
	defer func() {
		delCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, _ = e.do(delCtx, http.MethodDelete, fmt.Sprintf("/api/v1/namespaces/%s/pods/%s", e.namespace, podName), nil)
	}()

	exitCode := -1
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ExecuteResult{}, ctx.Err()
		case <-ticker.C:
		}

		data, err := e.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/namespaces/%s/pods/%s", e.namespace, podName), nil)
		if err != nil {
			continue // 瞬时 API 错误，等待下次轮询
		}
		var p kubePod
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}
		switch p.Status.Phase {
		case "Succeeded", "Failed":
			for _, cs := range p.Status.ContainerStatuses {
				if cs.Name == "main" && cs.State.Terminated != nil {
					exitCode = cs.State.Terminated.ExitCode
				}
			}
			if exitCode < 0 {
				exitCode = 0
				if p.Status.Phase == "Failed" {
					exitCode = 1
				}
			}
		default:
			continue
		}

		logData, err := e.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/log", e.namespace, podName), nil)
		if err != nil {
			logData = nil
		}
		exported, output := extractEnvBlock(string(logData))
		return ExecuteResult{ExitCode: exitCode, Output: output, ExportedEnv: exported}, nil
	}
}
