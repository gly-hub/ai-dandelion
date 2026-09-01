package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/dao"
	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
	"github.com/gly-hub/ai-dandelion/func-operation/internal/runtime/generatedapp"
	aiagent "github.com/gly-hub/ai-dandelion/proto/ai-agent"
	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	"github.com/google/uuid"
)

const aiAgentSessionTypeFunction = 2

const (
	functionDocumentSourceDraft     = "draft"
	functionDocumentSourceApplied   = "applied"
	functionDocumentSourcePublished = "published"
	functionDocumentTypeProduct     = "product"
	functionDocumentTypeTechnical   = "technical"

	functionConversationProduct    = "product"
	functionConversationTechnical  = "technical"
	functionConversationGeneration = "generation"
)

type FunctionLogic struct {
	functionDao            *dao.Function
	conversationOperations *dao.FunctionConversationOperation
	messageStore           *dao.AiAgentMessageStore
	generatedAppDao        *dao.GeneratedApp
	appRuntime             *generatedapp.Service
	previewRuntime         *generatedapp.Service
	aiAgentProvider        AiAgentClientProvider
	menuSync               *FunctionMenuSync
	authorizer             *FunctionAuthorizer
	releaseLogic           *ReleaseLogic
}

// AiAgentClientProvider resolves the downstream client only when an operation needs it.
type AiAgentClientProvider func(context.Context) (aiagent.AiAgentServiceClient, error)

type conversationInsights struct {
	Summary    string
	Highlights []string
	CorePages  []string
	DataModels []string
	APIs       []string
	NextSteps  []string
}

type functionDocumentPayload struct {
	DocType string
	Source  string
	Path    string
	Content string
	Exists  bool
	Version int64
}

type functionCodeStatePayload struct {
	AppID             string
	AppliedVersion    int64
	DraftVersion      int64
	DraftReady        bool
	AppExists         bool
	DraftExists       bool
	AppliedAppVersion string
	DraftAppVersion   string
	AppliedUpdatedAt  int64
	DraftUpdatedAt    int64
	Summary           string
}

func (f *FunctionLogic) buildFunctionCodeState(ctx context.Context, function *model.Function) (functionCodeStatePayload, error) {
	state := functionCodeStatePayload{
		AppliedVersion: function.CodeVersion,
		DraftVersion:   function.CodeVersion,
		DraftReady:     false,
	}
	if function == nil || strings.TrimSpace(function.GeneratedAppID) == "" {
		return state, nil
	}
	state.AppID = function.GeneratedAppID
	if f.appRuntime == nil {
		return state, nil
	}
	inspection, err := f.appRuntime.InspectApp(function.GeneratedAppID)
	if err != nil {
		return functionCodeStatePayload{}, err
	}
	state.DraftExists = inspection.Exists
	state.DraftAppVersion = inspection.ManifestVersion
	state.DraftUpdatedAt = inspection.UpdatedAtMicro
	state.AppExists = inspection.Exists
	if inspection.Exists {
		state.Summary = compactText(firstNonEmpty(inspection.Description, inspection.Name), 160)
		state.AppliedAppVersion = inspection.ManifestVersion
		state.AppliedUpdatedAt = inspection.UpdatedAtMicro
	}
	return state, nil
}

func (f *FunctionLogic) touchCodeDraftVersion(ctx context.Context, function *model.Function) error {
	if function == nil {
		return errors.New("function is required")
	}
	if strings.TrimSpace(function.GeneratedAppID) == "" {
		return errors.New("generated app is not ready")
	}
	if f.appRuntime == nil {
		return errors.New("generated app runtime is not configured")
	}
	inspection, err := f.appRuntime.InspectApp(function.GeneratedAppID)
	if err != nil {
		return err
	}
	if !inspection.Exists {
		return errors.New("generated app draft does not exist")
	}
	nextVersion := function.FunctionVersion + 1
	if function.CodeDraftVersion >= nextVersion {
		return nil
	}
	function.FunctionVersion = nextVersion
	function.CodeVersion = nextVersion
	function.CodeDraftVersion = nextVersion
	function.UpdatedAt = nowUnixMicro()
	return f.functionDao.Update(ctx, function)
}

