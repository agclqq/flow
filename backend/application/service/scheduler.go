package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"flow/domain/entity"
	"flow/domain/executor"
	"flow/domain/repository"
	"flow/domain/valueobject"
)

// ErrWorkflowPaused 表示流水线因等待人工审批而暂停。
var ErrWorkflowPaused = errors.New("workflow paused waiting for approval")

// 节点执行状态（调度内存态）。
const (
	nodePending         = 0
	nodeDone            = 1
	nodeSkipped         = 2
	nodeFailed          = 3
	nodeWaitingApproval = 4
)

// Scheduler 是 DAG 调度器，按拓扑顺序执行流水线中的节点。
// 节点的执行位置由 runtime_kind + resource_id 决定：
// local（本地调试）/ ssh（远程主机）/ k8s（一次性 Pod）。
type Scheduler struct {
	executor    executor.Executor // 兜底执行器（本地）
	resources   *ExecutorResourceService
	instRepo    repository.FlowInstanceRepository
	versionRepo repository.FlowTemplateVersionRepository
	logger      *slog.Logger
	// runCancels 正在执行的节点实例 → 取消函数，用于手动跳过时终止脚本。
	runCancels   map[int64]context.CancelFunc
	runCancelsMu sync.Mutex
}

// NewScheduler 构造调度器。
func NewScheduler(
	exec executor.Executor,
	resources *ExecutorResourceService,
	instRepo repository.FlowInstanceRepository,
	versionRepo repository.FlowTemplateVersionRepository,
	logger *slog.Logger,
) *Scheduler {
	return &Scheduler{
		executor:    exec,
		resources:   resources,
		instRepo:    instRepo,
		versionRepo: versionRepo,
		logger:      logger,
		runCancels:  map[int64]context.CancelFunc{},
	}
}

// resolveExecutor 根据节点运行时配置解析执行器。
func (s *Scheduler) resolveExecutor(ctx context.Context, n *entity.Node) (executor.Executor, error) {
	if n.RuntimeKind == "" || n.RuntimeKind == executor.RuntimeKindLocal {
		if s.executor != nil {
			return s.executor, nil
		}
	}
	return s.resources.resolveRuntime(ctx, n.RuntimeKind, n.ResourceID)
}

// Run 执行流水线实例（首次运行，创建节点实例）。
func (s *Scheduler) Run(ctx context.Context, inst *entity.FlowInstance) error {
	ver, err := s.versionRepo.GetByID(ctx, inst.TemplateVersionID)
	if err != nil {
		return fmt.Errorf("load version: %w", err)
	}

	nodeInstMap := make(map[int64]*entity.NodeInstance)
	for _, n := range ver.Nodes {
		if n.IsEdge() {
			continue
		}
		ni, err := entity.NewNodeInstance(inst.ID, n.ID, n.Type)
		if err != nil {
			return err
		}
		s.refreshNodeEnv(inst, ni, n)
		if err := s.instRepo.CreateNodeInstance(ctx, ni); err != nil {
			return err
		}
		nodeInstMap[n.ID] = ni
	}

	state := s.initialState(ver.Nodes)
	return s.execute(ctx, inst, ver.Nodes, nodeInstMap, state)
}

// Continue 从已有节点实例恢复并继续执行。
func (s *Scheduler) Continue(ctx context.Context, inst *entity.FlowInstance) error {
	ver, err := s.versionRepo.GetByID(ctx, inst.TemplateVersionID)
	if err != nil {
		return fmt.Errorf("load version: %w", err)
	}

	nis, err := s.instRepo.ListNodeInstances(ctx, inst.ID)
	if err != nil {
		return fmt.Errorf("load node instances: %w", err)
	}
	nodeInstMap := make(map[int64]*entity.NodeInstance, len(nis))
	for _, ni := range nis {
		nodeInstMap[ni.NodeID] = ni
	}

	state := s.rebuildState(ver.Nodes, nodeInstMap)
	return s.execute(ctx, inst, ver.Nodes, nodeInstMap, state)
}

