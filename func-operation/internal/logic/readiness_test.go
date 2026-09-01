package logic

import (
	"testing"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
)

func TestBuildFunctionReadinessProductStage(t *testing.T) {
	readiness := buildFunctionReadiness(&model.Function{})
	if readiness.Label != "待完善产品方案" || readiness.NextAction != readinessActionGenerateProductDoc {
		t.Fatalf("unexpected readiness: %#v", readiness)
	}

	readiness = buildFunctionReadiness(&model.Function{
		ProductDraftVersion: 1,
	})
	if readiness.NextAction != readinessActionAdoptProductDoc || !readiness.HasPendingProductDraft {
		t.Fatalf("unexpected draft readiness: %#v", readiness)
	}
}

func TestBuildFunctionReadinessTechnicalStale(t *testing.T) {
	readiness := buildFunctionReadiness(&model.Function{
		ProductDocVersion:   1,
		TechnicalDocVersion: 1,
		DocTechnicalStale:   true,
	})
	if readiness.NextAction != readinessActionGenerateTechnicalDoc || readiness.BlockingReason == "" {
		t.Fatalf("unexpected stale readiness: %#v", readiness)
	}
}

func TestBuildFunctionReadinessTechnicalStaleWithDraft(t *testing.T) {
	readiness := buildFunctionReadiness(&model.Function{
		ProductDocVersion:     2,
		TechnicalDocVersion:   1,
		TechnicalDraftVersion: 2,
		DocTechnicalStale:     true,
	})
	if readiness.NextAction != readinessActionAdoptTechnicalDoc || readiness.Label != "待确认技术方案" {
		t.Fatalf("unexpected stale draft readiness: %#v", readiness)
	}
}

func TestBuildFunctionReadinessScaffoldAppStillNeedsGeneration(t *testing.T) {
	readiness := buildFunctionReadiness(&model.Function{
		ProductDocVersion:   1,
		TechnicalDocVersion: 1,
		GeneratedAppID:      "app-1",
		WorkflowStage:       model.FunctionWorkflowStageTechnicalDoc,
		Status:              model.FunctionStatusDraft,
	})
	if readiness.NextAction != readinessActionGeneratePage || readiness.Label != "待生成页面" {
		t.Fatalf("unexpected scaffold readiness: %#v", readiness)
	}
}

func TestBuildFunctionReadinessPublish(t *testing.T) {
	readiness := buildFunctionReadiness(&model.Function{
		ProductDocVersion:   1,
		TechnicalDocVersion: 1,
		CodeVersion:         1,
		GeneratedAppID:      "app-1",
		WorkflowStage:       model.FunctionWorkflowStageCodeGenerated,
		Status:              model.FunctionStatusDraft,
	})
	if readiness.NextAction != readinessActionPublishFunction {
		t.Fatalf("unexpected publish readiness: %#v", readiness)
	}
}

func TestBuildFunctionReadinessNewFunctionDoesNotPublish(t *testing.T) {
	readiness := buildFunctionReadiness(&model.Function{
		GeneratedAppID: "app-1",
		WorkflowStage:  model.FunctionWorkflowStageProductDoc,
		Status:         model.FunctionStatusDraft,
	})
	if readiness.NextAction != readinessActionGenerateProductDoc {
		t.Fatalf("unexpected new function readiness: %#v", readiness)
	}
}

func TestBuildFunctionReadinessPublishedWithCodeDraft(t *testing.T) {
	readiness := buildFunctionReadiness(&model.Function{
		ProductDocVersion:   1,
		TechnicalDocVersion: 1,
		CodeVersion:         1,
		CodeDraftVersion:    2,
		GeneratedAppID:      "app-1",
		Status:              model.FunctionStatusPublished,
	})
	if readiness.NextAction != readinessActionOpenPublished {
		t.Fatalf("unexpected published code draft readiness: %#v", readiness)
	}
}

func TestBuildFunctionReadinessPublishWithPendingCodeDraft(t *testing.T) {
	readiness := buildFunctionReadiness(&model.Function{
		ProductDocVersion:   1,
		TechnicalDocVersion: 1,
		CodeVersion:         1,
		CodeDraftVersion:    2,
		GeneratedAppID:      "app-1",
		Status:              model.FunctionStatusDraft,
	})
	if readiness.NextAction != readinessActionPublishFunction {
		t.Fatalf("unexpected publish readiness with code draft: %#v", readiness)
	}
}