func functionCodeStateToProto(state functionCodeStatePayload) *funcoperation.FunctionCodeState {
	return &funcoperation.FunctionCodeState{
		AppId:             state.AppID,
		AppliedVersion:    state.AppliedVersion,
		DraftVersion:      state.DraftVersion,
		DraftReady:        state.DraftReady,
		AppExists:         state.AppExists,
		DraftExists:       state.DraftExists,
		AppliedAppVersion: state.AppliedAppVersion,
		DraftAppVersion:   state.DraftAppVersion,
		AppliedUpdatedAt:  state.AppliedUpdatedAt,
		DraftUpdatedAt:    state.DraftUpdatedAt,
		Summary:           state.Summary,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func NewFunctionLogic(functionDao *dao.Function, conversationOperations *dao.FunctionConversationOperation, messageStore *dao.AiAgentMessageStore, generatedAppDao *dao.GeneratedApp, appRuntime, previewRuntime *generatedapp.Service, aiAgentProvider AiAgentClientProvider, menuSync *FunctionMenuSync, authorizer *FunctionAuthorizer, releaseLogic *ReleaseLogic) *FunctionLogic {
	return &FunctionLogic{
		functionDao:            functionDao,
		conversationOperations: conversationOperations,
		messageStore:           messageStore,
		generatedAppDao:        generatedAppDao,
		appRuntime:             appRuntime,
		previewRuntime:         previewRuntime,
		aiAgentProvider:        aiAgentProvider,
		menuSync:               menuSync,
		authorizer:             authorizer,
		releaseLogic:           releaseLogic,
	}
}

func (f *FunctionLogic) ListFunctions(ctx context.Context, req *funcoperation.ListFunctionsReq) ([]*funcoperation.Function, error) {
	functions, err := f.functionDao.List(ctx)
	if err != nil {
		return nil, err
	}
	isEditor, authErr := f.authorizer.Allowed(ctx, functionPermissionEdit)
	if authErr != nil {
		return nil, authErr
	}

	out := make([]*funcoperation.Function, 0, len(functions))
	for i := range functions {
		if !isEditor && functions[i].Status != model.FunctionStatusPublished {
			continue
		}
		f.syncDocumentStateFromFiles(&functions[i])
		normalizeLegacyFunctionVersions(&functions[i])
		item := f.modelFunctionToProto(&functions[i])
		if !isEditor {
			redactFunctionEditorState(item)
		}
		out = append(out, item)
	}
	return out, nil
}

func redactFunctionEditorState(function *funcoperation.Function) {
	if function == nil {
		return
	}
	function.ProductDoc = ""
	function.TechnicalDoc = ""
	function.ProductDocPath = ""
	function.TechnicalDocPath = ""
	function.ProductDraftDocPath = ""
	function.TechnicalDraftDocPath = ""
	function.ProductSessionId = ""
	function.TechnicalSessionId = ""
	function.GenerationSessionId = ""
	function.AppDir = ""
}

func (f *FunctionLogic) CreateFunction(ctx context.Context, req *funcoperation.CreateFunctionReq) (*funcoperation.Function, error) {
	if err := f.authorizer.Require(ctx, functionPermissionCreate); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, errors.New("name is required")
	}

	now := nowUnixMicro()
	functionUUID := uuid.NewString()
	function := &model.Function{
		UUID:           functionUUID,
		Name:           name,
		Description:    strings.TrimSpace(req.GetDescription()),
		Status:         model.FunctionStatusDraft,
		WorkflowStage:  model.FunctionWorkflowStageProductDoc,
		GeneratedAppID: functionUUID,
		MenuParentID:   strings.TrimSpace(req.GetMenuParentId()),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := f.functionDao.Create(ctx, function); err != nil {
		return nil, err
	}
	if function.ID == 0 {
		return nil, errors.New("function id was not allocated")
	}
	if err := f.ensureFunctionWorkspace(function); err != nil {
		_ = f.functionDao.Delete(ctx, function.UUID)
		_ = os.RemoveAll(f.functionAppDir(function))
		return nil, err
	}
	f.syncDocumentStateFromFiles(function)
	normalizeLegacyFunctionVersions(function)
	return f.modelFunctionToProto(function), nil
}

func (f *FunctionLogic) DeleteFunction(ctx context.Context, req *funcoperation.DeleteFunctionReq) error {
	if err := f.authorizer.Require(ctx, functionPermissionDelete); err != nil {
		return err
	}
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return errors.New("function id is required")
	}
	function, err := f.functionDao.Get(ctx, id)
	if err != nil {
		return err
	}
	if f.releaseLogic != nil {
		if err := f.releaseLogic.RevokeFunctionReleases(ctx, function.UUID); err != nil {
			return err
		}
	}
	if err := f.functionDao.Delete(ctx, id); err != nil {
		return err
	}
	f.deleteFunctionSessions(ctx, function)
	_ = os.RemoveAll(f.functionAppDir(function))
	if err := f.deleteFunctionMenu(ctx, function); err != nil {
		if f.releaseLogic != nil {
			f.releaseLogic.RecordEvent(ctx, function.UUID, function.ActiveReleaseID, "menu.delete", map[string]string{"functionId": function.UUID})
		}
		return err
	}
	return nil
}

func (f *FunctionLogic) deleteFunctionMenu(ctx context.Context, function *model.Function) error {
	if f == nil || f.menuSync == nil || function == nil {
		return nil
	}
	return f.menuSync.Delete(ctx, function.UUID)
}

func (f *FunctionLogic) UpdateFunction(ctx context.Context, req *funcoperation.UpdateFunctionReq) (*funcoperation.Function, error) {
	if err := f.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return nil, err
	}
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, errors.New("function id is required")
	}

	function, err := f.functionDao.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	f.syncDocumentStateFromFiles(function)

	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, errors.New("name is required")
	}

	status := normalizeFunctionStatus(strings.TrimSpace(req.GetStatus()), function.Status)
	workflowStage := normalizeWorkflowStage(strings.TrimSpace(req.GetWorkflowStage()), function.WorkflowStage)
	menuParentID := strings.TrimSpace(req.GetMenuParentId())
	if menuParentID == "" {
		menuParentID = strings.TrimSpace(function.MenuParentID)
	}
	productDoc := strings.TrimSpace(function.ProductDoc)
	technicalDoc := strings.TrimSpace(function.TechnicalDoc)
	if status == model.FunctionStatusPublished && !functionHasGeneratedCode(function) {
		return nil, errors.New("generate function page before publishing")
	}
	if status == model.FunctionStatusPublished && menuParentID == "" {
		return nil, errors.New("menu parent directory is required before publishing")
	}
	if status == model.FunctionStatusPublished {
		syncCodeVersionsOnPublish(function)
	}
	if workflowStage == model.FunctionWorkflowStageTechnicalDoc && productDoc == "" {
		return nil, errors.New("product doc is required before technical design")
	}
	if (workflowStage == model.FunctionWorkflowStageCodeGeneration || workflowStage == model.FunctionWorkflowStageCodeGenerated) && technicalDoc == "" {
		return nil, errors.New("technical doc is required before code generation")
	}

	previousStatus := function.Status
	if previousStatus != status {
		permission := functionPermissionPublish
		if status != model.FunctionStatusPublished {
			permission = functionPermissionUnpublish
		}
		if err := f.authorizer.Require(ctx, permission); err != nil {
			return nil, err
		}
	}
	function.Name = name
	function.Description = strings.TrimSpace(req.GetDescription())
	function.Status = status
	function.WorkflowStage = workflowStage
	function.MenuParentID = menuParentID
	function.UpdatedAt = nowUnixMicro()
	if status == model.FunctionStatusPublished && previousStatus != model.FunctionStatusPublished {
		if f.releaseLogic == nil {
			return nil, errors.New("release runtime is not configured")
		}
		release, releaseErr := f.releaseLogic.Publish(ctx, function)
		if releaseErr != nil {
			return nil, releaseErr
		}
		function.ActiveReleaseID = release.UUID
	}

	if err := f.functionDao.Update(ctx, function); err != nil {
		return nil, err
	}
	if err := f.syncFunctionMenu(ctx, function, previousStatus, status); err != nil {
		if f.releaseLogic != nil {
			f.releaseLogic.RecordEvent(ctx, function.UUID, function.ActiveReleaseID, "menu.sync", map[string]string{"status": status})
		}
		return nil, err
	}
	if err := f.functionDao.Update(ctx, function); err != nil {
		return nil, err
	}
	if previousStatus != model.FunctionStatusPublished && status == model.FunctionStatusPublished && f.releaseLogic != nil {
		f.releaseLogic.DeliverFunctionPublished(ctx, function.ActiveReleaseID)
	}
	if previousStatus == model.FunctionStatusPublished && status != model.FunctionStatusPublished {
		if f.releaseLogic == nil {
			return nil, errors.New("release runtime is not configured")
		}
		if err := f.releaseLogic.RecordFunctionUnpublished(ctx, function); err != nil {
			return nil, fmt.Errorf("record function unpublish event: %w", err)
		}
	}
	f.syncDocumentStateFromFiles(function)
	return f.modelFunctionToProto(function), nil
}

func (f *FunctionLogic) LoadFunctionDocument(ctx context.Context, req *funcoperation.LoadFunctionDocumentReq) (*funcoperation.LoadFunctionDocumentResp, error) {
	if err := f.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return nil, err
	}
	function, err := f.functionDao.Get(ctx, strings.TrimSpace(req.GetId()))
	if err != nil {
		return nil, err
	}
	original := *function
	f.syncDocumentStateFromFiles(function)
	if normalizeFunctionDocumentSource(req.GetSource()) == functionDocumentSourceDraft {
		if err := f.persistFunctionDocumentVersionsIfChanged(ctx, function, &original); err != nil {
			return nil, err
		}
	}

	document, err := f.loadFunctionDocument(function, req.GetDocType(), req.GetSource())
	if err != nil {
		return nil, err
	}
	document.Version = f.documentVersion(function, document.DocType, document.Source)
	return &funcoperation.LoadFunctionDocumentResp{
		Document: functionDocumentToProto(document),
	}, nil
}

