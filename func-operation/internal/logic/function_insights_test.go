package logic

import (
	"testing"

	"github.com/team-dandelion/ai-dandelion/func-operation/internal/model"
)

func TestExtractSectionItems(t *testing.T) {
	content := `1. 功能定位
客户资料管理与分层维护。

2. 核心页面
- 客户列表页
- 客户详情页

3. 数据模型
- customer: 基础资料
- customer_tag: 标签

4. API 设计
- GET /customers
- POST /customers

5. 下一步实现计划
- 先完成列表与详情
- 再补充编辑能力`

	pages := extractSectionItems(content, []string{"核心页面"})
	if len(pages) != 2 || pages[0] != "客户列表页" || pages[1] != "客户详情页" {
		t.Fatalf("unexpected pages: %#v", pages)
	}

	models := extractSectionItems(content, []string{"数据模型"})
	if len(models) != 2 {
		t.Fatalf("unexpected models: %#v", models)
	}

	apis := extractSectionItems(content, []string{"API 设计"})
	if len(apis) != 2 || apis[0] != "GET /customers" {
		t.Fatalf("unexpected apis: %#v", apis)
	}

	next := extractSectionItems(content, []string{"下一步实现计划", "下一步"})
	if len(next) != 2 || next[0] != "先完成列表与详情" {
		t.Fatalf("unexpected next steps: %#v", next)
	}
}

func TestBuildInsightsFromDocs(t *testing.T) {
	function := &model.Function{
		ProductDoc: `1. 业务目标
- 支持运营录入客户信息
- 支持按标签筛选客户

2. 核心页面
- 客户列表页
- 客户详情页`,
		TechnicalDoc: `1. 数据模型
- customer
- customer_tag

2. API 设计
- GET /customers
- POST /customers

3. 实施步骤
- 先完成列表查询
- 再补充编辑表单`,
	}

	insights := buildInsightsFromDocs(function)
	if insights.Summary == "" {
		t.Fatal("expected summary")
	}
	if len(insights.CorePages) != 2 || insights.CorePages[0] != "客户列表页" {
		t.Fatalf("unexpected core pages: %#v", insights.CorePages)
	}
	if len(insights.DataModels) != 2 || insights.DataModels[0] != "customer" {
		t.Fatalf("unexpected data models: %#v", insights.DataModels)
	}
	if len(insights.APIs) != 2 || insights.APIs[0] != "GET /customers" {
		t.Fatalf("unexpected apis: %#v", insights.APIs)
	}
}
