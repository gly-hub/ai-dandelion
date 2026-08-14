package dao

import (
	"context"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
	"gorm.io/gorm"
)

type Function struct {
	db *gorm.DB
}

func NewFunction(db *gorm.DB) *Function {
	return &Function{db: db}
}

func (f *Function) List(ctx context.Context) ([]model.Function, error) {
	var functions []model.Function
	err := f.db.WithContext(ctx).
		Order("updated_at DESC").
		Find(&functions).
		Error
	return functions, err
}

// ListLegacyPublished returns the pre-release-model records that need a
// one-time immutable release record during an upgrade.
func (f *Function) ListLegacyPublished(ctx context.Context) ([]model.Function, error) {
	var functions []model.Function
	err := f.db.WithContext(ctx).
		Where("status = ? AND generated_app_id <> '' AND active_release_id = ''", model.FunctionStatusPublished).
		Order("id ASC").
		Find(&functions).
		Error
	return functions, err
}

func (f *Function) Exists(ctx context.Context, uuid string) (bool, error) {
	var count int64
	err := f.db.WithContext(ctx).
		Model(&model.Function{}).
		Where("uuid = ?", uuid).
		Count(&count).Error
	return count > 0, err
}

func (f *Function) Create(ctx context.Context, function *model.Function) error {
	return f.db.WithContext(ctx).Create(function).Error
}

func (f *Function) Get(ctx context.Context, uuid string) (*model.Function, error) {
	var function model.Function
	err := f.db.WithContext(ctx).
		Where("uuid = ?", uuid).
		First(&function).
		Error
	if err != nil {
		return nil, err
	}
	return &function, nil
}

func (f *Function) GetByGeneratedAppID(ctx context.Context, appID string) (*model.Function, error) {
	var function model.Function
	err := f.db.WithContext(ctx).
		Where("generated_app_id = ?", appID).
		First(&function).
		Error
	if err != nil {
		return nil, err
	}
	return &function, nil
}

func (f *Function) Delete(ctx context.Context, uuid string) error {
	result := f.db.WithContext(ctx).
		Where("uuid = ?", uuid).
		Delete(&model.Function{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (f *Function) Update(ctx context.Context, function *model.Function) error {
	return f.db.WithContext(ctx).
		Model(&model.Function{}).
		Where("id = ?", function.ID).
		Updates(map[string]any{
			"name":                    function.Name,
			"description":             function.Description,
			"status":                  function.Status,
			"workflow_stage":          function.WorkflowStage,
			"entry":                   function.Entry,
			"product_doc_path":        function.ProductDocPath,
			"technical_doc_path":      function.TechnicalDocPath,
			"product_session_id":      function.ProductSessionID,
			"technical_session_id":    function.TechnicalSessionID,
			"generation_session_id":   function.GenerationSessionID,
			"generated_app_id":        function.GeneratedAppID,
			"active_release_id":       function.ActiveReleaseID,
			"function_version":        function.FunctionVersion,
			"product_doc_version":     function.ProductDocVersion,
			"product_draft_version":   function.ProductDraftVersion,
			"technical_doc_version":   function.TechnicalDocVersion,
			"technical_draft_version": function.TechnicalDraftVersion,
			"code_version":            function.CodeVersion,
			"code_draft_version":      function.CodeDraftVersion,
			"menu_parent_id":          function.MenuParentID,
			"menu_id":                 function.MenuID,
			"updated_at":              function.UpdatedAt,
		}).Error
}