func TestBuildFunctionReadinessPendingProductDraftAfterAdopt(t *testing.T) {
	readiness := buildFunctionReadiness(&model.Function{
		ProductDocVersion:   1,
		ProductDraftVersion: 2,
		TechnicalDocVersion: 1,
	})
	if readiness.NextAction != readinessActionAdoptProductDoc || readiness.Label != "待确认产品方案" {
		t.Fatalf("unexpected pending product readiness: %#v", readiness)
	}
}

func TestBuildFunctionReadinessPendingTechnicalDraftAfterAdopt(t *testing.T) {
	readiness := buildFunctionReadiness(&model.Function{
		ProductDocVersion:     1,
		TechnicalDocVersion:   1,
		TechnicalDraftVersion: 2,
	})
	if readiness.NextAction != readinessActionAdoptTechnicalDoc || readiness.Label != "待确认技术方案" {
		t.Fatalf("unexpected pending technical readiness: %#v", readiness)
	}
}

func TestAdoptDocumentVersionUsesDraftVersion(t *testing.T) {
	if got := adoptDocumentVersion(1, 2); got != 2 {
		t.Fatalf("expected adopted version 2, got %d", got)
	}
}

func TestAdoptDocumentVersionIncrementsWhenDraftNotAhead(t *testing.T) {
	if got := adoptDocumentVersion(1, 1); got != 2 {
		t.Fatalf("expected adopted version 2, got %d", got)
	}
}

func TestAdoptDocumentVersionStartsAtOne(t *testing.T) {
	if got := adoptDocumentVersion(0, 0); got != 1 {
		t.Fatalf("expected adopted version 1, got %d", got)
	}
}

func TestDraftDocumentVersionIncrementsWhenContentDiffers(t *testing.T) {
	version := draftDocumentVersion(
		functionDocumentPayload{Exists: true, Content: "applied"},
		functionDocumentPayload{Exists: true, Content: "draft"},
		1,
		1,
	)
	if version != 2 {
		t.Fatalf("expected draft version 2, got %d", version)
	}
}

func TestDraftDocumentVersionKeepsStoredHigherVersion(t *testing.T) {
	version := draftDocumentVersion(
		functionDocumentPayload{Exists: true, Content: "applied"},
		functionDocumentPayload{Exists: true, Content: "draft"},
		1,
		3,
	)
	if version != 3 {
		t.Fatalf("expected stored draft version 3, got %d", version)
	}
}

func TestFunctionVersionFlagsKeepsTechnicalDraftWhenStale(t *testing.T) {
	productDraftReady, technicalDraftReady, technicalStale, _, _ := functionVersionFlags(&model.Function{
		ProductDocVersion:     2,
		ProductDraftVersion:   2,
		TechnicalDocVersion:   1,
		TechnicalDraftVersion: 2,
		DocTechnicalStale:     true,
	})
	if !technicalStale || !technicalDraftReady || productDraftReady {
		t.Fatalf("unexpected version flags: product=%v technical=%v stale=%v", productDraftReady, technicalDraftReady, technicalStale)
	}
}

func TestFunctionVersionFlagsUseRecordedDependencyVersions(t *testing.T) {
	_, _, technicalStale, codeStale, _ := functionVersionFlags(&model.Function{
		ProductDocVersion:             3,
		TechnicalDocVersion:           5,
		TechnicalSourceProductVersion: 2,
		CodeVersion:                   9,
		CodeSourceTechnicalVersion:    4,
	})
	if !technicalStale || !codeStale {
		t.Fatalf("expected stale dependencies, technical=%v code=%v", technicalStale, codeStale)
	}

	_, _, technicalStale, codeStale, _ = functionVersionFlags(&model.Function{
		ProductDocVersion:             3,
		TechnicalDocVersion:           5,
		TechnicalSourceProductVersion: 3,
		CodeVersion:                   1,
		CodeSourceTechnicalVersion:    5,
	})
	if technicalStale || codeStale {
		t.Fatalf("expected synchronized dependencies, technical=%v code=%v", technicalStale, codeStale)
	}
}
