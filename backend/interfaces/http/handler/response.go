package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"flow/domain/repository"
)

// Response 统一响应封装：{"code":0,"msg":"","data":{}}
// data 可以是对象或数组；无数据（nil）时不出现在 JSON 中。
type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

// 业务错误码分段定义：0 为成功，大于 0 为错误，同类错误一个段。
//
//	0          成功
//	1000~1999  通用错误（参数、未找到、状态冲突、内部错误等）
//	2000~2999  模板 / 触发相关
//	4000~4999  实例 / 节点执行相关
const (
	CodeOK = 0

	CodeInvalidParam  = 1001 // 参数错误
	CodeNotFound      = 1004 // 资源不存在
	CodeStateConflict = 1005 // 状态冲突 / 非法操作
	CodeInternal      = 1999 // 系统内部错误

	CodeTemplateNotPublished  = 2001 // 模板未发布（或没有可运行的已发布版本）
	CodeTriggerDisabled       = 2002 // 触发方式未开启
	CodeInvalidTriggerType    = 2003 // 触发类型非法
	CodeWebhookAuthFailed     = 2004 // webhook 鉴权失败
	CodeMissingRequiredParams = 2005 // 缺少必填参数

	CodeInstanceState = 4001 // 实例状态不允许该操作
	CodeNodeState     = 4002 // 节点状态不允许该操作
)

// ok 输出成功响应（HTTP 恒为 200，业务语义由 code 表达）。
func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Code: CodeOK, Data: data})
}

// writeError 输出统一格式的错误响应：按错误内容归类业务码。
func writeError(c *gin.Context, status int, err error) {
	c.JSON(http.StatusOK, Response{Code: classifyCode(status, err), Msg: err.Error()})
}

// badRequest 输出参数类错误。
func badRequest(c *gin.Context, err error) {
	writeError(c, http.StatusBadRequest, err)
}

// internalError 输出内部错误。
func internalError(c *gin.Context, err error) {
	writeError(c, http.StatusInternalServerError, err)
}

// templateBizErrors 模板 / 触发相关的业务错误消息 → 错误码。
var templateBizErrors = map[string]int{
	"template must be published before starting an instance": CodeTemplateNotPublished,
	"published version not found":                            CodeTemplateNotPublished,
	"template has no published version to run":               CodeTemplateNotPublished,
	"manual trigger is disabled for the published version":   CodeTriggerDisabled,
	"webhook trigger is disabled for the published version":  CodeTriggerDisabled,
	"invalid trigger type":                                   CodeInvalidTriggerType,
	"invalid webhook secret":                                 CodeWebhookAuthFailed,
}

// instanceBizErrors 实例 / 节点执行相关的业务错误消息 → 错误码。
var instanceBizErrors = map[string]int{
	"cannot stop a succeeded instance":        CodeInstanceState,
	"instance already failed":                 CodeInstanceState,
	"only failed instances can be retried":    CodeInstanceState,
	"instance is not running":                 CodeInstanceState,
	"instance does not belong to template":    CodeInstanceState,
	"node is not waiting for approval":        CodeNodeState,
	"node is not a manual node":               CodeNodeState,
	"only running nodes can be skipped":       CodeNodeState,
	"invalid node instance status transition": CodeNodeState,
}

// classifyCode 将错误归类为业务错误码；无法识别时按原 HTTP 语义落到通用段。
func classifyCode(status int, err error) int {
	if err == nil {
		return CodeInternal
	}
	if errors.Is(err, repository.ErrNotFound) {
		return CodeNotFound
	}
	msg := err.Error()
	if strings.Contains(msg, "缺少必填参数") {
		return CodeMissingRequiredParams
	}
	if code, ok := templateBizErrors[msg]; ok {
		return code
	}
	if code, ok := instanceBizErrors[msg]; ok {
		return code
	}
	switch status {
	case http.StatusBadRequest:
		return CodeInvalidParam
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusConflict:
		return CodeStateConflict
	default:
		return CodeInternal
	}
}