func (f *FunctionLogic) CommitFunctionDocument(ctx context.Context, req *funcoperation.CommitFunctionDocumentReq) (*funcoperation.CommitFunctionDocumentResp, error) {
	if err := f.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return nil, err
	}
	function, err := f.functionDao.Get(ctx, strings.TrimSpace(req.GetId()))
	if err != nil {
		return nil, err
	}
	f.syncDocumentStateFromFiles(function)

	if err := f.validateDocumentCommit(function, req.GetDocType()); err != nil {
		return nil, err
	}
	technicalSourceProductVersion := int64(0)
	if normalizeFunctionDocumentType(req.GetDocType()) == functionDocumentTypeTechnical {
		technicalSourceProductVersion, err = f.technicalDraftProductVersion(ctx, function)
		if err != nil {
			return nil, err
		}
		if technicalSourceProductVersion != function.ProductDocVersion {
			return nil, errors.New("product doc changed after this technical draft was generated; regenerate the technical doc")
		}
	}
	document, err := f.commitFunctionDocument(function, req.GetDocType())
	if err != nil {
		return nil, err
	}
	switch normalizeFunctionDocumentType(req.GetDocType()) {
	case functionDocumentTypeProduct:
		nextVersion := adoptDocumentVersion(function.ProductDocVersion, function.ProductDraftVersion)
		function.ProductDocVersion = nextVersion
		function.ProductDraftVersion = nextVersion
	case functionDocumentTypeTechnical:
		nextVersion := adoptDocumentVersion(function.TechnicalDocVersion, function.TechnicalDraftVersion)
		function.TechnicalDocVersion = nextVersion
		function.TechnicalDraftVersion = nextVersion
		function.TechnicalDraftOperationID = ""
		function.TechnicalSourceProductVersion = technicalSourceProductVersion
	}
	function.UpdatedAt = nowUnixMicro()
	switch normalizeFunctionDocumentType(req.GetDocType()) {
	case functionDocumentTypeProduct:
		function.WorkflowStage = model.FunctionWorkflowStageTechnicalDoc
	case functionDocumentTypeTechnical:
		function.WorkflowStage = model.FunctionWorkflowStageCodeGeneration
	}
	if err := f.functionDao.Update(ctx, function); err != nil {
		return nil, err
	}
	f.syncDocumentStateFromFiles(function)
	document.Version = f.documentVersion(function, document.DocType, functionDocumentSourceApplied)
	return &funcoperation.CommitFunctionDocumentResp{
		Function: f.modelFunctionToProto(function),
		Document: functionDocumentToProto(document),
	}, nil
}

func (f *FunctionLogic) MaterializeFunctionApp(ctx context.Context, req *funcoperation.MaterializeFunctionAppReq) (*funcoperation.MaterializeFunctionAppResp, error) {
	if err := f.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return nil, err
	}
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, errors.New("function id is required")
	}
	if f.previewRuntime == nil {
		return nil, errors.New("generated app debug runtime is not configured")
	}

	function, err := f.functionDao.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	f.syncDocumentStateFromFiles(function)
	if strings.TrimSpace(function.ProductDoc) == "" {
		return nil, errors.New("product doc is required before code generation")
	}
	if strings.TrimSpace(function.TechnicalDoc) == "" {
		return nil, errors.New("technical doc is required before code generation")
	}

	insights := f.buildFunctionInsights(ctx, function)

	app, err := f.resolveFunctionApp(ctx, function)
	if err != nil {
		appID := strings.TrimSpace(function.GeneratedAppID)
		if appID == "" {
			appID = uuid.NewString()
		}
		app, err = f.appRuntime.CreateAppScaffold(ctx, generatedapp.ScaffoldInput{
			AppID:       appID,
			Name:        function.Name,
			Description: function.Description,
			SessionID:   function.GenerationSessionID,
			TablePrefix: functionTablePrefix(function),
			Summary:     insights.Summary,
			Highlights:  insights.Highlights,
			CorePages:   insights.CorePages,
			DataModels:  insights.DataModels,
			APIs:        insights.APIs,
			NextSteps:   insights.NextSteps,
		})
		if err != nil {
			return nil, err
		}
	}

	function.GeneratedAppID = app.UUID
	if f.releaseLogic != nil {
		if _, err := f.releaseLogic.Stage(ctx, function); err != nil {
			return nil, err
		}
	}
	function.Entry = app.FrontendEntry
	function.WorkflowStage = model.FunctionWorkflowStageCodeGeneration
	function.FunctionVersion++
	function.CodeDraftVersion = function.FunctionVersion
	function.Status = defaultString(function.Status, model.FunctionStatusDraft, model.FunctionStatusDraft)
	function.UpdatedAt = nowUnixMicro()
	if err := f.functionDao.Update(ctx, function); err != nil {
		return nil, err
	}
	f.syncDocumentStateFromFiles(function)

	return &funcoperation.MaterializeFunctionAppResp{
		Function: f.modelFunctionToProto(function),
		App:      functionGeneratedAppToProto(app, function),
	}, nil
}

func (f *FunctionLogic) ListFunctionDataForms(ctx context.Context, req *funcoperation.ListFunctionDataFormsReq) (*funcoperation.ListFunctionDataFormsResp, error) {
	if err := f.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return nil, err
	}
	function, err := f.functionDao.Get(ctx, strings.TrimSpace(req.GetId()))
	if err != nil {
		return nil, err
	}
	forms, err := f.listFunctionDataForms(ctx, function)
	if err != nil {
		return nil, err
	}
	return &funcoperation.ListFunctionDataFormsResp{Forms: forms}, nil
}

func (f *FunctionLogic) DeleteFunctionDataForm(ctx context.Context, req *funcoperation.DeleteFunctionDataFormReq) (*funcoperation.DeleteFunctionDataFormResp, error) {
	if err := f.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return nil, err
	}
	function, err := f.functionDao.Get(ctx, strings.TrimSpace(req.GetId()))
	if err != nil {
		return nil, err
	}
	if f.previewRuntime == nil {
		return nil, errors.New("generated app debug runtime is not configured")
	}
	appID := strings.TrimSpace(function.GeneratedAppID)
	if appID == "" {
		return nil, errors.New("generated app is not ready")
	}
	forms, err := f.previewRuntime.DeleteDataForm(ctx, appID, req.GetName())
	if err != nil {
		return nil, err
	}
	return &funcoperation.DeleteFunctionDataFormResp{Forms: dataFormSummariesToProto(forms)}, nil
}

func (f *FunctionLogic) listFunctionDataForms(ctx context.Context, function *model.Function) ([]*funcoperation.FunctionDataForm, error) {
	if f.previewRuntime == nil {
		return nil, errors.New("generated app debug runtime is not configured")
	}
	appID := strings.TrimSpace(function.GeneratedAppID)
	if appID == "" {
		return []*funcoperation.FunctionDataForm{}, nil
	}
	forms, err := f.previewRuntime.ListDataForms(ctx, appID)
	if err != nil {
		return nil, err
	}
	return dataFormSummariesToProto(forms), nil
}

