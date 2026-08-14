package logic

import (
	"context"
	"errors"
	"strings"

	systemproto "github.com/team-dandelion/ai-dandelion/proto/system"
	"github.com/team-dandelion/ai-dandelion/toolbox/authctx"
)

type FunctionMenuSync struct {
	system systemproto.SystemServiceClient
}

func NewFunctionMenuSync(system systemproto.SystemServiceClient) *FunctionMenuSync {
	return &FunctionMenuSync{system: system}
}

func (f *FunctionMenuSync) SyncPublished(ctx context.Context, functionID, name, parentID string, actionKeys []string) (string, error) {
	if f == nil || f.system == nil {
		return "", nil
	}
	resp, err := f.system.SyncGeneratedFunctionMenu(authctx.ForwardUserContext(ctx), &systemproto.SyncGeneratedFunctionMenuReq{
		FunctionId: strings.TrimSpace(functionID),
		Name:       strings.TrimSpace(name),
		ParentId:   strings.TrimSpace(parentID),
		Action:     "publish",
		ActionKeys: normalizeActionKeys(actionKeys),
	})
	if err != nil {
		return "", err
	}
	if resp.GetMenu() == nil {
		return "", nil
	}
	return resp.GetMenu().GetId(), nil
}

func (f *FunctionMenuSync) SyncMetadata(ctx context.Context, functionID, name, parentID string, actionKeys []string) (string, error) {
	if f == nil || f.system == nil {
		return "", nil
	}
	resp, err := f.system.SyncGeneratedFunctionMenu(authctx.ForwardUserContext(ctx), &systemproto.SyncGeneratedFunctionMenuReq{
		FunctionId: strings.TrimSpace(functionID),
		Name:       strings.TrimSpace(name),
		ParentId:   strings.TrimSpace(parentID),
		Action:     "sync",
		ActionKeys: normalizeActionKeys(actionKeys),
	})
	if err != nil {
		return "", err
	}
	if resp.GetMenu() == nil {
		return "", nil
	}
	return resp.GetMenu().GetId(), nil
}

func (f *FunctionMenuSync) Unpublish(ctx context.Context, functionID string) error {
	if f == nil || f.system == nil {
		return nil
	}
	_, err := f.system.SyncGeneratedFunctionMenu(authctx.ForwardUserContext(ctx), &systemproto.SyncGeneratedFunctionMenuReq{
		FunctionId: strings.TrimSpace(functionID),
		Action:     "unpublish",
	})
	return err
}

func (f *FunctionMenuSync) Delete(ctx context.Context, functionID string) error {
	if f == nil || f.system == nil {
		return nil
	}
	_, err := f.system.SyncGeneratedFunctionMenu(authctx.ForwardUserContext(ctx), &systemproto.SyncGeneratedFunctionMenuReq{
		FunctionId: strings.TrimSpace(functionID),
		Action:     "delete",
	})
	return err
}

func (f *FunctionMenuSync) CheckAccess(ctx context.Context, userID, functionID, actionKey string) (bool, error) {
	if ctxUserID, err := authctx.RequireUserID(ctx); err == nil {
		userID = ctxUserID
	}
	if f == nil || f.system == nil {
		return false, errors.New("function menu authorization is not configured")
	}
	if strings.TrimSpace(userID) == "" {
		return false, nil
	}
	resp, err := f.system.CheckFunctionMenuAccess(authctx.ForwardUserContext(ctx), &systemproto.CheckFunctionMenuAccessReq{
		UserId:     strings.TrimSpace(userID),
		FunctionId: strings.TrimSpace(functionID),
		ActionKey:  strings.TrimSpace(actionKey),
	})
	if err != nil {
		return false, err
	}
	return resp.GetAllowed(), nil
}

func (f *FunctionMenuSync) SyncPublishedActions(ctx context.Context, functionID, name, parentID string, actionKeys []string) (string, error) {
	if f == nil || f.system == nil {
		return "", nil
	}
	resp, err := f.system.SyncGeneratedFunctionMenu(authctx.ForwardUserContext(ctx), &systemproto.SyncGeneratedFunctionMenuReq{
		FunctionId: strings.TrimSpace(functionID),
		Name:       strings.TrimSpace(name),
		ParentId:   strings.TrimSpace(parentID),
		Action:     "sync",
		ActionKeys: normalizeActionKeys(actionKeys),
	})
	if err != nil {
		return "", err
	}
	if resp.GetMenu() == nil {
		return "", nil
	}
	return resp.GetMenu().GetId(), nil
}

func normalizeActionKeys(items []string) []string {
	normalized := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, key)
	}
	return normalized
}
