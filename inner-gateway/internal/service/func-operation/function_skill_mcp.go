package funcoperation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"github.com/gofiber/fiber/v2"
)

const functionSkillMCPProtocolVersion = "2024-11-05"

type functionSkillMCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type functionSkillMCPResponse struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      json.RawMessage        `json:"id"`
	Result  any                    `json:"result,omitempty"`
	Error   *functionSkillMCPError `json:"error,omitempty"`
}

type functionSkillMCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// FunctionSkillMCP is a small Streamable HTTP MCP transport adapter. Authorization,
// permission checks, and execution remain in func-operation.
func (f *FuncOperationServerController) FunctionSkillMCP(ctx *fiber.Ctx) error {
	if ctx.Method() != fiber.MethodPost {
		return ctx.SendStatus(fiber.StatusMethodNotAllowed)
	}
	var request functionSkillMCPRequest
	if err := json.Unmarshal(ctx.Body(), &request); err != nil {
		return functionSkillMCPErrorResponse(ctx, request.ID, -32700, "invalid JSON-RPC request")
	}
	if request.JSONRPC != "2.0" || strings.TrimSpace(request.Method) == "" {
		return functionSkillMCPErrorResponse(ctx, request.ID, -32600, "invalid JSON-RPC request")
	}
	if len(request.ID) == 0 || string(request.ID) == "null" {
		return ctx.SendStatus(fiber.StatusAccepted)
	}
	grantToken := functionSkillMCPBearerToken(ctx.Get(fiber.HeaderAuthorization))
	if grantToken == "" {
		return functionSkillMCPErrorResponse(ctx, request.ID, -32001, "function skill grant is required")
	}
	client, err := f.getFuncOperationClient(context.Background())
	if err != nil {
		return functionSkillMCPErrorResponse(ctx, request.ID, -32603, "function skill service unavailable")
	}
	var result any
	switch request.Method {
	case "initialize":
		result = fiber.Map{
			"protocolVersion": functionSkillMCPProtocolVersion,
			"capabilities":    fiber.Map{"tools": fiber.Map{}},
			"serverInfo":      fiber.Map{"name": "function-skills", "version": "1.0.0"},
		}
	case "ping":
		result = fiber.Map{}
	case "tools/list":
		tools, callErr := client.GetFunctionSkillTools(context.Background(), &funcoperation.GetFunctionSkillToolsReq{GrantToken: grantToken})
		if callErr != nil {
			return functionSkillMCPErrorResponse(ctx, request.ID, -32001, callErr.Error())
		}
		items := make([]fiber.Map, 0, len(tools.GetTools()))
		for _, tool := range tools.GetTools() {
			var inputSchema any
			if err := json.Unmarshal([]byte(tool.GetInputSchemaJson()), &inputSchema); err != nil {
				continue
			}
			items = append(items, fiber.Map{"name": tool.GetName(), "description": tool.GetDescription(), "inputSchema": inputSchema})
		}
		result = fiber.Map{"tools": items}
	case "tools/call":
		name, input, approvalToken, parseErr := functionSkillMCPCallParams(request.Params)
		if parseErr != nil {
			return functionSkillMCPErrorResponse(ctx, request.ID, -32602, parseErr.Error())
		}
		inputJSON, err := json.Marshal(input)
		if err != nil {
			return functionSkillMCPErrorResponse(ctx, request.ID, -32602, "tool arguments are invalid")
		}
		out, callErr := client.ExecuteFunctionSkill(context.Background(), &funcoperation.ExecuteFunctionSkillReq{
			GrantToken: grantToken, ToolName: name, ToolUseId: functionSkillMCPToolUseID(request.ID), InputJson: string(inputJSON), ApprovalToken: approvalToken,
		})
		if callErr != nil {
			return functionSkillMCPToolResult(ctx, request.ID, "", callErr.Error(), true)
		}
		return functionSkillMCPToolResult(ctx, request.ID, out.GetResultJson(), out.GetErrorMessage(), out.GetIsError())
	default:
		return functionSkillMCPErrorResponse(ctx, request.ID, -32601, "method not found")
	}
	return ctx.JSON(functionSkillMCPResponse{JSONRPC: "2.0", ID: request.ID, Result: result})
}

func functionSkillMCPCallParams(raw json.RawMessage) (string, map[string]any, string, error) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return "", nil, "", errors.New("tool call parameters are invalid")
	}
	if strings.TrimSpace(params.Name) == "" {
		return "", nil, "", errors.New("tool name is required")
	}
	if params.Arguments == nil {
		params.Arguments = make(map[string]any)
	}
	approval, _ := params.Arguments["__function_skill_approval"].(string)
	delete(params.Arguments, "__function_skill_approval")
	return strings.TrimSpace(params.Name), params.Arguments, strings.TrimSpace(approval), nil
}

func functionSkillMCPToolResult(ctx *fiber.Ctx, id json.RawMessage, resultJSON, errorMessage string, isError bool) error {
	text := strings.TrimSpace(resultJSON)
	if isError {
		text = strings.TrimSpace(errorMessage)
		if text == "" {
			text = "function skill execution failed"
		}
	}
	return ctx.JSON(functionSkillMCPResponse{JSONRPC: "2.0", ID: id, Result: fiber.Map{"content": []fiber.Map{{"type": "text", "text": text}}, "isError": isError}})
}

func functionSkillMCPErrorResponse(ctx *fiber.Ctx, id json.RawMessage, code int, message string) error {
	return ctx.JSON(functionSkillMCPResponse{JSONRPC: "2.0", ID: id, Error: &functionSkillMCPError{Code: code, Message: message}})
}

func functionSkillMCPBearerToken(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 7 && strings.EqualFold(value[:7], "bearer ") {
		return strings.TrimSpace(value[7:])
	}
	return ""
}

func functionSkillMCPToolUseID(id json.RawMessage) string {
	var value any
	if err := json.Unmarshal(id, &value); err != nil {
		return string(id)
	}
	return fmt.Sprint(value)
}