func dataFormSummariesToProto(forms []generatedapp.DataFormSummary) []*funcoperation.FunctionDataForm {
	out := make([]*funcoperation.FunctionDataForm, 0, len(forms))
	for _, form := range forms {
		out = append(out, &funcoperation.FunctionDataForm{
			Name:       form.Name,
			Label:      form.Label,
			FieldCount: form.FieldCount,
			RowCount:   form.RowCount,
			TableName:  form.TableName,
		})
	}
	return out
}

func (f *FunctionLogic) LoadFunctionCodeState(ctx context.Context, req *funcoperation.LoadFunctionCodeStateReq) (*funcoperation.LoadFunctionCodeStateResp, error) {
	if err := f.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return nil, err
	}
	function, err := f.functionDao.Get(ctx, strings.TrimSpace(req.GetId()))
	if err != nil {
		return nil, err
	}
	f.syncDocumentStateFromFiles(function)
	normalizeLegacyFunctionVersions(function)
	state, err := f.buildFunctionCodeState(ctx, function)
	if err != nil {
		return nil, err
	}
	return &funcoperation.LoadFunctionCodeStateResp{
		State: functionCodeStateToProto(state),
	}, nil
}

func (f *FunctionLogic) TouchFunctionCodeDraft(ctx context.Context, req *funcoperation.TouchFunctionCodeDraftReq) (*funcoperation.TouchFunctionCodeDraftResp, error) {
	if err := f.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return nil, err
	}
	function, err := f.functionDao.Get(ctx, strings.TrimSpace(req.GetId()))
	if err != nil {
		return nil, err
	}
	f.syncDocumentStateFromFiles(function)
	normalizeLegacyFunctionVersions(function)
	if err := f.touchCodeDraftVersion(ctx, function); err != nil {
		return nil, err
	}
	state, err := f.buildFunctionCodeState(ctx, function)
	if err != nil {
		return nil, err
	}
	return &funcoperation.TouchFunctionCodeDraftResp{
		Function: f.modelFunctionToProto(function),
		State:    functionCodeStateToProto(state),
	}, nil
}

func (f *FunctionLogic) ApplyFunctionCode(ctx context.Context, req *funcoperation.ApplyFunctionCodeReq) (*funcoperation.ApplyFunctionCodeResp, error) {
	if err := f.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return nil, err
	}
	function, err := f.functionDao.Get(ctx, strings.TrimSpace(req.GetId()))
	if err != nil {
		return nil, err
	}
	f.syncDocumentStateFromFiles(function)
	normalizeLegacyFunctionVersions(function)
	if strings.TrimSpace(function.GeneratedAppID) == "" {
		return nil, errors.New("generate app scaffold before applying code")
	}
	if function.ProductDocVersion == 0 {
		return nil, errors.New("product doc must be applied before code")
	}
	if function.TechnicalDocVersion == 0 {
		return nil, errors.New("technical doc must be applied before code")
	}
	if function.DocTechnicalStale {
		return nil, errors.New("technical doc is stale, please regenerate and apply technical doc")
	}
	codeSourceTechnicalVersion, err := f.codeDraftTechnicalVersion(ctx, function)
	if err != nil {
		return nil, err
	}
	if codeSourceTechnicalVersion != function.TechnicalDocVersion {
		return nil, errors.New("technical doc changed after this page was generated; regenerate the page")
	}
	if f.appRuntime == nil {
		return nil, errors.New("generated app runtime is not configured")
	}
	if f.releaseLogic == nil {
		return nil, errors.New("release runtime is not configured")
	}
	if _, err := f.releaseLogic.Stage(ctx, function); err != nil {
		return nil, err
	}
	app, err := f.resolveFunctionApp(ctx, function)
	if err != nil {
		return nil, err
	}
	function.Entry = app.FrontendEntry
	function.FunctionVersion++
	function.CodeVersion = function.FunctionVersion
	function.CodeDraftVersion = function.CodeVersion
	function.CodeDraftOperationID = ""
	function.CodeSourceTechnicalVersion = codeSourceTechnicalVersion
	function.WorkflowStage = model.FunctionWorkflowStageCodeGenerated
	function.UpdatedAt = nowUnixMicro()
	if err := f.functionDao.Update(ctx, function); err != nil {
		return nil, err
	}
	f.syncDocumentStateFromFiles(function)
	state, err := f.buildFunctionCodeState(ctx, function)
	if err != nil {
		return nil, err
	}
	return &funcoperation.ApplyFunctionCodeResp{
		Function: f.modelFunctionToProto(function),
		App:      functionGeneratedAppToProto(app, function),
		State:    functionCodeStateToProto(state),
	}, nil
}

func functionGeneratedAppToProto(app *model.GeneratedApp, function *model.Function) *funcoperation.GeneratedApp {
	out := modelGeneratedAppToProto(app)
	if out != nil {
		out.TablePrefix = functionTablePrefix(function)
	}
	return out
}

func (f *FunctionLogic) modelFunctionToProto(function *model.Function) *funcoperation.Function {
	if function == nil {
		return nil
	}
	productDraftReady, technicalDraftReady, technicalStale, codeStale, codeDraftReady := functionVersionFlags(function)
	productDocPath := strings.TrimSpace(function.ProductDocPath)
	if productDocPath == "" {
		productDocPath = f.functionDocumentPath(function, functionDocumentTypeProduct, functionDocumentSourceApplied)
	}
	technicalDocPath := strings.TrimSpace(function.TechnicalDocPath)
	if technicalDocPath == "" {
		technicalDocPath = f.functionDocumentPath(function, functionDocumentTypeTechnical, functionDocumentSourceApplied)
	}
	return &funcoperation.Function{
		Id:                    function.UUID,
		Name:                  function.Name,
		Description:           function.Description,
		Status:                function.Status,
		WorkflowStage:         function.WorkflowStage,
		ProductDoc:            function.ProductDoc,
		TechnicalDoc:          function.TechnicalDoc,
		ProductDocPath:        productDocPath,
		TechnicalDocPath:      technicalDocPath,
		Entry:                 function.Entry,
		ProductSessionId:      function.ProductSessionID,
		TechnicalSessionId:    function.TechnicalSessionID,
		GenerationSessionId:   function.GenerationSessionID,
		GeneratedAppId:        function.GeneratedAppID,
		ActiveReleaseId:       function.ActiveReleaseID,
		CreatedAt:             function.CreatedAt,
		UpdatedAt:             function.UpdatedAt,
		FunctionVersion:       function.FunctionVersion,
		ProductDocVersion:     function.ProductDocVersion,
		ProductDraftVersion:   function.ProductDraftVersion,
		TechnicalDocVersion:   function.TechnicalDocVersion,
		TechnicalDraftVersion: function.TechnicalDraftVersion,
		CodeVersion:           function.CodeVersion,
		CodeDraftVersion:      function.CodeDraftVersion,
		ProductDraftReady:     productDraftReady,
		TechnicalDraftReady:   technicalDraftReady,
		TechnicalStale:        technicalStale,
		CodeStale:             codeStale,
		CodeDraftReady:        codeDraftReady,
		NumericId:             int64(function.ID),
		Readiness:             buildFunctionReadiness(function),
		MenuParentId:          function.MenuParentID,
		MenuId:                function.MenuID,
		AppDir:                f.functionAppDir(function),
		ProductDraftDocPath:   f.functionDocumentPath(function, functionDocumentTypeProduct, functionDocumentSourceDraft),
		TechnicalDraftDocPath: f.functionDocumentPath(function, functionDocumentTypeTechnical, functionDocumentSourceDraft),
	}
}

