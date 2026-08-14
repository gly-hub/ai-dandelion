package logic

import (
	"errors"
	"strings"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/runtime/generatedapp"
)

const (
	invokeErrorCodeFrontendLoad = "frontend_load_failed"
	invokeErrorCodeWasmCompile  = "wasm_compile_failed"
	invokeErrorCodeInvoke       = "invoke_failed"
	invokeErrorCodeSQLValidate  = "sql_validation_failed"
)

type invokeErrorDetail struct {
	ErrorCode    string
	ErrorMessage string
	Stage        string
	Hint         string
}

func classifyInvokeError(err error) invokeErrorDetail {
	if err == nil {
		return invokeErrorDetail{}
	}

	message := strings.TrimSpace(err.Error())
	lower := strings.ToLower(message)

	switch {
	case errors.Is(err, generatedapp.ErrAppNotFound):
		return invokeErrorDetail{
			ErrorCode:    invokeErrorCodeFrontendLoad,
			ErrorMessage: "功能页面尚未生成或尚未加载",
			Stage:        "runtime_lookup",
			Hint:         "请先生成页面，或点击重新加载页面代码",
		}
	case strings.Contains(lower, "compile generated app"),
		strings.Contains(lower, "decode wasm module"),
		strings.Contains(lower, "read wasm module"):
		return invokeErrorDetail{
			ErrorCode:    invokeErrorCodeWasmCompile,
			ErrorMessage: message,
			Stage:        "wasm_compile",
			Hint:         "页面后端代码编译失败，请让 AI 修复页面或重新生成",
		}
	case strings.Contains(lower, "instantiate generated app"),
		strings.Contains(lower, "initialize generated app"):
		return invokeErrorDetail{
			ErrorCode:    invokeErrorCodeWasmCompile,
			ErrorMessage: message,
			Stage:        "wasm_instantiate",
			Hint:         "页面后端初始化失败，请让 AI 修复页面",
		}
	case strings.Contains(lower, "missing export"),
		strings.Contains(lower, "unsupported handle signature"),
		strings.Contains(lower, "allocate request memory"),
		strings.Contains(lower, "read generated app"):
		return invokeErrorDetail{
			ErrorCode:    invokeErrorCodeInvoke,
			ErrorMessage: message,
			Stage:        "wasm_invoke",
			Hint:         "页面运行接口异常，请查看错误详情并让 AI 修复",
		}
	case strings.Contains(lower, "sql"),
		strings.Contains(lower, "ddl"),
		strings.Contains(lower, "schema"):
		return invokeErrorDetail{
			ErrorCode:    invokeErrorCodeSQLValidate,
			ErrorMessage: message,
			Stage:        "sql_execution",
			Hint:         "数据操作失败，请检查表结构或让 AI 修复后端逻辑",
		}
	default:
		return invokeErrorDetail{
			ErrorCode:    invokeErrorCodeInvoke,
			ErrorMessage: message,
			Stage:        "invoke",
			Hint:         "功能运行失败，请查看错误详情或让 AI 修复页面",
		}
	}
}
