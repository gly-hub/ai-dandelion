package system

import (
	"context"
	"strconv"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gofiber/fiber/v2"
)

// ListOperationLogs
// @tags System
// @summary 查询操作记录
// @router /system/operation-logs [GET]
func (s *SystemServerController) ListOperationLogs(ctx *fiber.Ctx) error {
	rpcParam := &systemproto.ListOperationLogsReq{
		Module:       ctx.Query("module"),
		Action:       ctx.Query("action"),
		ResourceType: ctx.Query("resourceType"),
		ResourceId:   ctx.Query("resourceId"),
		OperatorId:   ctx.Query("operatorId"),
		Keyword:      ctx.Query("keyword"),
		Page:         int32(queryPositiveInt(ctx, "page", 1)),
		PageSize:     int32(queryPositiveInt(ctx, "pageSize", 20)),
	}
	handler := func(ctx context.Context, req *systemproto.ListOperationLogsReq) (interface{}, error) {
		client, err := s.getSystemClient(ctx)
		if err != nil {
			return nil, err
		}
		return client.ListOperationLogs(ctx, req)
	}
	return s.baseHandler.GRPCCall(ctx, rpcParam, handler)
}

func queryPositiveInt(ctx *fiber.Ctx, key string, fallback int) int {
	value, err := strconv.Atoi(ctx.Query(key))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
