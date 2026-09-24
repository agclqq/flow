package valueobject

// NodeType 表示节点的类型。
// 一条流水线中有且仅有一个 start 与一个 end，
// task 为普通执行节点，edge 为流转连线（仅拓扑），manual 为人工审批节点。
// join 已废弃，读取旧数据时仍识别，运行态按 task 处理。
type NodeType string

const (
	NodeTypeStart  NodeType = "start"
	NodeTypeEnd    NodeType = "end"
	NodeTypeTask   NodeType = "task"
	NodeTypeEdge   NodeType = "edge"
	NodeTypeJoin   NodeType = "join" // 已废弃，仅兼容旧数据
	NodeTypeManual NodeType = "manual"
)

// NodeType 的 tinyint 存储值。
// 0: start  1: end  2: task  3: edge  4: join（废弃） 5: manual
const (
	NodeTypeStartInt  int8 = 0
	NodeTypeEndInt    int8 = 1
	NodeTypeTaskInt   int8 = 2
	NodeTypeEdgeInt   int8 = 3
	NodeTypeJoinInt   int8 = 4
	NodeTypeManualInt int8 = 5
)

// Int 返回节点类型的 tinyint 存储值。
func (t NodeType) Int() int8 {
	switch t {
	case NodeTypeEnd:
		return NodeTypeEndInt
	case NodeTypeTask:
		return NodeTypeTaskInt
	case NodeTypeEdge:
		return NodeTypeEdgeInt
	case NodeTypeJoin:
		return NodeTypeJoinInt
	case NodeTypeManual:
		return NodeTypeManualInt
	default:
		return NodeTypeStartInt
	}
}

// NodeTypeFromInt 将 tinyint 值转换为 NodeType。
func NodeTypeFromInt(v int8) NodeType {
	switch v {
	case NodeTypeEndInt:
		return NodeTypeEnd
	case NodeTypeTaskInt:
		return NodeTypeTask
	case NodeTypeEdgeInt:
		return NodeTypeEdge
	case NodeTypeJoinInt:
		return NodeTypeJoin
	case NodeTypeManualInt:
		return NodeTypeManual
	default:
		return NodeTypeStart
	}
}

// Valid 校验节点类型是否合法（新建节点不允许 join）。
func (t NodeType) Valid() bool {
	switch t {
	case NodeTypeStart, NodeTypeEnd, NodeTypeTask, NodeTypeEdge, NodeTypeManual:
		return true
	default:
		return false
	}
}

// ValidStored 校验数据库中存储的节点类型（含废弃 join）。
func (t NodeType) ValidStored() bool {
	switch t {
	case NodeTypeStart, NodeTypeEnd, NodeTypeTask, NodeTypeEdge, NodeTypeJoin, NodeTypeManual:
		return true
	default:
		return false
	}
}

// String 返回节点类型的字符串表示。
func (t NodeType) String() string {
	return string(t)
}

// StartMode 表示节点启动门禁策略。
type StartMode string

const (
	// StartModeAny 任一前驱成功即可启动（默认）。
	StartModeAny StartMode = "any"
	// StartModeAll 所有前驱均成功才启动。
	StartModeAll StartMode = "all"
	// StartModeCustom 指定前驱子集 + 可选 start_script。
	StartModeCustom StartMode = "custom"
)

// StartMode 的 tinyint 存储值（复用原 join_type 列）。
// 0: all  1: any  2: custom
const (
	StartModeAllInt    int8 = 0
	StartModeAnyInt    int8 = 1
	StartModeCustomInt int8 = 2
)

// Int 返回启动策略的 tinyint 存储值。
func (m StartMode) Int() int8 {
	switch m {
	case StartModeAny:
		return StartModeAnyInt
	case StartModeCustom:
		return StartModeCustomInt
	default:
		return StartModeAllInt
	}
}

// StartModeFromInt 将 tinyint 值转换为 StartMode。
func StartModeFromInt(v int8) StartMode {
	switch v {
	case StartModeAnyInt:
		return StartModeAny
	case StartModeCustomInt:
		return StartModeCustom
	default:
		return StartModeAll
	}
}

// Valid 校验启动策略是否合法。
func (m StartMode) Valid() bool {
	return m == StartModeAny || m == StartModeAll || m == StartModeCustom
}

// String 返回启动策略的字符串表示。
func (m StartMode) String() string {
	return string(m)
}

// JoinType 已废弃，保留类型别名以兼容旧引用。
type JoinType = StartMode

const (
	JoinTypeAll = StartModeAll
	JoinTypeAny = StartModeAny
)

const (
	JoinTypeAllInt = StartModeAllInt
	JoinTypeAnyInt = StartModeAnyInt
)

// JoinTypeFromInt 已废弃，委托 StartModeFromInt。
func JoinTypeFromInt(v int8) JoinType {
	return StartModeFromInt(v)
}
