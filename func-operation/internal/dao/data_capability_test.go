package dao

import (
	"strings"
	"testing"
)

func TestCompileDataModelGeneratesAppScopedTable(t *testing.T) {
	t.Parallel()

	model := DataModel{
		Name: "book",
		Fields: []DataField{
			{Name: "title", Type: "string", Required: true, MaxLength: 100},
			{Name: "stock", Type: "int"},
		},
		Indexes: []string{"title"},
	}

	first, err := compileDataModel("app-a", model)
	if err != nil {
		t.Fatalf("compileDataModel() error = %v", err)
	}
	second, err := compileDataModel("app-b", model)
	if err != nil {
		t.Fatalf("compileDataModel() error = %v", err)
	}
	if first.tableName == second.tableName {
		t.Fatalf("table names should differ across apps, got %q", first.tableName)
	}
	if !strings.HasPrefix(first.tableName, "func_app_") || !strings.HasSuffix(first.tableName, "__book") {
		t.Fatalf("unexpected app-scoped table name %q", first.tableName)
	}
}

func TestValidateDataRecordRejectsUnknownAndSystemFields(t *testing.T) {
	t.Parallel()

	model, err := compileDataModel("app-a", DataModel{
		Name:   "book",
		Fields: []DataField{{Name: "title", Type: "string", Required: true}},
	})
	if err != nil {
		t.Fatalf("compileDataModel() error = %v", err)
	}

	if _, err := validateDataRecord(model, map[string]any{"title": "Go", "other": "x"}, false); err == nil {
		t.Fatal("validateDataRecord() should reject undeclared field")
	}
	if _, err := validateDataRecord(model, map[string]any{"title": "Go", "id": 1}, false); err == nil {
		t.Fatal("validateDataRecord() should reject platform-managed field")
	}
	if _, err := validateDataRecord(model, map[string]any{}, false); err == nil {
		t.Fatal("validateDataRecord() should reject missing required field")
	}
}

func TestCompileDataWhereRejectsUnknownField(t *testing.T) {
	t.Parallel()

	model, err := compileDataModel("app-a", DataModel{
		Name:   "book",
		Fields: []DataField{{Name: "title", Type: "string"}},
	})
	if err != nil {
		t.Fatalf("compileDataModel() error = %v", err)
	}

	if _, _, err := compileDataWhere(model, []DataCondition{{Field: "other", Op: "eq", Value: "x"}}); err == nil {
		t.Fatal("compileDataWhere() should reject undeclared field")
	}
	where, args, err := compileDataWhere(model, []DataCondition{{Field: "title", Op: "contains", Value: "Go"}})
	if err != nil {
		t.Fatalf("compileDataWhere() error = %v", err)
	}
	if where != " WHERE `title` LIKE ?" {
		t.Fatalf("where = %q", where)
	}
	if len(args) != 1 || args[0] != "%Go%" {
		t.Fatalf("args = %#v", args)
	}
}

func TestDataModelDDLUsesDeclaredFieldsOnly(t *testing.T) {
	t.Parallel()

	model, err := compileDataModel("app-a", DataModel{
		Name: "book",
		Fields: []DataField{
			{Name: "title", Type: "string", Required: true, MaxLength: 100},
			{Name: "note", Type: "text"},
		},
	})
	if err != nil {
		t.Fatalf("compileDataModel() error = %v", err)
	}
	ddl, err := dataModelDDL(model, "mysql")
	if err != nil {
		t.Fatalf("dataModelDDL() error = %v", err)
	}
	for _, want := range []string{"`title` VARCHAR(100) NOT NULL", "`note` TEXT NULL", "`uuid` VARCHAR(120) NOT NULL UNIQUE"} {
		if !strings.Contains(ddl, want) {
			t.Fatalf("ddl %q missing %q", ddl, want)
		}
	}
}

func TestDataColumnDefinitionForAdditiveMigrationIsNullable(t *testing.T) {
	t.Parallel()

	field := DataField{Name: "isbn", Type: "string", Required: true, MaxLength: 32}
	initial, err := dataColumnDefinition(field, false, "mysql")
	if err != nil {
		t.Fatalf("dataColumnDefinition(initial) error = %v", err)
	}
	if initial != "`isbn` VARCHAR(32) NOT NULL" {
		t.Fatalf("initial column = %q", initial)
	}
	additive, err := dataColumnDefinition(field, true, "mysql")
	if err != nil {
		t.Fatalf("dataColumnDefinition(additive) error = %v", err)
	}
	if additive != "`isbn` VARCHAR(32) NULL" {
		t.Fatalf("additive column = %q", additive)
	}
}

func TestCompileDataRelationsValidatesFields(t *testing.T) {
	t.Parallel()

	models, err := compileDataModels("app-a", []DataModel{
		{Name: "book", Fields: []DataField{{Name: "categoryId", Type: "id"}}},
		{Name: "category", Fields: []DataField{{Name: "name", Type: "string"}}},
	})
	if err != nil {
		t.Fatalf("compileDataModels() error = %v", err)
	}
	relations, err := compileDataRelations(models, []DataRelation{{
		Name: "bookCategory",
		From: "book.categoryId",
		To:   "category.id",
	}})
	if err != nil {
		t.Fatalf("compileDataRelations() error = %v", err)
	}
	if relations["bookCategory"].fromModel != "book" || relations["bookCategory"].toModel != "category" {
		t.Fatalf("unexpected relation runtime: %#v", relations["bookCategory"])
	}
	if _, err := compileDataRelations(models, []DataRelation{{
		Name: "bad",
		From: "book.missing",
		To:   "category.id",
	}}); err == nil {
		t.Fatal("compileDataRelations() should reject unknown field")
	}
}

func TestCompileJoinClausesRequireJoinedModels(t *testing.T) {
	t.Parallel()

	models, err := compileDataModels("app-a", []DataModel{
		{Name: "book", Fields: []DataField{{Name: "title", Type: "string"}, {Name: "categoryId", Type: "id"}}},
		{Name: "category", Fields: []DataField{{Name: "name", Type: "string"}}},
	})
	if err != nil {
		t.Fatalf("compileDataModels() error = %v", err)
	}
	aliases := map[string]string{"book": "t0", "category": "t1"}
	selectSQL, err := compileJoinSelect(models, aliases, []string{"book.title", "category.name"})
	if err != nil {
		t.Fatalf("compileJoinSelect() error = %v", err)
	}
	for _, want := range []string{"`t0`.`title` AS `book_title`", "`t1`.`name` AS `category_name`"} {
		if !strings.Contains(selectSQL, want) {
			t.Fatalf("selectSQL %q missing %q", selectSQL, want)
		}
	}
	whereSQL, args, err := compileJoinWhere(models, aliases, []DataCondition{{Field: "category.name", Op: "eq", Value: "科技"}})
	if err != nil {
		t.Fatalf("compileJoinWhere() error = %v", err)
	}
	if whereSQL != " WHERE `t1`.`name` = ?" || len(args) != 1 || args[0] != "科技" {
		t.Fatalf("whereSQL=%q args=%#v", whereSQL, args)
	}
	if _, err := compileJoinSelect(models, map[string]string{"book": "t0"}, []string{"category.name"}); err == nil {
		t.Fatal("compileJoinSelect() should reject fields from non-joined model")
	}
}