func (f *FunctionLogic) syncFunctionMenu(ctx context.Context, function *model.Function, previousStatus, nextStatus string) error {
	if f == nil || f.menuSync == nil || function == nil {
		return nil
	}
	actionKeys := f.functionActionKeys(function)
	switch {
	case nextStatus == model.FunctionStatusPublished && previousStatus != model.FunctionStatusPublished:
		menuID, err := f.menuSync.SyncPublished(ctx, function.UUID, function.Name, function.MenuParentID, actionKeys)
		if err != nil {
			return err
		}
		if menuID != "" {
			function.MenuID = menuID
		}
	case previousStatus == model.FunctionStatusPublished && nextStatus != model.FunctionStatusPublished:
		if err := f.menuSync.Unpublish(ctx, function.UUID); err != nil {
			return err
		}
	case previousStatus == model.FunctionStatusPublished && nextStatus == model.FunctionStatusPublished:
		menuID, err := f.menuSync.SyncMetadata(ctx, function.UUID, function.Name, function.MenuParentID, actionKeys)
		if err != nil {
			return err
		}
		if menuID != "" {
			function.MenuID = menuID
		}
	}
	return nil
}

func (f *FunctionLogic) functionActionKeys(function *model.Function) []string {
	if f == nil || f.appRuntime == nil || function == nil {
		return nil
	}
	appID := strings.TrimSpace(function.GeneratedAppID)
	if appID == "" {
		return nil
	}
	inspection, err := f.appRuntime.InspectApp(appID)
	if err != nil || !inspection.Exists {
		return nil
	}
	return inspection.Actions
}

func (f *FunctionLogic) EnsureFunctionSession(ctx context.Context, req *funcoperation.EnsureFunctionSessionReq) (*funcoperation.EnsureFunctionSessionResp, error) {
	if err := f.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return nil, err
	}
	function, err := f.functionDao.Get(ctx, strings.TrimSpace(req.GetId()))
	if err != nil {
		return nil, err
	}
	if function.ID == 0 {
		return nil, errors.New("function id is not ready")
	}

	conversation := normalizeFunctionConversation(req.GetConversation())
	sessionID := functionSessionKey(function.ID, conversation)
	title := functionSessionTitle(function.Name, conversation)
	createdSession, created, err := f.ensureAiAgentSession(ctx, sessionID, title)
	if err != nil {
		return nil, err
	}

	switch conversation {
	case functionConversationProduct:
		function.ProductSessionID = createdSession.GetId()
	case functionConversationTechnical:
		function.TechnicalSessionID = createdSession.GetId()
	case functionConversationGeneration:
		function.GenerationSessionID = createdSession.GetId()
	default:
		return nil, errors.New("unsupported conversation")
	}
	function.UpdatedAt = nowUnixMicro()
	if err := f.functionDao.Update(ctx, function); err != nil {
		return nil, err
	}
	f.syncDocumentStateFromFiles(function)
	normalizeLegacyFunctionVersions(function)
	return &funcoperation.EnsureFunctionSessionResp{
		Function:  f.modelFunctionToProto(function),
		SessionId: createdSession.GetId(),
		Created:   created,
	}, nil
}

func (f *FunctionLogic) ensureAiAgentSession(ctx context.Context, sessionID string, title string) (*aiagent.Session, bool, error) {
	if f.aiAgentProvider == nil {
		return nil, false, errors.New("ai-agent client is not configured")
	}
	aiAgentClient, err := f.aiAgentProvider(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("get ai-agent client: %w", err)
	}
	resp, err := aiAgentClient.EnsureSession(authctx.ForwardUserContext(ctx), &aiagent.EnsureSessionReq{
		Id:          sessionID,
		Title:       title,
		SessionType: aiAgentSessionTypeFunction,
	})
	if err != nil {
		return nil, false, err
	}
	if resp.GetSession() == nil || strings.TrimSpace(resp.GetSession().GetId()) == "" {
		return nil, false, errors.New("ai-agent session id is empty")
	}
	return resp.GetSession(), resp.GetCreated(), nil
}

func (f *FunctionLogic) deleteFunctionSessions(ctx context.Context, function *model.Function) {
	if function == nil || function.ID == 0 {
		return
	}
	for _, conversation := range []string{
		functionConversationProduct,
		functionConversationTechnical,
		functionConversationGeneration,
	} {
		f.deleteAiAgentSession(ctx, functionSessionKey(function.ID, conversation))
	}
	if strings.TrimSpace(function.ProductSessionID) != "" {
		f.deleteAiAgentSession(ctx, function.ProductSessionID)
	}
	if strings.TrimSpace(function.TechnicalSessionID) != "" {
		f.deleteAiAgentSession(ctx, function.TechnicalSessionID)
	}
	if strings.TrimSpace(function.GenerationSessionID) != "" {
		f.deleteAiAgentSession(ctx, function.GenerationSessionID)
	}
}

func (f *FunctionLogic) deleteAiAgentSession(ctx context.Context, sessionID string) {
	if f.aiAgentProvider == nil || strings.TrimSpace(sessionID) == "" {
		return
	}
	aiAgentClient, err := f.aiAgentProvider(ctx)
	if err != nil {
		return
	}
	_, _ = aiAgentClient.DeleteSession(authctx.ForwardUserContext(ctx), &aiagent.DeleteSessionReq{Id: sessionID})
}

func defaultString(value string, fallback string, zero string) string {
	if value != "" {
		return value
	}
	if fallback != "" {
		return fallback
	}
	return zero
}

func normalizeFunctionStatus(value string, fallback string) string {
	switch value {
	case "", model.FunctionStatusDraft, model.FunctionStatusPublished:
		return defaultString(value, fallback, model.FunctionStatusDraft)
	default:
		return fallback
	}
}

func normalizeWorkflowStage(value string, fallback string) string {
	switch value {
	case "",
		model.FunctionWorkflowStageProductDoc,
		model.FunctionWorkflowStageTechnicalDoc,
		model.FunctionWorkflowStageCodeGeneration,
		model.FunctionWorkflowStageCodeGenerated:
		return defaultString(value, fallback, model.FunctionWorkflowStageProductDoc)
	default:
		return fallback
	}
}

func (f *FunctionLogic) resolveFunctionApp(ctx context.Context, function *model.Function) (*model.GeneratedApp, error) {
	if f.appRuntime == nil {
		return nil, errors.New("generated app runtime is not configured")
	}

	currentAppID := strings.TrimSpace(function.GeneratedAppID)
	if currentAppID == "" {
		return nil, generatedapp.ErrAppNotFound
	}

	// Reading a generated draft must not load it into the executable runtime.
	if app, err := f.appRuntime.CandidateApp(ctx, currentAppID); err == nil {
		return app, nil
	}
	if f.generatedAppDao != nil {
		if app, err := f.generatedAppDao.Get(ctx, currentAppID); err == nil {
			return app, nil
		}
	}
	return nil, generatedapp.ErrAppNotFound
}