// ApproveManualNode 审批通过 manual 节点：可选执行脚本后标记成功。
func (s *Scheduler) ApproveManualNode(ctx context.Context, inst *entity.FlowInstance, n *entity.Node, ni *entity.NodeInstance) error {
	if ni.Status != valueobject.NodeInstanceStatusWaitingApproval {
		return errors.New("node is not waiting for approval")
	}
	if n.PreScript != "" || n.Script != "" || n.PostScript != "" {
		if !s.executeScripts(ctx, n, inst, ni) {
			s.completeNodeInst(ctx, ni, valueobject.NodeInstanceStatusFailed, ni.OutputEnv)
			s.markInstanceFailed(ctx, inst)
			return inst.Transition(valueobject.FlowInstanceStatusFailed)
		}
	}
	s.completeNodeInst(ctx, ni, valueobject.NodeInstanceStatusSucceeded, ni.OutputEnv)
	return nil
}

func (s *Scheduler) initialState(nodes []*entity.Node) map[int64]int {
	state := make(map[int64]int, len(nodes))
	for _, n := range nodes {
		if n.IsEdge() {
			continue
		}
		state[n.ID] = nodePending
	}
	return state
}

func (s *Scheduler) rebuildState(nodes []*entity.Node, nodeInstMap map[int64]*entity.NodeInstance) map[int64]int {
	state := s.initialState(nodes)
	for _, n := range nodes {
		if n.IsEdge() {
			continue
		}
		ni, ok := nodeInstMap[n.ID]
		if !ok {
			continue
		}
		switch ni.Status {
		case valueobject.NodeInstanceStatusSucceeded, valueobject.NodeInstanceStatusSkipped:
			// 手动跳过视为成功：下游继续调度。
			state[n.ID] = nodeDone
		case valueobject.NodeInstanceStatusFailedSkipped:
			state[n.ID] = nodeSkipped
		case valueobject.NodeInstanceStatusFailed:
			state[n.ID] = nodeFailed
		case valueobject.NodeInstanceStatusWaitingApproval:
			state[n.ID] = nodeWaitingApproval
		case valueobject.NodeInstanceStatusRunning, valueobject.NodeInstanceStatusRetrying:
			state[n.ID] = nodePending
		}
	}
	return state
}

// registerCancel 登记执行中节点的取消函数。
func (s *Scheduler) registerCancel(nodeInstanceID int64, cancel context.CancelFunc) {
	s.runCancelsMu.Lock()
	defer s.runCancelsMu.Unlock()
	s.runCancels[nodeInstanceID] = cancel
}

// popCancel 取出并移除节点的取消函数；不存在返回 nil。
func (s *Scheduler) popCancel(nodeInstanceID int64) context.CancelFunc {
	s.runCancelsMu.Lock()
	defer s.runCancelsMu.Unlock()
	cancel, ok := s.runCancels[nodeInstanceID]
	if ok {
		delete(s.runCancels, nodeInstanceID)
	}
	return cancel
}

// SkipNode 手动跳过运行中的节点：终止其脚本执行，状态标记为跳过。
// 跳过视为成功，调度循环会继续调度后续节点。
func (s *Scheduler) SkipNode(ctx context.Context, ni *entity.NodeInstance) error {
	if ni.Status != valueobject.NodeInstanceStatusRunning {
		return errors.New("only running nodes can be skipped")
	}
	// 先终止执行中的脚本；执行 goroutine 结束时发现已是终态则不再覆盖结果。
	if cancel := s.popCancel(ni.ID); cancel != nil {
		cancel()
	}
	if err := ni.Transition(valueobject.NodeInstanceStatusSkipped); err != nil {
		return err
	}
	return s.instRepo.UpdateNodeInstance(ctx, ni)
}

type graphIndex struct {
	incomingEdges map[int64][]*entity.Node
	edgeByID      map[int64]*entity.Node
}

func buildGraphIndex(nodes []*entity.Node) graphIndex {
	idx := graphIndex{
		incomingEdges: make(map[int64][]*entity.Node),
		edgeByID:      make(map[int64]*entity.Node),
	}
	for _, n := range nodes {
		if n.IsEdge() {
			idx.incomingEdges[n.ToNodeID] = append(idx.incomingEdges[n.ToNodeID], n)
			idx.edgeByID[n.ID] = n
		}
	}
	return idx
}

