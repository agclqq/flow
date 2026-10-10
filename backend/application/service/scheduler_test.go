package service

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"flow/domain/entity"
	"flow/domain/executor"
	"flow/domain/repository"
	"flow/domain/valueobject"
)

type memInstRepo struct {
	mu        sync.Mutex
	instances map[int64]*entity.FlowInstance
	nodeInsts map[int64][]*entity.NodeInstance
	nextNIID  int64
}

func newMemInstRepo() *memInstRepo {
	return &memInstRepo{
		instances: map[int64]*entity.FlowInstance{},
		nodeInsts: map[int64][]*entity.NodeInstance{},
		nextNIID:  1,
	}
}

func (r *memInstRepo) Create(_ context.Context, inst *entity.FlowInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.instances[inst.ID] = inst
	return nil
}

func (r *memInstRepo) GetByID(_ context.Context, id int64) (*entity.FlowInstance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	inst, ok := r.instances[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return inst, nil
}

func (r *memInstRepo) Update(_ context.Context, inst *entity.FlowInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.instances[inst.ID] = inst
	return nil
}

func (r *memInstRepo) ListByTemplate(context.Context, int64, int, int) ([]*entity.FlowInstance, int64, error) {
	return nil, 0, nil
}

func (r *memInstRepo) ListNodeInstances(_ context.Context, flowInstanceID int64) ([]*entity.NodeInstance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.nodeInsts[flowInstanceID], nil
}

func (r *memInstRepo) CreateNodeInstance(_ context.Context, ni *entity.NodeInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ni.ID = r.nextNIID
	r.nextNIID++
	r.nodeInsts[ni.FlowInstanceID] = append(r.nodeInsts[ni.FlowInstanceID], ni)
	return nil
}

func (r *memInstRepo) UpdateNodeInstance(_ context.Context, ni *entity.NodeInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, existing := range r.nodeInsts[ni.FlowInstanceID] {
		if existing.ID == ni.ID {
			r.nodeInsts[ni.FlowInstanceID][i] = ni
			return nil
		}
	}
	return nil
}

type memVersionRepo struct {
	versions map[int64]*entity.FlowTemplateVersion
}

func (r *memVersionRepo) Create(context.Context, *entity.FlowTemplateVersion) error { return nil }
func (r *memVersionRepo) GetByID(_ context.Context, id int64) (*entity.FlowTemplateVersion, error) {
	ver, ok := r.versions[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return ver, nil
}
func (r *memVersionRepo) GetByTemplateAndVersion(context.Context, int64, int) (*entity.FlowTemplateVersion, error) {
	return nil, repository.ErrNotFound
}

func (r *memVersionRepo) GetLatestPublishedByTemplate(context.Context, int64) (*entity.FlowTemplateVersion, error) {
	return nil, repository.ErrNotFound
}
func (r *memVersionRepo) ListByTemplate(context.Context, int64) ([]*entity.FlowTemplateVersion, error) {
	return nil, nil
}
func (r *memVersionRepo) ListPublishedCronTriggers(context.Context) ([]entity.PublishedCronRule, error) {
	return nil, nil
}
func (r *memVersionRepo) Update(context.Context, *entity.FlowTemplateVersion) error { return nil }

func buildParallelJoinPipeline() *entity.FlowTemplateVersion {
	ver, _ := entity.NewFlowTemplateVersion(1, 1)
	ver.ID = 1

	start, _ := entity.NewNode(1, 1, valueobject.NodeTypeStart, "start")
	start.ID = 1
	printABC, _ := entity.NewNode(1, 1, valueobject.NodeTypeTask, "print_abc")
	printABC.ID = 2
	printABC.Script = `echo "a=$a b=$b c=$c"`
	nodeA, _ := entity.NewNode(1, 1, valueobject.NodeTypeTask, "node_a")
	nodeA.ID = 3
	nodeA.StartScript = `[ "$a" = "1" ]`
	nodeA.Script = `echo "d=1" >> "$FLOW_ENV_OUT"`
	nodeB, _ := entity.NewNode(1, 1, valueobject.NodeTypeTask, "node_b")
	nodeB.ID = 4
	nodeB.StartScript = `[ "$b" = "1" ]`
	nodeB.Script = `echo "e=1" >> "$FLOW_ENV_OUT"`
	nodeC, _ := entity.NewNode(1, 1, valueobject.NodeTypeTask, "node_c")
	nodeC.ID = 5
	nodeC.StartScript = `[ "$c" = "1" ]`
	nodeC.Script = `echo "f=1" >> "$FLOW_ENV_OUT"`
	mergeNode, _ := entity.NewNode(1, 1, valueobject.NodeTypeTask, "merge")
	mergeNode.ID = 6
	mergeNode.StartMode = valueobject.StartModeCustom
	mergeNode.RequiredPredecessorIDs = []int64{3, 4}
	mergeNode.Script = `echo "d=$d e=$e f=$f"`
	end, _ := entity.NewNode(1, 1, valueobject.NodeTypeEnd, "end")
	end.ID = 7

	edgeStartPrint, _ := entity.NewNode(1, 1, valueobject.NodeTypeEdge, "start_to_print")
	edgeStartPrint.ID = 8
	_ = edgeStartPrint.SetEdge(1, 2)
	edgePrintA, _ := entity.NewNode(1, 1, valueobject.NodeTypeEdge, "print_to_a")
	edgePrintA.ID = 9
	_ = edgePrintA.SetEdge(2, 3)
	edgePrintB, _ := entity.NewNode(1, 1, valueobject.NodeTypeEdge, "print_to_b")
	edgePrintB.ID = 10
	_ = edgePrintB.SetEdge(2, 4)
	edgePrintC, _ := entity.NewNode(1, 1, valueobject.NodeTypeEdge, "print_to_c")
	edgePrintC.ID = 11
	_ = edgePrintC.SetEdge(2, 5)
	edgeAToMerge, _ := entity.NewNode(1, 1, valueobject.NodeTypeEdge, "a_to_merge")
	edgeAToMerge.ID = 12
	_ = edgeAToMerge.SetEdge(3, 6)
	edgeBToMerge, _ := entity.NewNode(1, 1, valueobject.NodeTypeEdge, "b_to_merge")
	edgeBToMerge.ID = 13
	_ = edgeBToMerge.SetEdge(4, 6)
	edgeCToMerge, _ := entity.NewNode(1, 1, valueobject.NodeTypeEdge, "c_to_merge")
	edgeCToMerge.ID = 14
	_ = edgeCToMerge.SetEdge(5, 6)
	edgeMergeEnd, _ := entity.NewNode(1, 1, valueobject.NodeTypeEdge, "merge_to_end")
	edgeMergeEnd.ID = 15
	_ = edgeMergeEnd.SetEdge(6, 7)

	ver.Nodes = []*entity.Node{
		start, printABC, nodeA, nodeB, nodeC, mergeNode, end,
		edgeStartPrint, edgePrintA, edgePrintB, edgePrintC,
		edgeAToMerge, edgeBToMerge, edgeCToMerge, edgeMergeEnd,
	}
	return ver
}

func TestSchedulerParallelJoinPipeline(t *testing.T) {
	ver := buildParallelJoinPipeline()
	instRepo := newMemInstRepo()
	versionRepo := &memVersionRepo{versions: map[int64]*entity.FlowTemplateVersion{1: ver}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	scheduler := NewScheduler(executor.NewLocalExecutor(), nil, instRepo, versionRepo, logger, "")

	inst, err := entity.NewFlowInstance(1, 1, 1, valueobject.TriggerTypeManual)
	require.NoError(t, err)
	inst.ID = 1
	inst.SetParams(map[string]string{"a": "1", "b": "1", "c": "0"})
	require.NoError(t, instRepo.Create(context.Background(), inst))
	require.NoError(t, inst.Transition(valueobject.FlowInstanceStatusRunning))

	err = scheduler.Run(context.Background(), inst)
	require.NoError(t, err)
	assert.Equal(t, valueobject.FlowInstanceStatusSucceeded, inst.Status)
	assert.Equal(t, "1", inst.Env["d"])
	assert.Equal(t, "1", inst.Env["e"])
	_, hasF := inst.Env["f"]
	assert.False(t, hasF, "f should not be exported when node C is skipped")

	nis, err := instRepo.ListNodeInstances(context.Background(), inst.ID)
	require.NoError(t, err)

	statusByNode := map[int64]valueobject.NodeInstanceStatus{}
	for _, ni := range nis {
		statusByNode[ni.NodeID] = ni.Status
	}
	assert.Equal(t, valueobject.NodeInstanceStatusSucceeded, statusByNode[3])
	assert.Equal(t, valueobject.NodeInstanceStatusSucceeded, statusByNode[4])
	assert.Equal(t, valueobject.NodeInstanceStatusFailedSkipped, statusByNode[5])
	assert.Equal(t, valueobject.NodeInstanceStatusSucceeded, statusByNode[6])
}

func TestSchedulerEndNodeMarksInstanceSucceeded(t *testing.T) {
	start, _ := entity.NewNode(1, 1, valueobject.NodeTypeStart, "开始")
	start.ID = 1
	end, _ := entity.NewNode(1, 1, valueobject.NodeTypeEnd, "结束")
	end.ID = 2
	edge, _ := entity.NewNode(1, 1, valueobject.NodeTypeEdge, "start_to_end")
	edge.ID = 3
	require.NoError(t, edge.SetEdge(1, 2))

	ver := &entity.FlowTemplateVersion{ID: 1, TemplateID: 1, Version: 1}
	ver.Nodes = []*entity.Node{start, end, edge}

	instRepo := newMemInstRepo()
	versionRepo := &memVersionRepo{versions: map[int64]*entity.FlowTemplateVersion{1: ver}}
	scheduler := NewScheduler(executor.NewLocalExecutor(), nil, instRepo, versionRepo, slog.New(slog.NewTextHandler(io.Discard, nil)), "")

	inst, err := entity.NewFlowInstance(1, 1, 1, valueobject.TriggerTypeManual)
	require.NoError(t, err)
	inst.ID = 1
	require.NoError(t, instRepo.Create(context.Background(), inst))
	require.NoError(t, inst.Transition(valueobject.FlowInstanceStatusRunning))

	err = scheduler.Run(context.Background(), inst)
	require.NoError(t, err)
	assert.Equal(t, valueobject.FlowInstanceStatusSucceeded, inst.Status)
	assert.NotNil(t, inst.FinishedAt)

	nis, err := instRepo.ListNodeInstances(context.Background(), inst.ID)
	require.NoError(t, err)
	statusByNode := map[int64]valueobject.NodeInstanceStatus{}
	for _, ni := range nis {
		statusByNode[ni.NodeID] = ni.Status
	}
	assert.Equal(t, valueobject.NodeInstanceStatusSucceeded, statusByNode[1])
	assert.Equal(t, valueobject.NodeInstanceStatusSucceeded, statusByNode[2])
}