func functionTablePrefix(function *model.Function) string {
	if function == nil {
		return "func_unknown"
	}
	if function.ID > 0 {
		return "func_" + uint64ToString(function.ID)
	}
	return "func_" + snakeIdentifier(function.UUID)
}

func snakeIdentifier(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastUnderscore := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(builder.String(), "_")
	if out == "" {
		return "unknown"
	}
	return out
}

func uint64ToString(value uint64) string {
	return strconv.FormatUint(value, 10)
}

func (f *FunctionLogic) buildFunctionInsights(ctx context.Context, function *model.Function) conversationInsights {
	insights := buildInsightsFromDocs(function)
	if insights.Summary != "" && len(insights.CorePages) > 0 {
		return insights
	}

	sessionID := ""
	if function != nil {
		sessionID = function.GenerationSessionID
	}
	if f.messageStore == nil || strings.TrimSpace(sessionID) == "" {
		return insights
	}
	messages, err := f.messageStore.ListRecentBySession(ctx, sessionID, 6)
	if err != nil || len(messages) == 0 {
		return insights
	}

	if insights.Summary == "" {
		insights.Summary = buildSummaryFromMessages(messages)
	}
	if len(insights.Highlights) == 0 {
		insights.Highlights = buildHighlightsFromMessages(messages)
	}
	planSource := latestAssistantContent(messages)
	if strings.TrimSpace(planSource) == "" {
		planSource = strings.TrimSpace(messages[0].Content)
	}
	if len(insights.CorePages) == 0 {
		insights.CorePages = extractSectionItems(planSource, []string{"核心页面", "页面", "页面设计"})
	}
	if len(insights.DataModels) == 0 {
		insights.DataModels = extractSectionItems(planSource, []string{"数据模型", "模型设计", "数据结构"})
	}
	if len(insights.APIs) == 0 {
		insights.APIs = extractSectionItems(planSource, []string{"API 设计", "接口设计", "接口"})
	}
	if len(insights.NextSteps) == 0 {
		insights.NextSteps = extractSectionItems(planSource, []string{"下一步", "下一步实现计划", "实现计划"})
	}
	if len(insights.NextSteps) == 0 {
		insights.NextSteps = insights.Highlights
	}
	return insights
}

func (f *FunctionLogic) syncDocumentStateFromFiles(function *model.Function) {
	if function == nil {
		return
	}
	if err := f.ensureFunctionWorkspace(function); err != nil {
		return
	}

	productApplied, _ := f.loadFunctionDocument(function, functionDocumentTypeProduct, functionDocumentSourceApplied)
	productDraft, _ := f.loadFunctionDocument(function, functionDocumentTypeProduct, functionDocumentSourceDraft)
	technicalApplied, _ := f.loadFunctionDocument(function, functionDocumentTypeTechnical, functionDocumentSourceApplied)
	technicalDraft, _ := f.loadFunctionDocument(function, functionDocumentTypeTechnical, functionDocumentSourceDraft)

	function.ProductDoc = productApplied.Content
	function.TechnicalDoc = technicalApplied.Content
	function.ProductDocPath = productApplied.Path
	function.TechnicalDocPath = technicalApplied.Path

	storedProductDraftVersion := function.ProductDraftVersion
	storedTechnicalDraftVersion := function.TechnicalDraftVersion

	function.ProductDocVersion = appliedDocumentVersion(productApplied, function.ProductDocVersion)
	function.ProductDraftVersion = draftDocumentVersion(productApplied, productDraft, function.ProductDocVersion, storedProductDraftVersion)
	function.TechnicalDocVersion = appliedDocumentVersion(technicalApplied, function.TechnicalDocVersion)
	function.TechnicalDraftVersion = draftDocumentVersion(technicalApplied, technicalDraft, function.TechnicalDocVersion, storedTechnicalDraftVersion)

	function.DocTechnicalStale = false
	if productApplied.Exists && technicalApplied.Exists {
		if function.TechnicalSourceProductVersion > 0 {
			function.DocTechnicalStale = function.TechnicalSourceProductVersion != function.ProductDocVersion
		} else {
			// Existing functions predate source-version tracking. Preserve their
			// file-time behavior until their next technical document is adopted.
			function.DocTechnicalStale = fileModTime(productApplied.Path) > fileModTime(technicalApplied.Path)
		}
	}
	if function.DocTechnicalStale {
		function.TechnicalDraftVersion = function.TechnicalDocVersion
	}
}

func appliedDocumentVersion(document functionDocumentPayload, storedVersion int64) int64 {
	if !document.Exists || strings.TrimSpace(document.Content) == "" {
		return 0
	}
	if storedVersion > 0 {
		return storedVersion
	}
	return 1
}

func draftDocumentVersion(applied, draft functionDocumentPayload, appliedVersion int64, storedDraftVersion int64) int64 {
	if !draft.Exists || strings.TrimSpace(draft.Content) == "" {
		return appliedVersion
	}
	next := appliedVersion
	if !applied.Exists || strings.TrimSpace(applied.Content) == "" {
		if appliedVersion == 0 {
			next = 1
		} else {
			next = appliedVersion + 1
		}
	} else if strings.TrimSpace(draft.Content) != strings.TrimSpace(applied.Content) {
		if appliedVersion == 0 {
			next = 1
		} else {
			next = appliedVersion + 1
		}
	}
	if storedDraftVersion > next {
		return storedDraftVersion
	}
	return next
}

func adoptDocumentVersion(appliedVersion int64, draftVersion int64) int64 {
	next := draftVersion
	if next <= appliedVersion {
		next = appliedVersion + 1
	}
	if next < 1 {
		next = 1
	}
	return next
}

func (f *FunctionLogic) persistFunctionDocumentVersionsIfChanged(ctx context.Context, function *model.Function, original *model.Function) error {
	if function == nil || original == nil {
		return nil
	}
	changed := function.ProductDraftVersion != original.ProductDraftVersion ||
		function.TechnicalDraftVersion != original.TechnicalDraftVersion ||
		function.ProductDocVersion != original.ProductDocVersion ||
		function.TechnicalDocVersion != original.TechnicalDocVersion ||
		function.DocTechnicalStale != original.DocTechnicalStale
	if !changed {
		return nil
	}
	function.UpdatedAt = nowUnixMicro()
	return f.functionDao.Update(ctx, function)
}

func fileModTime(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.ModTime().UnixMicro()
}

func (f *FunctionLogic) generatedAppsRoot() string {
	root := ""
	if f.appRuntime != nil {
		root = strings.TrimSpace(f.appRuntime.RootDir())
	}
	if root == "" {
		root = "generated_apps"
	}
	if filepath.IsAbs(root) {
		return filepath.Clean(root)
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return filepath.Clean(root)
	}
	return absolute
}

func (f *FunctionLogic) functionAppID(function *model.Function) string {
	if function == nil {
		return ""
	}
	if appID := strings.TrimSpace(function.GeneratedAppID); appID != "" {
		return appID
	}
	return strings.TrimSpace(function.UUID)
}

func (f *FunctionLogic) functionAppDir(function *model.Function) string {
	appID := f.functionAppID(function)
	if appID == "" {
		return f.generatedAppsRoot()
	}
	return filepath.Join(f.generatedAppsRoot(), appID)
}