func predecessorIDs(incomingEdges map[int64][]*entity.Node, nodeID int64) []int64 {
	edges := incomingEdges[nodeID]
	ids := make([]int64, 0, len(edges))
	for _, e := range edges {
		ids = append(ids, e.FromNodeID)
	}
	return ids
}

func (s *Scheduler) execute(ctx context.Context, inst *entity.FlowInstance, nodes []*entity.Node, nodeInstMap map[int64]*entity.NodeInstance, state map[int64]int) error {
	idx := buildGraphIndex(nodes)

	for {
		if inst.Status == valueobject.FlowInstanceStatusFailed {
			break
		}
		if inst.Status == valueobject.FlowInstanceStatusSucceeded {
			break
		}

		progress := false
		var readyTasks []*entity.Node
		var readyManual *entity.Node

		for _, n := range nodes {
			if n.IsEdge() {
				continue
			}
			if state[n.ID] != nodePending {
				continue
			}
			if inst.Status == valueobject.FlowInstanceStatusFailed {
				continue
			}

			switch n.Type {
			case valueobject.NodeTypeStart:
				state[n.ID] = nodeDone
				s.transitionNodeInst(ctx, nodeInstMap[n.ID], valueobject.NodeInstanceStatusRunning)
				s.completeNodeInst(ctx, nodeInstMap[n.ID], valueobject.NodeInstanceStatusSucceeded, nil)
				progress = true

			case valueobject.NodeTypeTask, valueobject.NodeTypeJoin:
				preds := predecessorIDs(idx.incomingEdges, n.ID)
				if s.isNodeSkippable(n, preds, state, idx) {
					state[n.ID] = nodeSkipped
					s.transitionNodeInst(ctx, nodeInstMap[n.ID], valueobject.NodeInstanceStatusRunning)
					s.completeNodeInst(ctx, nodeInstMap[n.ID], valueobject.NodeInstanceStatusFailedSkipped, nil)
					progress = true
					continue
				}
				if s.isNodeStartReady(n, preds, state, idx) {
					// 收集本轮所有就绪任务，稍后并发执行。
					readyTasks = append(readyTasks, n)
					progress = true
				}

			case valueobject.NodeTypeManual:
				preds := predecessorIDs(idx.incomingEdges, n.ID)
				if s.isNodeSkippable(n, preds, state, idx) {
					state[n.ID] = nodeSkipped
					s.transitionNodeInst(ctx, nodeInstMap[n.ID], valueobject.NodeInstanceStatusRunning)
					s.completeNodeInst(ctx, nodeInstMap[n.ID], valueobject.NodeInstanceStatusFailedSkipped, nil)
					progress = true
					continue
				}
				if s.isNodeStartReady(n, preds, state, idx) && readyManual == nil {
					// 人工节点延后到本轮并发任务全部完成后处理。
					readyManual = n
				}

			case valueobject.NodeTypeEnd:
				if s.isEndNodeReady(n.ID, idx.incomingEdges, state) {
					state[n.ID] = nodeDone
					s.transitionNodeInst(ctx, nodeInstMap[n.ID], valueobject.NodeInstanceStatusRunning)
					s.completeNodeInst(ctx, nodeInstMap[n.ID], valueobject.NodeInstanceStatusSucceeded, nil)
					s.markInstanceSucceeded(ctx, inst)
					progress = true
				}
			}
		}

		switch {
		case len(readyTasks) > 0:
			// 同一轮就绪的任务节点并发执行（含 start_script 与跳过判定）。
			s.executeTaskNodesConcurrently(ctx, readyTasks, inst, nodeInstMap, state)
			progress = true

		case readyManual != nil:
			if !s.evaluateStartScript(ctx, readyManual, inst, nodeInstMap[readyManual.ID]) {
				state[readyManual.ID] = nodeSkipped
				s.transitionNodeInst(ctx, nodeInstMap[readyManual.ID], valueobject.NodeInstanceStatusRunning)
				s.completeNodeInst(ctx, nodeInstMap[readyManual.ID], valueobject.NodeInstanceStatusFailedSkipped, nil)
			} else {
				s.pauseManualNode(ctx, nodeInstMap[readyManual.ID], state, readyManual.ID)
				if err := s.instRepo.Update(ctx, inst); err != nil {
					s.logger.Error("update instance env failed", "instance_id", inst.ID, "error", err)
				}
				return ErrWorkflowPaused
			}
			progress = true
		}

		if !progress {
			break
		}
	}

	if err := s.instRepo.Update(ctx, inst); err != nil {
		s.logger.Error("update instance env failed", "instance_id", inst.ID, "error", err)
	}

	if inst.Status == valueobject.FlowInstanceStatusFailed {
		return inst.Transition(valueobject.FlowInstanceStatusFailed)
	}

	if inst.Status == valueobject.FlowInstanceStatusSucceeded {
		return nil
	}

	for _, st := range state {
		if st == nodeWaitingApproval {
			return ErrWorkflowPaused
		}
	}

	for _, n := range nodes {
		if n.IsEdge() {
			continue
		}
		if state[n.ID] == nodePending {
			if inst.Status == valueobject.FlowInstanceStatusFailed {
				return inst.Transition(valueobject.FlowInstanceStatusFailed)
			}
		}
	}

	return inst.Transition(valueobject.FlowInstanceStatusSucceeded)
}

