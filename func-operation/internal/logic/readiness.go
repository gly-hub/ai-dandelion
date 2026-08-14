package logic

import (
	"strings"

	"github.com/team-dandelion/ai-dandelion/func-operation/internal/model"
	funcoperation "github.com/team-dandelion/ai-dandelion/proto/func-operation"
)

const (
	readinessActionGenerateProductDoc   = "generate_product_doc"
	readinessActionAdoptProductDoc      = "adopt_product_doc"
	readinessActionGenerateTechnicalDoc = "generate_technical_doc"
	readinessActionAdoptTechnicalDoc    = "adopt_technical_doc"
	readinessActionGeneratePage         = "generate_page"
	readinessActionPreviewLatestPage    = "preview_latest_page"
	readinessActionPublishFunction      = "publish_function"
	readinessActionOpenPublished        = "open_published_function"
)

func buildFunctionReadiness(function *model.Function) *funcoperation.FunctionReadiness {
	if function == nil {
		return &funcoperation.FunctionReadiness{}
	}

	productDraftReady, technicalDraftReady, technicalStale, codeStale, codeDraftReady := functionVersionFlags(function)
	readiness := &funcoperation.FunctionReadiness{
		HasPendingProductDraft:   productDraftReady,
		HasPendingTechnicalDraft: technicalDraftReady,
		HasPendingCodeDraft:      codeDraftReady,
	}

	if function.ProductDocVersion == 0 {
		readiness.Label = "待完善产品方案"
		if productDraftReady {
			readiness.NextAction = readinessActionAdoptProductDoc
			return readiness
		}
		readiness.NextAction = readinessActionGenerateProductDoc
		return readiness
	}

	if productDraftReady {
		readiness.Label = "待确认产品方案"
		readiness.NextAction = readinessActionAdoptProductDoc
		return readiness
	}

	if technicalDraftReady {
		readiness.Label = "待确认技术方案"
		readiness.NextAction = readinessActionAdoptTechnicalDoc
		return readiness
	}

	if function.TechnicalDocVersion == 0 || technicalStale {
		readiness.Label = "待完善技术方案"
		if technicalStale {
			readiness.BlockingReason = "产品方案已更新，技术方案需要同步"
		}
		readiness.NextAction = readinessActionGenerateTechnicalDoc
		return readiness
	}

	if codeStale {
		readiness.Label = "待生成页面"
		readiness.BlockingReason = "技术方案已更新，页面需要重新生成"
		readiness.NextAction = readinessActionGeneratePage
		return readiness
	}

	if !functionHasGeneratedCode(function) {
		readiness.Label = "待生成页面"
		readiness.NextAction = readinessActionGeneratePage
		return readiness
	}

	if function.Status == model.FunctionStatusPublished {
		if productDraftReady || technicalDraftReady {
			readiness.Label = "已发布"
			readiness.BlockingReason = "发布后有新改动，尚未采用"
			if technicalDraftReady {
				readiness.NextAction = readinessActionAdoptTechnicalDoc
			} else {
				readiness.NextAction = readinessActionAdoptProductDoc
			}
			return readiness
		}
		readiness.Label = "已发布"
		readiness.NextAction = readinessActionOpenPublished
		return readiness
	}

	readiness.Label = "页面已就绪"
	readiness.NextAction = readinessActionPublishFunction
	return readiness
}

func functionHasGeneratedCode(function *model.Function) bool {
	return function != nil && strings.TrimSpace(function.GeneratedAppID) != "" && function.CodeVersion > 0
}
