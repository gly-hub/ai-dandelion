package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/team-dandelion/ai-dandelion/func-operation/internal/dao"
	"github.com/team-dandelion/ai-dandelion/func-operation/internal/model"
	"github.com/team-dandelion/ai-dandelion/func-operation/internal/runtime/generatedapp"
	"github.com/team-dandelion/ai-dandelion/toolbox/authctx"
	"github.com/team-dandelion/ai-dandelion/toolbox/eventbus"
	"gorm.io/gorm"
)

type ReleaseLogic struct {
	releases  *dao.FunctionRelease
	outbox    *dao.FunctionOutbox
	runtime   *generatedapp.Service
	functions *dao.Function
	bus       eventbus.Bus
}

func NewReleaseLogic(releases *dao.FunctionRelease, outbox *dao.FunctionOutbox, runtime *generatedapp.Service, functions *dao.Function, buses ...eventbus.Bus) *ReleaseLogic {
	var bus eventbus.Bus
	if len(buses) > 0 {
		bus = buses[0]
	}
	return &ReleaseLogic{releases: releases, outbox: outbox, runtime: runtime, functions: functions, bus: bus}
}

// BackfillLegacyPublished prevents an upgrade from taking trusted legacy
// published functions offline. The artifact is validated and fingerprinted
// before the release record is committed, so future boots use the normal
// published-release path.
func (r *ReleaseLogic) BackfillLegacyPublished(ctx context.Context) error {
	if r == nil || r.releases == nil || r.runtime == nil || r.functions == nil {
		return errors.New("release runtime is not configured")
	}
	functions, err := r.functions.ListLegacyPublished(ctx)
	if err != nil {
		return err
	}
	for i := range functions {
		function := &functions[i]
		snapshot, err := r.runtime.ArtifactSnapshot(function.GeneratedAppID)
		if err != nil {
			return err
		}
		if err := r.runtime.PromoteArtifact(snapshot.AppID, snapshot.SHA256); err != nil {
			return err
		}
		if _, err := r.releases.BackfillPublished(ctx, function.UUID, snapshot.AppID, snapshot.SHA256, snapshot.ManifestJSON, nowUnixMicro()); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReleaseLogic) RestorePublished(ctx context.Context) error {
	if r == nil || r.releases == nil || r.runtime == nil {
		return errors.New("release runtime is not configured")
	}
	releases, err := r.releases.ListPublished(ctx)
	if err != nil {
		return err
	}
	approvals := make(map[string]string, len(releases))
	for _, release := range releases {
		approvals[release.AppID] = release.ArtifactSHA256
	}
	return r.runtime.LoadApprovedArtifacts(ctx, approvals)
}

// RevokeOrphanedPublished prevents historical function deletes from leaving a
// runnable release without an owning function.
func (r *ReleaseLogic) RevokeOrphanedPublished(ctx context.Context) error {
	if r == nil || r.releases == nil || r.functions == nil {
		return errors.New("release runtime is not configured")
	}
	releases, err := r.releases.ListPublished(ctx)
	if err != nil {
		return err
	}
	for _, release := range releases {
		exists, existsErr := r.functions.Exists(ctx, release.FunctionID)
		if existsErr != nil {
			return existsErr
		}
		if !exists {
			if err := r.releases.RevokeByFunctionID(ctx, release.FunctionID, nowUnixMicro()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *ReleaseLogic) RevokeFunctionReleases(ctx context.Context, functionID string) error {
	if r == nil || r.releases == nil {
		return nil
	}
	return r.releases.RevokeByFunctionID(ctx, functionID, nowUnixMicro())
}

// ReconcileArtifactStore verifies every published release before removing
// unreferenced artifacts through the configured storage backend. A validation
// error is returned to the caller so startup and runtime access remain
// fail-closed.
func (r *ReleaseLogic) ReconcileArtifactStore(ctx context.Context, staleStagingAfter time.Duration) (generatedapp.ArtifactReconcileResult, error) {
	if r == nil || r.releases == nil || r.runtime == nil {
		return generatedapp.ArtifactReconcileResult{}, errors.New("release runtime is not configured")
	}
	releases, err := r.releases.ListAll(ctx)
	if err != nil {
		return generatedapp.ArtifactReconcileResult{}, err
	}
	referenced := make(map[string]struct{}, len(releases))
	for _, release := range releases {
		referenced[release.ArtifactSHA256] = struct{}{}
		if release.Status != model.FunctionReleaseStatusPublished {
			continue
		}
		if _, err := r.runtime.PublishedArtifactSnapshot(release.AppID, release.ArtifactSHA256); err != nil {
			return generatedapp.ArtifactReconcileResult{}, fmt.Errorf("validate published release %q: %w", release.UUID, err)
		}
	}
	return r.runtime.ReconcileArtifacts(ctx, referenced, staleStagingAfter)
}

func (r *ReleaseLogic) Stage(ctx context.Context, function *model.Function) (*model.FunctionRelease, error) {
	if r == nil || r.releases == nil || r.runtime == nil || function == nil {
		return nil, errors.New("release runtime is not configured")
	}
	appID := strings.TrimSpace(function.GeneratedAppID)
	if appID == "" {
		return nil, errors.New("generated app is not ready")
	}
	snapshot, err := r.runtime.ArtifactSnapshot(appID)
	if err != nil {
		return nil, err
	}
	latest, err := r.releases.Latest(ctx, function.UUID)
	if err == nil && latest.AppID == appID && latest.ArtifactSHA256 == snapshot.SHA256 && latest.Status != model.FunctionReleaseStatusRevoked {
		return latest, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	version := int64(1)
	if latest != nil {
		version = latest.Version + 1
	}
	userID, _ := authctx.RequireUserID(ctx)
	now := nowUnixMicro()
	release := &model.FunctionRelease{
		FunctionID:     function.UUID,
		AppID:          appID,
		Version:        version,
		ArtifactSHA256: snapshot.SHA256,
		ManifestJSON:   snapshot.ManifestJSON,
		Status:         model.FunctionReleaseStatusStaged,
		CreatedBy:      userID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := r.releases.CreateStaged(ctx, release); err != nil {
		return nil, err
	}
	return release, nil
}

func (r *ReleaseLogic) Publish(ctx context.Context, function *model.Function) (*model.FunctionRelease, error) {
	release, err := r.Stage(ctx, function)
	if err != nil {
		return nil, err
	}
	if err := r.runtime.ApplyDeclaredDataModels(ctx, release.AppID); err != nil {
		return nil, err
	}
	if err := r.runtime.PromoteArtifact(release.AppID, release.ArtifactSHA256); err != nil {
		return nil, err
	}
	now := nowUnixMicro()
	payload, err := json.Marshal(functionVersionPublishedPayload{
		FunctionID:   function.UUID,
		FunctionName: function.Name,
		Version:      release.Version,
		ReleaseID:    release.UUID,
		PublishedAt:  now,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal function release event: %w", err)
	}
	var outboxEvent *model.FunctionOutboxEvent
	if r.outbox != nil {
		outboxEvent = &model.FunctionOutboxEvent{
			FunctionID: function.UUID, ReleaseID: release.UUID, EventType: FunctionVersionPublishedEventType,
			PayloadJSON: string(payload), Status: "pending", CreatedAt: now, UpdatedAt: now,
		}
	}
	if outboxEvent != nil {
		if err := r.releases.PublishWithOutbox(ctx, release.UUID, now, outboxEvent); err != nil {
			return nil, fmt.Errorf("publish function release: %w", err)
		}
	} else if err := r.releases.Publish(ctx, release.UUID, now); err != nil {
		return nil, err
	}
	function.ActiveReleaseID = release.UUID
	if err := r.RestorePublished(ctx); err != nil {
		return nil, err
	}
	release.Status = model.FunctionReleaseStatusPublished
	release.PublishedAt = now
	release.UpdatedAt = now
	return release, nil
}

const FunctionVersionPublishedEventType = "func-operation.function.version-published"
const FunctionUnpublishedEventType = "func-operation.function.unpublished"

type functionVersionPublishedPayload struct {
	FunctionID   string `json:"functionId"`
	FunctionName string `json:"functionName"`
	Version      int64  `json:"version"`
	ReleaseID    string `json:"releaseId"`
	PublishedAt  int64  `json:"publishedAt"`
}

type functionUnpublishedPayload struct {
	FunctionID    string `json:"functionId"`
	FunctionName  string `json:"functionName"`
	UnpublishedAt int64  `json:"unpublishedAt"`
}

func (r *ReleaseLogic) RecordFunctionUnpublished(ctx context.Context, function *model.Function) error {
	if r == nil || r.outbox == nil || function == nil {
		return errors.New("function release outbox is not configured")
	}
	now := nowUnixMicro()
	payload, err := json.Marshal(functionUnpublishedPayload{
		FunctionID: function.UUID, FunctionName: function.Name, UnpublishedAt: now,
	})
	if err != nil {
		return fmt.Errorf("marshal function unpublish event: %w", err)
	}
	event := &model.FunctionOutboxEvent{
		FunctionID:  function.UUID,
		ReleaseID:   function.ActiveReleaseID,
		EventType:   FunctionUnpublishedEventType,
		PayloadJSON: string(payload),
		Status:      "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := r.outbox.Create(ctx, event); err != nil {
		return err
	}
	r.tryDeliverRealtimeEvent(ctx, event)
	return nil
}

func (r *ReleaseLogic) DeliverFunctionPublished(ctx context.Context, releaseID string) {
	if r == nil || r.outbox == nil {
		return
	}
	event, err := r.outbox.GetByReleaseID(ctx, FunctionVersionPublishedEventType, releaseID)
	if err != nil {
		return
	}
	r.tryDeliverRealtimeEvent(ctx, event)
}

// tryDeliverRealtimeEvent keeps the UI responsive while the outbox remains
// the durable retry path for transient Redis or gateway failures.
func (r *ReleaseLogic) tryDeliverRealtimeEvent(ctx context.Context, outboxEvent *model.FunctionOutboxEvent) {
	if r == nil || r.bus == nil || r.outbox == nil || outboxEvent == nil {
		return
	}
	_, err := publishFunctionRealtimeEvent(ctx, r.bus, *outboxEvent)
	if err != nil {
		return
	}
	if outboxEvent.UUID != "" {
		_ = r.outbox.MarkDoneByUUID(ctx, outboxEvent.UUID, nowUnixMicro())
	}
}

func (r *ReleaseLogic) RequireActiveArtifact(ctx context.Context, appID string) (*model.FunctionRelease, error) {
	if r == nil || r.releases == nil || r.runtime == nil {
		return nil, errors.New("release runtime is not configured")
	}
	release, err := r.releases.ActiveForApp(ctx, strings.TrimSpace(appID))
	if err != nil {
		return nil, err
	}
	if !r.runtime.HasApprovedArtifact(release.AppID, release.ArtifactSHA256) {
		return nil, errors.New("active artifact does not match its published release")
	}
	return release, nil
}

func (r *ReleaseLogic) PublishedAppIDs(ctx context.Context) (map[string]struct{}, error) {
	if r == nil || r.releases == nil {
		return nil, errors.New("release runtime is not configured")
	}
	releases, err := r.releases.ListPublished(ctx)
	if err != nil {
		return nil, err
	}
	ids := make(map[string]struct{}, len(releases))
	for _, release := range releases {
		ids[release.AppID] = struct{}{}
	}
	return ids, nil
}

func (r *ReleaseLogic) RecordEvent(ctx context.Context, functionID, releaseID, eventType string, payload any) {
	if r == nil || r.outbox == nil {
		return
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	now := nowUnixMicro()
	_ = r.outbox.Create(ctx, &model.FunctionOutboxEvent{
		FunctionID:  functionID,
		ReleaseID:   releaseID,
		EventType:   eventType,
		PayloadJSON: string(raw),
		Status:      "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
}