func (s *Scheduler) effectiveStartMode(n *entity.Node) valueobject.StartMode {
	if n.StartMode.Valid() {
		return n.StartMode
	}
	if n.IsJoin() {
		return valueobject.StartModeAll
	}
	return valueobject.StartModeAny
}

func (s *Scheduler) effectiveRequiredPredecessors(n *entity.Node, idx graphIndex) []int64 {
	if len(n.RequiredPredecessorIDs) > 0 {
		return n.RequiredPredecessorIDs
	}
	if len(n.RequiredEdgeIDs) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(n.RequiredEdgeIDs))
	for _, edgeID := range n.RequiredEdgeIDs {
		edge, ok := idx.edgeByID[edgeID]
		if !ok || edge.FromNodeID == 0 {
			continue
		}
		ids = append(ids, edge.FromNodeID)
	}
	return ids
}

func (s *Scheduler) allPredecessorsSettled(predIDs []int64, state map[int64]int) bool {
	for _, id := range predIDs {
		st := state[id]
		if st == nodePending || st == nodeWaitingApproval {
			return false
		}
	}
	return true
}

func (s *Scheduler) hasFailedPredecessor(predIDs []int64, state map[int64]int) bool {
	for _, id := range predIDs {
		if state[id] == nodeFailed {
			return true
		}
	}
	return false
}