func (f *FunctionLogic) ensureFunctionWorkspace(function *model.Function) error {
	if function == nil {
		return errors.New("function is required")
	}
	for _, docType := range []string{functionDocumentTypeProduct, functionDocumentTypeTechnical} {
		for _, source := range []string{functionDocumentSourceDraft, functionDocumentSourceApplied} {
			path := f.functionDocumentPath(function, docType, source)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *FunctionLogic) validateDocumentCommit(function *model.Function, docType string) error {
	docType = normalizeFunctionDocumentType(docType)
	if function == nil {
		return errors.New("function is required")
	}
	if docType == functionDocumentTypeTechnical && function.ProductDocVersion == 0 {
		return errors.New("product doc must be applied before technical doc")
	}
	return nil
}

func (f *FunctionLogic) technicalDraftProductVersion(ctx context.Context, function *model.Function) (int64, error) {
	return f.completedOperationBaselineVersion(
		ctx,
		function,
		function.TechnicalDraftOperationID,
		functionConversationTechnical,
		"productDocVersion",
		function.ProductDocVersion,
	)
}

func (f *FunctionLogic) codeDraftTechnicalVersion(ctx context.Context, function *model.Function) (int64, error) {
	return f.completedOperationBaselineVersion(
		ctx,
		function,
		function.CodeDraftOperationID,
		functionConversationGeneration,
		"technicalDocVersion",
		function.TechnicalDocVersion,
	)
}

func (f *FunctionLogic) completedOperationBaselineVersion(
	ctx context.Context,
	function *model.Function,
	operationID string,
	conversation string,
	versionKey string,
	legacyVersion int64,
) (int64, error) {
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return legacyVersion, nil
	}
	if f.conversationOperations == nil {
		return 0, errors.New("conversation operation store is not configured")
	}
	operation, err := f.conversationOperations.Get(ctx, operationID)
	if err != nil {
		return 0, err
	}
	if operation.FunctionID != function.UUID || operation.Conversation != conversation || operation.State != model.ConversationOperationStateCompleted {
		return 0, errors.New("generation operation is no longer eligible for adoption")
	}
	var baseline map[string]int64
	if err := json.Unmarshal([]byte(operation.BaselineJSON), &baseline); err != nil {
		return 0, errors.New("generation operation baseline is invalid")
	}
	version := baseline[versionKey]
	if version <= 0 {
		return 0, errors.New("generation operation baseline is missing the source document version")
	}
	return version, nil
}

func (f *FunctionLogic) loadFunctionDocument(function *model.Function, docType string, source string) (functionDocumentPayload, error) {
	docType = normalizeFunctionDocumentType(docType)
	if docType == "" {
		return functionDocumentPayload{}, errors.New("doc type is required")
	}
	source = normalizeFunctionDocumentSource(source)
	path := f.functionDocumentPath(function, docType, source)
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return functionDocumentPayload{
				DocType: docType,
				Source:  source,
				Path:    path,
				Exists:  false,
			}, nil
		}
		return functionDocumentPayload{}, err
	}
	return functionDocumentPayload{
		DocType: docType,
		Source:  source,
		Path:    path,
		Content: strings.TrimSpace(string(content)),
		Exists:  true,
	}, nil
}

func (f *FunctionLogic) commitFunctionDocument(function *model.Function, docType string) (functionDocumentPayload, error) {
	docType = normalizeFunctionDocumentType(docType)
	if docType == "" {
		return functionDocumentPayload{}, errors.New("doc type is required")
	}
	draftPath := f.functionDocumentPath(function, docType, functionDocumentSourceDraft)
	appliedPath := f.functionDocumentPath(function, docType, functionDocumentSourceApplied)
	content, err := os.ReadFile(draftPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return functionDocumentPayload{}, errors.New("draft document does not exist")
		}
		return functionDocumentPayload{}, err
	}
	if err := os.MkdirAll(filepath.Dir(appliedPath), 0o755); err != nil {
		return functionDocumentPayload{}, err
	}
	if err := os.WriteFile(appliedPath, content, 0o644); err != nil {
		return functionDocumentPayload{}, err
	}
	return functionDocumentPayload{
		DocType: docType,
		Source:  functionDocumentSourceApplied,
		Path:    appliedPath,
		Content: strings.TrimSpace(string(content)),
		Exists:  true,
	}, nil
}

func normalizeFunctionDocumentSource(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", functionDocumentSourceApplied, functionDocumentSourcePublished:
		return functionDocumentSourceApplied
	case functionDocumentSourceDraft:
		return functionDocumentSourceDraft
	default:
		return functionDocumentSourceApplied
	}
}

func (f *FunctionLogic) functionDocumentPath(function *model.Function, docType string, source string) string {
	filename := "document.md"
	switch docType {
	case functionDocumentTypeProduct:
		filename = "product-doc.md"
	case functionDocumentTypeTechnical:
		filename = "technical-doc.md"
	}
	return filepath.Join(f.functionAppDir(function), "documents", docType, normalizeFunctionDocumentSource(source), filename)
}

func (f *FunctionLogic) documentVersion(function *model.Function, docType string, source string) int64 {
	if function == nil {
		return 0
	}
	docType = normalizeFunctionDocumentType(docType)
	source = normalizeFunctionDocumentSource(source)
	switch docType {
	case functionDocumentTypeProduct:
		if source == functionDocumentSourceDraft {
			return function.ProductDraftVersion
		}
		return function.ProductDocVersion
	case functionDocumentTypeTechnical:
		if source == functionDocumentSourceDraft {
			return function.TechnicalDraftVersion
		}
		return function.TechnicalDocVersion
	default:
		return 0
	}
}

func syncCodeVersionsOnPublish(function *model.Function) {
	if function == nil {
		return
	}
	if function.CodeDraftVersion > function.CodeVersion {
		function.CodeVersion = function.CodeDraftVersion
	}
	if function.CodeVersion == 0 {
		if function.FunctionVersion > 0 {
			function.CodeVersion = function.FunctionVersion
		} else {
			function.CodeVersion = 1
		}
	}
	if function.CodeDraftVersion < function.CodeVersion {
		function.CodeDraftVersion = function.CodeVersion
	}
}

func functionVersionFlags(function *model.Function) (productDraftReady bool, technicalDraftReady bool, technicalStale bool, codeStale bool, codeDraftReady bool) {
	if function == nil {
		return false, false, false, false, false
	}
	productDraftReady = function.ProductDraftVersion > function.ProductDocVersion
	technicalDraftReady = function.TechnicalDraftVersion > function.TechnicalDocVersion
	codeDraftReady = false
	technicalStale = function.DocTechnicalStale
	if !technicalStale &&
		function.TechnicalDocVersion > 0 &&
		function.ProductDocVersion > 0 &&
		function.TechnicalSourceProductVersion > 0 &&
		function.TechnicalSourceProductVersion != function.ProductDocVersion {
		technicalStale = true
	}
	if function.CodeVersion > 0 {
		if function.CodeSourceTechnicalVersion > 0 {
			codeStale = function.CodeSourceTechnicalVersion != function.TechnicalDocVersion
		} else {
			// Existing generated pages have no recorded technical source version.
			codeStale = function.TechnicalDocVersion > function.CodeVersion
		}
	}
	return productDraftReady, technicalDraftReady, technicalStale, codeStale, codeDraftReady
}

