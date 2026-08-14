package logic

import (
	"context"
	"strings"

	"github.com/google/uuid"
	systemproto "github.com/team-dandelion/ai-dandelion/proto/system"
	"github.com/team-dandelion/ai-dandelion/system/internal/dao"
	"github.com/team-dandelion/ai-dandelion/system/internal/model"
	"github.com/team-dandelion/ai-dandelion/toolbox/authctx"
)

const (
	OperationModuleSystem = "system"
	OperationResourceRole = "role"
	OperationResourceUser = "user"
	OperationResourceMenu = "menu"
)

type OperationLogInput struct {
	Module       string
	Action       string
	ActionLabel  string
	ResourceType string
	ResourceID   string
	ResourceName string
	Summary      string
	BeforeData   string
	AfterData    string
}

type OperationLogLogic struct {
	operationLogDao *dao.OperationLog
}

func NewOperationLogLogic(operationLogDao *dao.OperationLog) *OperationLogLogic {
	return &OperationLogLogic{operationLogDao: operationLogDao}
}

func (l *OperationLogLogic) Record(ctx context.Context, input OperationLogInput) error {
	return l.record(ctx, l.operationLogDao, input)
}

func (l *OperationLogLogic) RecordWithDAO(ctx context.Context, operationLogDao *dao.OperationLog, input OperationLogInput) error {
	return l.record(ctx, operationLogDao, input)
}

func (l *OperationLogLogic) ListOperationLogs(ctx context.Context, req *systemproto.ListOperationLogsReq) ([]*systemproto.OperationLog, int64, error) {
	page := int(req.GetPage())
	if page < 1 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	items, total, err := l.operationLogDao.List(ctx, dao.OperationLogListFilter{
		Module:       strings.TrimSpace(req.GetModule()),
		Action:       strings.TrimSpace(req.GetAction()),
		ResourceType: strings.TrimSpace(req.GetResourceType()),
		ResourceID:   strings.TrimSpace(req.GetResourceId()),
		OperatorID:   strings.TrimSpace(req.GetOperatorId()),
		Keyword:      strings.TrimSpace(req.GetKeyword()),
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]*systemproto.OperationLog, 0, len(items))
	for i := range items {
		out = append(out, modelOperationLogToProto(&items[i]))
	}
	return out, total, nil
}

func (l *OperationLogLogic) record(ctx context.Context, operationLogDao *dao.OperationLog, input OperationLogInput) error {
	operatorID, operatorName := "system", "系统"
	if user, ok := authctx.CurrentUser(ctx); ok {
		operatorID = user.ID
		if strings.TrimSpace(user.Username) != "" {
			operatorName = user.Username
		}
	}
	return operationLogDao.Create(ctx, &model.OperationLog{
		ID:           uuid.NewString(),
		Module:       strings.TrimSpace(input.Module),
		Action:       strings.TrimSpace(input.Action),
		ActionLabel:  strings.TrimSpace(input.ActionLabel),
		ResourceType: strings.TrimSpace(input.ResourceType),
		ResourceID:   strings.TrimSpace(input.ResourceID),
		ResourceName: strings.TrimSpace(input.ResourceName),
		OperatorID:   operatorID,
		OperatorName: operatorName,
		Summary:      strings.TrimSpace(input.Summary),
		BeforeData:   defaultAuditJSON(input.BeforeData),
		AfterData:    defaultAuditJSON(input.AfterData),
		CreatedAt:    nowUnixMicro(),
	})
}

func defaultAuditJSON(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "{}"
	}
	return value
}

func modelOperationLogToProto(item *model.OperationLog) *systemproto.OperationLog {
	if item == nil {
		return nil
	}
	return &systemproto.OperationLog{
		Id:           item.ID,
		Module:       item.Module,
		Action:       item.Action,
		ActionLabel:  item.ActionLabel,
		ResourceType: item.ResourceType,
		ResourceId:   item.ResourceID,
		ResourceName: item.ResourceName,
		OperatorId:   item.OperatorID,
		OperatorName: item.OperatorName,
		Summary:      item.Summary,
		BeforeData:   item.BeforeData,
		AfterData:    item.AfterData,
		CreatedAt:    item.CreatedAt,
	}
}