func (s *Scheduler) meetsStartMode(n *entity.Node, predIDs []int64, state map[int64]int, idx graphIndex) bool {
	if len(predIDs) == 0 {
		return true
	}

	mode := s.effectiveStartMode(n)
	switch mode {
	case valueobject.StartModeAny:
		for _, id := range predIDs {
			if state[id] == nodeDone {
				return true
			}
		}
		return false
	case valueobject.StartModeAll:
		for _, id := range predIDs {
			if state[id] != nodeDone {
				return false
			}
		}
		return true
	case valueobject.StartModeCustom:
		required := s.effectiveRequiredPredecessors(n, idx)
		if len(required) > 0 {
			for _, id := range required {
				if state[id] != nodeDone {
					return false
				}
			}
			return true
		}
		for _, id := range predIDs {
			if state[id] == nodeDone {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func (s *Scheduler) isNodeStartReady(n *entity.Node, predIDs []int64, state map[int64]int, idx graphIndex) bool {
	if !s.allPredecessorsSettled(predIDs, state) {
		return false
	}
	if s.hasFailedPredecessor(predIDs, state) {
		return false
	}
	return s.meetsStartMode(n, predIDs, state, idx)
}

func (s *Scheduler) isNodeSkippable(n *entity.Node, predIDs []int64, state map[int64]int, idx graphIndex) bool {
	if len(predIDs) == 0 {
		return false
	}
	if !s.allPredecessorsSettled(predIDs, state) {
		return false
	}
	if s.hasFailedPredecessor(predIDs, state) {
		return false
	}
	return !s.meetsStartMode(n, predIDs, state, idx)
}

func (s *Scheduler) isEndNodeReady(nodeID int64, incomingEdges map[int64][]*entity.Node, state map[int64]int) bool {
	preds := predecessorIDs(incomingEdges, nodeID)
	if len(preds) == 0 {
		return true
	}
	return s.allPredecessorsSettled(preds, state)
}

func (s *Scheduler) evaluateStartScript(ctx context.Context, n *entity.Node, inst *entity.FlowInstance, ni *entity.NodeInstance) bool {
	if n.StartScript == "" {
		return true
	}
	result, err := s.runScript(ctx, n, inst, ni, n.StartScript)
	if err != nil {
		s.logger.Error("start script execution failed", "node_id", n.ID, "error", err)
		return false
	}
	return result.ExitCode == 0
}

func (s *Scheduler) pauseManualNode(ctx context.Context, ni *entity.NodeInstance, state map[int64]int, nodeID int64) {
	s.transitionNodeInst(ctx, ni, valueobject.NodeInstanceStatusRunning)
	s.transitionNodeInst(ctx, ni, valueobject.NodeInstanceStatusWaitingApproval)
	state[nodeID] = nodeWaitingApproval
}

func (s *Scheduler) markInstanceFailed(ctx context.Context, inst *entity.FlowInstance) {
	if inst.Status != valueobject.FlowInstanceStatusRunning {
		return
	}
	if err := inst.Transition(valueobject.FlowInstanceStatusFailed); err != nil {
		s.logger.Error("instance transition to failed", "instance_id", inst.ID, "error", err)
		return
	}
	if err := s.instRepo.Update(ctx, inst); err != nil {
		s.logger.Error("persist failed instance", "instance_id", inst.ID, "error", err)
	}
}

func (s *Scheduler) markInstanceSucceeded(ctx context.Context, inst *entity.FlowInstance) {
	if inst.Status != valueobject.FlowInstanceStatusRunning {
		return
	}
	if err := inst.Transition(valueobject.FlowInstanceStatusSucceeded); err != nil {
		s.logger.Error("instance transition to succeeded", "instance_id", inst.ID, "error", err)
		return
	}
	if err := s.instRepo.Update(ctx, inst); err != nil {
		s.logger.Error("persist succeeded instance", "instance_id", inst.ID, "error", err)
	}
}

func (s *Scheduler) refreshNodeEnv(inst *entity.FlowInstance, ni *entity.NodeInstance, node *entity.Node) {
	if ni.Env == nil {
		ni.Env = map[string]string{}
	}
	for k, v := range inst.Params {
		ni.Env[k] = v
	}
	for k, v := range inst.Env {
		ni.Env[k] = v
	}
	// 节点级系统参数：负责人列表以逗号连接。
	if node != nil && len(node.Owners) > 0 {
		ni.Env["node_owner"] = strings.Join(node.Owners, ",")
	}
	ni.Env[executor.EnvOutFileKey] = s.envOutPath(inst.ID, ni.NodeID)
}

func (s *Scheduler) envOutPath(instanceID, nodeID int64) string {
	return filepath.Join(os.TempDir(), "flow", fmt.Sprintf("%d", instanceID), fmt.Sprintf("%d.env.out", nodeID))
}

func (s *Scheduler) mergeExportedEnv(ctx context.Context, inst *entity.FlowInstance, ni *entity.NodeInstance, exported map[string]string) {
	if len(exported) == 0 {
		return
	}
	inst.UpdateEnv(exported)
	for k, v := range exported {
		ni.Env[k] = v
	}
	ni.SetOutputEnv(exported)
	if err := s.instRepo.Update(ctx, inst); err != nil {
		s.logger.Error("persist instance env failed", "instance_id", inst.ID, "error", err)
	}
	_ = s.instRepo.UpdateNodeInstance(ctx, ni)
}

func (s *Scheduler) runScript(ctx context.Context, n *entity.Node, inst *entity.FlowInstance, ni *entity.NodeInstance, command string) (executor.ExecuteResult, error) {
	s.refreshNodeEnv(inst, ni, n)
	result, err := s.executor.Execute(ctx, executor.ExecuteSpec{
		Command: command,
		Env:     ni.Env,
	})
	if err != nil {
		return result, err
	}
	s.mergeExportedEnv(ctx, inst, ni, result.ExportedEnv)
	return result, nil
}

func (s *Scheduler) executeScripts(ctx context.Context, n *entity.Node, inst *entity.FlowInstance, ni *entity.NodeInstance) bool {
	scripts := []struct {
		name    string
		content string
	}{
		{"pre_script", n.PreScript},
		{"script", n.Script},
		{"post_script", n.PostScript},
	}

	for _, spt := range scripts {
		if spt.content == "" {
			continue
		}
		result, err := s.runScript(ctx, n, inst, ni, spt.content)
		if err != nil {
			s.logger.Error("script execution error", "node_id", n.ID, "script", spt.name, "error", err)
			return false
		}
		if result.ExitCode != 0 {
			s.logger.Warn("script exited with non-zero code", "node_id", n.ID, "script", spt.name, "exit_code", result.ExitCode)
			return false
		}
	}
	return true
}

// taskRunResult 是单个任务节点脚本的执行结果。
// exportedEnv 汇总该节点各脚本导出的环境变量，由调度器在全部完成后串行合并。
type taskRunResult struct {
	ok          bool
	skipped     bool
	exportedEnv map[string]string
}

// executeTaskNodesConcurrently 并发执行同一轮就绪的任务节点。
// 各节点基于启动时的同一份环境快照运行；导出的环境变量在全部完成后串行合并，
// 避免并发修改 inst.Env 产生数据竞争。失败不影响其他并发节点的执行。
func (s *Scheduler) executeTaskNodesConcurrently(
	ctx context.Context,
	readyNodes []*entity.Node,
	inst *entity.FlowInstance,
	nodeInstMap map[int64]*entity.NodeInstance,
	state map[int64]int,
) {
	baseEnv := make(map[string]string, len(inst.Params)+len(inst.Env)+1)
	for k, v := range inst.Params {
		baseEnv[k] = v
	}
	for k, v := range inst.Env {
		baseEnv[k] = v
	}

	results := make([]taskRunResult, len(readyNodes))
	var wg sync.WaitGroup
	for i, n := range readyNodes {
		ni := nodeInstMap[n.ID]
		s.transitionNodeInst(ctx, ni, valueobject.NodeInstanceStatusRunning)
		// 解析节点执行位置：本地 / 远程 SSH / k8s Pod。
		exec, err := s.resolveExecutor(ctx, n)
		if err != nil {
			s.logger.Error("resolve executor failed", "node_id", n.ID, "error", err)
			state[n.ID] = nodeFailed
			s.completeNodeInst(ctx, ni, valueobject.NodeInstanceStatusFailed, nil)
			s.markInstanceFailed(ctx, inst)
			continue
		}
		wg.Add(1)
		go func(i int, n *entity.Node, ni *entity.NodeInstance, exec executor.Executor) {
			defer wg.Done()
			// 每个节点独立可取消：手动跳过时终止脚本执行。
			runCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			s.registerCancel(ni.ID, cancel)
			defer s.popCancel(ni.ID)

			res := s.runTaskScripts(runCtx, exec, n, inst.ID, ni, baseEnv)
			results[i] = res
			// 已被手动跳过（终态）：不覆盖状态，结果由串行阶段按跳过处理。
			if ni.Status == valueobject.NodeInstanceStatusSkipped {
				return
			}
			// 节点实例相互独立，各自完成时立即落终态（前端状态实时变化）。
			switch {
			case res.skipped:
				s.completeNodeInst(ctx, ni, valueobject.NodeInstanceStatusFailedSkipped, nil)
			case !res.ok:
				s.completeNodeInst(ctx, ni, valueobject.NodeInstanceStatusFailed, ni.OutputEnv)
			default:
				s.completeNodeInst(ctx, ni, valueobject.NodeInstanceStatusSucceeded, ni.OutputEnv)
			}
		}(i, n, ni, exec)
	}
	wg.Wait()

	// 实例级共享状态（环境变量、state 表、失败标记）在批次结束后串行应用。
	anyFailed := false
	for i, n := range readyNodes {
		ni := nodeInstMap[n.ID]
		res := results[i]
		// 手动跳过的节点：视为成功继续调度，不再合并被终止脚本导出的环境变量。
		if ni.Status == valueobject.NodeInstanceStatusSkipped {
			state[n.ID] = nodeDone
			continue
		}
		s.mergeExportedEnv(ctx, inst, ni, res.exportedEnv)
		switch {
		case res.skipped:
			state[n.ID] = nodeSkipped
		case !res.ok:
			state[n.ID] = nodeFailed
			anyFailed = true
		default:
			state[n.ID] = nodeDone
		}
	}
	if anyFailed {
		s.markInstanceFailed(ctx, inst)
	}
}

// runTaskScripts 按顺序执行 start_script / pre_script / script / post_script。
// 在独立 goroutine 中运行，只读取传入的环境快照并写入本地副本，
// 导出的环境变量通过返回值交由调用方串行合并。
func (s *Scheduler) runTaskScripts(
	ctx context.Context,
	exec executor.Executor,
	n *entity.Node,
	instanceID int64,
	ni *entity.NodeInstance,
	baseEnv map[string]string,
) taskRunResult {
	env := make(map[string]string, len(baseEnv)+1)
	for k, v := range baseEnv {
		env[k] = v
	}
	env[executor.EnvOutFileKey] = s.envOutPath(instanceID, ni.NodeID)
	exported := map[string]string{}
	merge := func(e map[string]string) {
		for k, v := range e {
			exported[k] = v
			env[k] = v
		}
	}

	if n.StartScript != "" {
		result, err := exec.Execute(ctx, executor.ExecuteSpec{InstanceID: instanceID, Command: n.StartScript, Env: env})
		if err != nil {
			s.logger.Error("start script execution failed", "node_id", n.ID, "error", err)
			return taskRunResult{skipped: true, exportedEnv: exported}
		}
		if result.ExitCode != 0 {
			return taskRunResult{skipped: true, exportedEnv: exported}
		}
		merge(result.ExportedEnv)
	}

	scripts := []struct {
		name    string
		content string
	}{
		{"pre_script", n.PreScript},
		{"script", n.Script},
		{"post_script", n.PostScript},
	}
	for _, spt := range scripts {
		if spt.content == "" {
			continue
		}
		result, err := exec.Execute(ctx, executor.ExecuteSpec{InstanceID: instanceID, Command: spt.content, Env: env})
		if err != nil {
			s.logger.Error("script execution error", "node_id", n.ID, "script", spt.name, "error", err)
			return taskRunResult{exportedEnv: exported}
		}
		if result.ExitCode != 0 {
			s.logger.Warn("script exited with non-zero code", "node_id", n.ID, "script", spt.name, "exit_code", result.ExitCode)
			return taskRunResult{exportedEnv: exported}
		}
		merge(result.ExportedEnv)
	}
	return taskRunResult{ok: true, exportedEnv: exported}
}

func (s *Scheduler) transitionNodeInst(ctx context.Context, ni *entity.NodeInstance, target valueobject.NodeInstanceStatus) {
	if err := ni.Transition(target); err != nil {
		s.logger.Error("node instance transition failed", "ni_id", ni.ID, "target", target, "error", err)
		return
	}
	_ = s.instRepo.UpdateNodeInstance(ctx, ni)
}

func (s *Scheduler) completeNodeInst(ctx context.Context, ni *entity.NodeInstance, status valueobject.NodeInstanceStatus, outputEnv map[string]string) {
	if err := ni.Transition(status); err != nil {
		s.logger.Debug("node instance already terminal", "ni_id", ni.ID, "status", ni.Status)
	}
	if outputEnv != nil {
		ni.SetOutputEnv(outputEnv)
	}
	_ = s.instRepo.UpdateNodeInstance(ctx, ni)
}