func normalizeLegacyFunctionVersions(function *model.Function) {
	if function == nil {
		return
	}
	if function.ProductDocVersion == 0 && strings.TrimSpace(function.ProductDoc) != "" {
		function.ProductDocVersion = 1
		if function.FunctionVersion < 1 {
			function.FunctionVersion = 1
		}
	}
	if function.TechnicalDocVersion == 0 && strings.TrimSpace(function.TechnicalDoc) != "" {
		next := function.ProductDocVersion + 1
		if next <= function.ProductDocVersion {
			next = function.ProductDocVersion + 1
		}
		if next < 2 {
			next = 2
		}
		function.TechnicalDocVersion = next
		if function.FunctionVersion < next {
			function.FunctionVersion = next
		}
	}
	if function.CodeVersion == 0 && strings.TrimSpace(function.GeneratedAppID) != "" {
		if function.WorkflowStage == model.FunctionWorkflowStageCodeGenerated {
			next := function.TechnicalDocVersion + 1
			if next <= function.TechnicalDocVersion {
				next = function.TechnicalDocVersion + 1
			}
			if next < 1 {
				next = 1
			}
			function.CodeVersion = next
			if function.FunctionVersion < next {
				function.FunctionVersion = next
			}
		}
	}
	if function.CodeDraftVersion == 0 && function.CodeVersion > 0 {
		function.CodeDraftVersion = function.CodeVersion
	}
}

func functionDocumentToProto(document functionDocumentPayload) *funcoperation.FunctionDocument {
	return &funcoperation.FunctionDocument{
		DocType: document.DocType,
		Source:  document.Source,
		Path:    document.Path,
		Content: document.Content,
		Exists:  document.Exists,
		Version: document.Version,
	}
}

func normalizeFunctionDocumentType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case functionDocumentTypeProduct:
		return functionDocumentTypeProduct
	case functionDocumentTypeTechnical:
		return functionDocumentTypeTechnical
	default:
		return ""
	}
}

func functionSessionKey(functionID uint64, conversation string) string {
	return strconv.FormatUint(functionID, 10) + ":" + conversation
}

func functionSessionTitle(name string, conversation string) string {
	switch conversation {
	case functionConversationProduct:
		return strings.TrimSpace(name) + " 产品文档"
	case functionConversationTechnical:
		return strings.TrimSpace(name) + " 研发文档"
	case functionConversationGeneration:
		return strings.TrimSpace(name) + " 代码生成"
	default:
		return strings.TrimSpace(name)
	}
}

func normalizeFunctionConversation(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case functionConversationProduct:
		return functionConversationProduct
	case functionConversationTechnical:
		return functionConversationTechnical
	case functionConversationGeneration, "code", "code_generation":
		return functionConversationGeneration
	default:
		return ""
	}
}

func buildInsightsFromDocs(function *model.Function) conversationInsights {
	if function == nil {
		return conversationInsights{}
	}

	productDoc := strings.TrimSpace(function.ProductDoc)
	technicalDoc := strings.TrimSpace(function.TechnicalDoc)
	if productDoc == "" && technicalDoc == "" {
		return conversationInsights{}
	}

	summary := productDoc
	if summary == "" {
		summary = technicalDoc
	}

	highlights := extractSectionItems(productDoc, []string{"核心目标", "业务目标", "用户价值", "范围", "核心流程"})
	if len(highlights) == 0 {
		highlights = splitHighlights(productDoc)
	}
	corePages := extractSectionItems(productDoc, []string{"核心页面", "页面规划", "页面", "交互流程"})
	dataModels := extractSectionItems(technicalDoc, []string{"数据模型", "数据结构", "模型设计"})
	apis := extractSectionItems(technicalDoc, []string{"API 设计", "接口设计", "接口"})
	nextSteps := extractSectionItems(technicalDoc, []string{"实施步骤", "开发计划", "实现计划", "下一步"})
	if len(nextSteps) == 0 {
		nextSteps = extractSectionItems(productDoc, []string{"下一步", "待确认", "开放问题"})
	}

	return conversationInsights{
		Summary:    compactText(summary, 240),
		Highlights: highlights,
		CorePages:  corePages,
		DataModels: dataModels,
		APIs:       apis,
		NextSteps:  nextSteps,
	}
}

func buildSummaryFromMessages(messages []dao.AiAgentMessage) string {
	if len(messages) == 0 {
		return ""
	}
	for _, item := range messages {
		text := strings.TrimSpace(item.Content)
		if text == "" {
			continue
		}
		if item.Role == "assistant" {
			return compactText(text, 240)
		}
	}
	return compactText(strings.TrimSpace(messages[0].Content), 240)
}

func buildHighlightsFromMessages(messages []dao.AiAgentMessage) []string {
	highlights := make([]string, 0, 4)
	for _, item := range messages {
		lines := splitHighlights(item.Content)
		for _, line := range lines {
			highlights = append(highlights, line)
			if len(highlights) >= 4 {
				return highlights
			}
		}
	}
	return highlights
}

func latestAssistantContent(messages []dao.AiAgentMessage) string {
	for _, item := range messages {
		if item.Role == "assistant" && strings.TrimSpace(item.Content) != "" {
			return item.Content
		}
	}
	return ""
}

func extractSectionItems(content string, headings []string) []string {
	rows := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	collecting := false
	out := make([]string, 0, 4)
	for _, row := range rows {
		trimmed := strings.TrimSpace(row)
		if trimmed == "" {
			if collecting && len(out) > 0 {
				break
			}
			continue
		}

		if matchesHeading(trimmed, headings) {
			collecting = true
			continue
		}
		if collecting && looksLikeHeading(trimmed) && !looksLikeListItem(trimmed) {
			break
		}
		if !collecting {
			continue
		}

		item := strings.TrimSpace(strings.TrimLeft(trimmed, "-*0123456789. "))
		if item == "" {
			continue
		}
		out = append(out, compactText(item, 90))
		if len(out) >= 4 {
			break
		}
	}
	return out
}

func matchesHeading(line string, headings []string) bool {
	normalized := normalizeSectionLine(line)
	for _, heading := range headings {
		candidate := normalizeSectionLine(heading)
		if normalized == candidate || strings.Contains(normalized, candidate) {
			return true
		}
	}
	return false
}

func looksLikeHeading(line string) bool {
	line = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "#-*0123456789. "))
	return strings.Contains(line, "：") || strings.Contains(line, ":")
}

func looksLikeListItem(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "*")
}

func normalizeSectionLine(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimLeft(line, "#-*0123456789. ")
	line = strings.TrimSpace(strings.Trim(line, "：: "))
	return strings.ToLower(line)
}

func splitHighlights(content string) []string {
	rows := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	out := make([]string, 0, 4)
	for _, row := range rows {
		row = strings.TrimSpace(row)
		row = strings.TrimLeft(row, "-*0123456789. ")
		row = strings.TrimSpace(row)
		if row == "" {
			continue
		}
		out = append(out, compactText(row, 80))
		if len(out) >= 4 {
			break
		}
	}
	return out
}

func compactText(value string, limit int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if limit <= 0 || len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}
