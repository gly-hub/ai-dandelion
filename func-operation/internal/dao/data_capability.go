package dao

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/team-dandelion/ai-dandelion/toolbox/gormutil"
)

const (
	defaultDataLimit = 20
	maxDataLimit     = 100
)

var dataIdentifierPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

type DataModel struct {
	Name    string      `json:"name"`
	Label   string      `json:"label,omitempty"`
	Fields  []DataField `json:"fields"`
	Indexes []string    `json:"indexes,omitempty"`
}

type DataRelation struct {
	Name string `json:"name"`
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type,omitempty"`
}

type DataQuery struct {
	Name    string          `json:"name"`
	From    string          `json:"from"`
	Joins   []DataJoin      `json:"joins,omitempty"`
	Select  []string        `json:"select"`
	Where   []DataCondition `json:"where,omitempty"`
	OrderBy []DataOrder     `json:"orderBy,omitempty"`
	Limit   int             `json:"limit,omitempty"`
}

type DataField struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Required  bool     `json:"required,omitempty"`
	MaxLength int      `json:"maxLength,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	Values    []string `json:"values,omitempty"`
}

type DataListRequest struct {
	Model   string          `json:"model"`
	Where   []DataCondition `json:"where,omitempty"`
	OrderBy []DataOrder     `json:"orderBy,omitempty"`
	Page    DataPage        `json:"page,omitempty"`
	Limit   int             `json:"limit,omitempty"`
}

type DataJoinRequest struct {
	From    string          `json:"from"`
	Joins   []DataJoin      `json:"joins,omitempty"`
	Select  []string        `json:"select"`
	Where   []DataCondition `json:"where,omitempty"`
	OrderBy []DataOrder     `json:"orderBy,omitempty"`
	Limit   int             `json:"limit,omitempty"`
}

type DataRunQueryRequest struct {
	Query  string         `json:"query"`
	Params map[string]any `json:"params,omitempty"`
}

type DataJoin struct {
	Relation string `json:"relation"`
	Type     string `json:"type,omitempty"`
}

type DataGetRequest struct {
	Model string `json:"model"`
	ID    any    `json:"id"`
}

type DataWriteRequest struct {
	Model  string         `json:"model"`
	ID     any            `json:"id,omitempty"`
	Record map[string]any `json:"record,omitempty"`
	Patch  map[string]any `json:"patch,omitempty"`
}

type DataCondition struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value any    `json:"value"`
}

type DataOrder struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type DataPage struct {
	Limit int `json:"limit,omitempty"`
}

type DataListResult struct {
	Rows  []map[string]any `json:"rows"`
	Total int              `json:"total"`
}

type DataWriteResult struct {
	ID           any            `json:"id,omitempty"`
	RowsAffected int64          `json:"rowsAffected,omitempty"`
	Record       map[string]any `json:"record,omitempty"`
}

type dataModelRuntime struct {
	model     DataModel
	tableName string
	fields    map[string]DataField
}

type dataRelationRuntime struct {
	relation  DataRelation
	fromModel string
	fromField string
	toModel   string
	toField   string
}

type DataModelSummary struct {
	Model     DataModel
	TableName string
	RowCount  int64
}

func (g *GeneratedApp) ListDataModelSummaries(ctx context.Context, appID string, models []DataModel) ([]DataModelSummary, error) {
	out := make([]DataModelSummary, 0, len(models))
	for _, model := range models {
		runtimeModel, err := compileDataModel(appID, model)
		if err != nil {
			return nil, err
		}
		var count int64
		if g.db != nil && g.db.WithContext(ctx).Migrator().HasTable(runtimeModel.tableName) {
			row := g.db.WithContext(ctx).Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s", quoteIdent(runtimeModel.tableName))).Row()
			if err := row.Scan(&count); err != nil {
				return nil, fmt.Errorf("count data model %q: %w", model.Name, err)
			}
		}
		out = append(out, DataModelSummary{Model: runtimeModel.model, TableName: runtimeModel.tableName, RowCount: count})
	}
	return out, nil
}

func (g *GeneratedApp) DropDataModel(ctx context.Context, appID string, model DataModel) error {
	runtimeModel, err := compileDataModel(appID, model)
	if err != nil {
		return err
	}
	if g.db == nil || !g.db.WithContext(ctx).Migrator().HasTable(runtimeModel.tableName) {
		return nil
	}
	statement := fmt.Sprintf("DROP TABLE %s", quoteIdent(runtimeModel.tableName))
	if err := g.db.WithContext(ctx).Exec(statement).Error; err != nil {
		return fmt.Errorf("drop data model table %q: %w", model.Name, err)
	}
	return nil
}

func (g *GeneratedApp) ApplyDataModels(ctx context.Context, appID string, models []DataModel) error {
	for _, model := range models {
		runtimeModel, err := compileDataModel(appID, model)
		if err != nil {
			return err
		}
		if !g.db.WithContext(ctx).Migrator().HasTable(runtimeModel.tableName) {
			ddls, err := dataModelDDLs(runtimeModel, gormutil.DialectName(g.db))
			if err != nil {
				return err
			}
			for _, ddl := range ddls {
				if err := g.db.WithContext(ctx).Exec(ddl).Error; err != nil {
					return fmt.Errorf("apply data model schema %q: %w", model.Name, err)
				}
			}
			continue
		}
		if err := g.applyDataModelAdditiveChanges(ctx, runtimeModel); err != nil {
			return err
		}
	}
	return nil
}

func (g *GeneratedApp) applyDataModelAdditiveChanges(ctx context.Context, model dataModelRuntime) error {
	existingColumns, err := g.dataModelColumns(ctx, model.tableName)
	if err != nil {
		return fmt.Errorf("inspect data model schema %q: %w", model.model.Name, err)
	}
	fieldNames := sortedDataFieldNames(model.fields)
	for _, name := range fieldNames {
		if isSystemDataField(name) {
			continue
		}
		if existingColumns[strings.ToLower(name)] {
			continue
		}
		columnSQL, err := dataColumnDefinition(model.fields[name], true, gormutil.DialectName(g.db))
		if err != nil {
			return err
		}
		statement := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", quoteIdent(model.tableName), columnSQL)
		if err := g.db.WithContext(ctx).Exec(statement).Error; err != nil {
			return fmt.Errorf("add data model field %q.%q: %w", model.model.Name, name, err)
		}
	}
	existingIndexes, err := g.dataModelIndexes(ctx, model.tableName)
	if err != nil {
		return fmt.Errorf("inspect data model indexes %q: %w", model.model.Name, err)
	}
	for _, index := range model.model.Indexes {
		index = strings.TrimSpace(index)
		if index == "" || isSystemDataField(index) {
			continue
		}
		if _, ok := model.fields[index]; !ok {
			return fmt.Errorf("index field %q is not declared", index)
		}
		indexName := "idx_" + index
		if existingIndexes[strings.ToLower(indexName)] {
			continue
		}
		statement := dataCreateIndexSQL(gormutil.DialectName(g.db), model.tableName, indexName, index)
		if err := g.db.WithContext(ctx).Exec(statement).Error; err != nil {
			return fmt.Errorf("add data model index %q.%q: %w", model.model.Name, indexName, err)
		}
	}
	return nil
}

func (g *GeneratedApp) dataModelColumns(ctx context.Context, tableName string) (map[string]bool, error) {
	columnTypes, err := g.db.WithContext(ctx).Migrator().ColumnTypes(tableName)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(columnTypes))
	for _, columnType := range columnTypes {
		out[strings.ToLower(columnType.Name())] = true
	}
	return out, nil
}

func (g *GeneratedApp) dataModelIndexes(ctx context.Context, tableName string) (map[string]bool, error) {
	if gormutil.IsSQLite(g.db) {
		type sqliteIndexRow struct {
			Name string `gorm:"column:name"`
		}
		var rows []sqliteIndexRow
		if err := g.db.WithContext(ctx).Raw("PRAGMA index_list(" + quoteIdent(tableName) + ")").Scan(&rows).Error; err != nil {
			return nil, err
		}
		out := make(map[string]bool, len(rows))
		for _, row := range rows {
			out[strings.ToLower(row.Name)] = true
		}
		return out, nil
	}

	type indexRow struct {
		KeyName string `gorm:"column:Key_name"`
	}
	var rows []indexRow
	if err := g.db.WithContext(ctx).Raw("SHOW INDEX FROM " + quoteIdent(tableName)).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, row := range rows {
		out[strings.ToLower(row.KeyName)] = true
	}
	return out, nil
}

func (g *GeneratedApp) DataList(ctx context.Context, appID string, models []DataModel, req DataListRequest) (DataListResult, error) {
	runtimeModel, err := findDataModel(appID, models, req.Model)
	if err != nil {
		return DataListResult{}, err
	}
	limit := normalizeDataLimit(firstPositive(req.Page.Limit, req.Limit))
	whereSQL, args, err := compileDataWhere(runtimeModel, req.Where)
	if err != nil {
		return DataListResult{}, err
	}
	orderSQL, err := compileDataOrder(runtimeModel, req.OrderBy)
	if err != nil {
		return DataListResult{}, err
	}
	query := fmt.Sprintf("SELECT * FROM %s%s%s LIMIT ?", quoteIdent(runtimeModel.tableName), whereSQL, orderSQL)
	args = append(args, limit)
	rows, err := g.queryRows(ctx, query, args)
	if err != nil {
		return DataListResult{}, err
	}
	return DataListResult{Rows: rows, Total: len(rows)}, nil
}

func (g *GeneratedApp) DataJoinQuery(ctx context.Context, appID string, models []DataModel, relations []DataRelation, req DataJoinRequest) (DataListResult, error) {
	if len(req.Select) == 0 {
		return DataListResult{}, errors.New("select is required")
	}
	if len(req.Joins) > 3 {
		return DataListResult{}, errors.New("join depth exceeds limit")
	}
	modelMap, err := compileDataModels(appID, models)
	if err != nil {
		return DataListResult{}, err
	}
	base, ok := modelMap[strings.TrimSpace(req.From)]
	if !ok {
		return DataListResult{}, fmt.Errorf("from model %q is not declared", req.From)
	}
	relationMap, err := compileDataRelations(modelMap, relations)
	if err != nil {
		return DataListResult{}, err
	}
	aliases := map[string]string{base.model.Name: "t0"}
	joinedModels := map[string]dataModelRuntime{base.model.Name: base}
	joinSQL := make([]string, 0, len(req.Joins))
	for i, join := range req.Joins {
		relation, ok := relationMap[strings.TrimSpace(join.Relation)]
		if !ok {
			return DataListResult{}, fmt.Errorf("relation %q is not declared", join.Relation)
		}
		leftAlias, rightModel, rightField, leftField, err := resolveJoinSide(aliases, modelMap, relation)
		if err != nil {
			return DataListResult{}, err
		}
		rightAlias := fmt.Sprintf("t%d", i+1)
		joinType := strings.ToUpper(strings.TrimSpace(join.Type))
		if joinType == "" {
			joinType = "LEFT"
		}
		if joinType != "LEFT" && joinType != "INNER" {
			return DataListResult{}, fmt.Errorf("unsupported join type %q", join.Type)
		}
		aliases[rightModel.model.Name] = rightAlias
		joinedModels[rightModel.model.Name] = rightModel
		joinSQL = append(joinSQL, fmt.Sprintf("%s JOIN %s %s ON %s.%s = %s.%s",
			joinType,
			quoteIdent(rightModel.tableName),
			quoteIdent(rightAlias),
			quoteIdent(leftAlias),
			quoteIdent(leftField),
			quoteIdent(rightAlias),
			quoteIdent(rightField),
		))
	}
	selectSQL, err := compileJoinSelect(joinedModels, aliases, req.Select)
	if err != nil {
		return DataListResult{}, err
	}
	whereSQL, args, err := compileJoinWhere(joinedModels, aliases, req.Where)
	if err != nil {
		return DataListResult{}, err
	}
	orderSQL, err := compileJoinOrder(joinedModels, aliases, req.OrderBy)
	if err != nil {
		return DataListResult{}, err
	}
	limit := normalizeDataLimit(req.Limit)
	query := fmt.Sprintf("SELECT %s FROM %s %s %s%s%s LIMIT ?",
		selectSQL,
		quoteIdent(base.tableName),
		quoteIdent("t0"),
		strings.Join(joinSQL, " "),
		whereSQL,
		orderSQL,
	)
	args = append(args, limit)
	rows, err := g.queryRows(ctx, query, args)
	if err != nil {
		return DataListResult{}, err
	}
	return DataListResult{Rows: rows, Total: len(rows)}, nil
}

func (g *GeneratedApp) DataRunQuery(ctx context.Context, appID string, models []DataModel, relations []DataRelation, queries []DataQuery, req DataRunQueryRequest) (DataListResult, error) {
	queryName := strings.TrimSpace(req.Query)
	for _, query := range queries {
		if strings.TrimSpace(query.Name) != queryName {
			continue
		}
		return g.DataJoinQuery(ctx, appID, models, relations, DataJoinRequest{
			From:    query.From,
			Joins:   query.Joins,
			Select:  query.Select,
			Where:   query.Where,
			OrderBy: query.OrderBy,
			Limit:   query.Limit,
		})
	}
	return DataListResult{}, fmt.Errorf("data query %q is not declared", queryName)
}

func (g *GeneratedApp) DataGet(ctx context.Context, appID string, models []DataModel, req DataGetRequest) (map[string]any, error) {
	runtimeModel, err := findDataModel(appID, models, req.Model)
	if err != nil {
		return nil, err
	}
	rows, err := g.queryRows(ctx, fmt.Sprintf("SELECT * FROM %s WHERE %s = ? LIMIT 1", quoteIdent(runtimeModel.tableName), quoteIdent("id")), []any{req.ID})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("record not found")
	}
	return rows[0], nil
}

func (g *GeneratedApp) DataCreate(ctx context.Context, appID string, models []DataModel, req DataWriteRequest) (DataWriteResult, error) {
	runtimeModel, err := findDataModel(appID, models, req.Model)
	if err != nil {
		return DataWriteResult{}, err
	}
	record, err := validateDataRecord(runtimeModel, req.Record, false)
	if err != nil {
		return DataWriteResult{}, err
	}
	now := nowUnixMicro()
	record["uuid"] = uuid.NewString()
	record["created_at"] = now
	record["updated_at"] = now

	columns := sortedKeys(record)
	placeholders := make([]string, 0, len(columns))
	args := make([]any, 0, len(columns))
	for _, column := range columns {
		placeholders = append(placeholders, "?")
		args = append(args, record[column])
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quoteIdent(runtimeModel.tableName),
		quoteIdentList(columns),
		strings.Join(placeholders, ", "),
	)
	result := g.db.WithContext(ctx).Exec(query, args...)
	if result.Error != nil {
		return DataWriteResult{}, fmt.Errorf("create data record: %w", result.Error)
	}
	return DataWriteResult{RowsAffected: result.RowsAffected, Record: record}, nil
}

func (g *GeneratedApp) DataUpdate(ctx context.Context, appID string, models []DataModel, req DataWriteRequest) (DataWriteResult, error) {
	runtimeModel, err := findDataModel(appID, models, req.Model)
	if err != nil {
		return DataWriteResult{}, err
	}
	patch, err := validateDataRecord(runtimeModel, req.Patch, true)
	if err != nil {
		return DataWriteResult{}, err
	}
	if len(patch) == 0 {
		return DataWriteResult{}, errors.New("patch is required")
	}
	patch["updated_at"] = nowUnixMicro()
	columns := sortedKeys(patch)
	assignments := make([]string, 0, len(columns))
	args := make([]any, 0, len(columns)+1)
	for _, column := range columns {
		assignments = append(assignments, quoteIdent(column)+" = ?")
		args = append(args, patch[column])
	}
	args = append(args, req.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?",
		quoteIdent(runtimeModel.tableName),
		strings.Join(assignments, ", "),
		quoteIdent("id"),
	)
	result := g.db.WithContext(ctx).Exec(query, args...)
	if result.Error != nil {
		return DataWriteResult{}, fmt.Errorf("update data record: %w", result.Error)
	}
	return DataWriteResult{ID: req.ID, RowsAffected: result.RowsAffected}, nil
}

func (g *GeneratedApp) DataDelete(ctx context.Context, appID string, models []DataModel, req DataGetRequest) (DataWriteResult, error) {
	runtimeModel, err := findDataModel(appID, models, req.Model)
	if err != nil {
		return DataWriteResult{}, err
	}
	result := g.db.WithContext(ctx).Exec(
		fmt.Sprintf("DELETE FROM %s WHERE %s = ?", quoteIdent(runtimeModel.tableName), quoteIdent("id")),
		req.ID,
	)
	if result.Error != nil {
		return DataWriteResult{}, fmt.Errorf("delete data record: %w", result.Error)
	}
	return DataWriteResult{ID: req.ID, RowsAffected: result.RowsAffected}, nil
}

func (g *GeneratedApp) queryRows(ctx context.Context, query string, args []any) ([]map[string]any, error) {
	rows, err := g.db.WithContext(ctx).Raw(query, args...).Rows()
	if err != nil {
		return nil, fmt.Errorf("query data records: %w", err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		dest := make([]any, len(columns))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(columns))
		for i, column := range columns {
			row[column] = normalizeSQLValue(values[i], sqlColumnType(columnTypes, i))
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func findDataModel(appID string, models []DataModel, name string) (dataModelRuntime, error) {
	name = strings.TrimSpace(name)
	for _, model := range models {
		if model.Name != name {
			continue
		}
		return compileDataModel(appID, model)
	}
	return dataModelRuntime{}, fmt.Errorf("data model %q is not declared", name)
}

func compileDataModels(appID string, models []DataModel) (map[string]dataModelRuntime, error) {
	out := make(map[string]dataModelRuntime, len(models))
	for _, model := range models {
		runtimeModel, err := compileDataModel(appID, model)
		if err != nil {
			return nil, err
		}
		if _, exists := out[runtimeModel.model.Name]; exists {
			return nil, fmt.Errorf("duplicate data model %q", runtimeModel.model.Name)
		}
		out[runtimeModel.model.Name] = runtimeModel
	}
	return out, nil
}

func compileDataModel(appID string, model DataModel) (dataModelRuntime, error) {
	model.Name = strings.TrimSpace(model.Name)
	if !dataIdentifierPattern.MatchString(model.Name) {
		return dataModelRuntime{}, fmt.Errorf("invalid data model name %q", model.Name)
	}
	fields := map[string]DataField{
		"id":         {Name: "id", Type: "id"},
		"uuid":       {Name: "uuid", Type: "string", MaxLength: 120},
		"created_at": {Name: "created_at", Type: "int"},
		"updated_at": {Name: "updated_at", Type: "int"},
	}
	for _, field := range model.Fields {
		field.Name = strings.TrimSpace(field.Name)
		field.Type = normalizeDataFieldType(field.Type)
		if !dataIdentifierPattern.MatchString(field.Name) {
			return dataModelRuntime{}, fmt.Errorf("invalid data field name %q", field.Name)
		}
		if _, ok := fields[field.Name]; ok && !isSystemDataField(field.Name) {
			return dataModelRuntime{}, fmt.Errorf("duplicate data field %q", field.Name)
		}
		if isSystemDataField(field.Name) {
			continue
		}
		if field.Type == "" {
			return dataModelRuntime{}, fmt.Errorf("data field %q type is required", field.Name)
		}
		fields[field.Name] = field
	}
	model.Fields = make([]DataField, 0, len(fields))
	for _, name := range sortedDataFieldNames(fields) {
		if isSystemDataField(name) {
			continue
		}
		model.Fields = append(model.Fields, fields[name])
	}
	return dataModelRuntime{
		model:     model,
		tableName: dataTableName(appID, model.Name),
		fields:    fields,
	}, nil
}

func compileDataRelations(models map[string]dataModelRuntime, relations []DataRelation) (map[string]dataRelationRuntime, error) {
	out := make(map[string]dataRelationRuntime, len(relations))
	for _, relation := range relations {
		relation.Name = strings.TrimSpace(relation.Name)
		if !dataIdentifierPattern.MatchString(relation.Name) {
			return nil, fmt.Errorf("invalid data relation name %q", relation.Name)
		}
		fromModel, fromField, err := splitQualifiedField(relation.From)
		if err != nil {
			return nil, fmt.Errorf("invalid relation %q from: %w", relation.Name, err)
		}
		toModel, toField, err := splitQualifiedField(relation.To)
		if err != nil {
			return nil, fmt.Errorf("invalid relation %q to: %w", relation.Name, err)
		}
		if err := ensureModelField(models, fromModel, fromField); err != nil {
			return nil, fmt.Errorf("invalid relation %q from: %w", relation.Name, err)
		}
		if err := ensureModelField(models, toModel, toField); err != nil {
			return nil, fmt.Errorf("invalid relation %q to: %w", relation.Name, err)
		}
		if _, exists := out[relation.Name]; exists {
			return nil, fmt.Errorf("duplicate data relation %q", relation.Name)
		}
		out[relation.Name] = dataRelationRuntime{
			relation:  relation,
			fromModel: fromModel,
			fromField: fromField,
			toModel:   toModel,
			toField:   toField,
		}
	}
	return out, nil
}

func resolveJoinSide(joined map[string]string, models map[string]dataModelRuntime, relation dataRelationRuntime) (string, dataModelRuntime, string, string, error) {
	if alias, ok := joined[relation.fromModel]; ok {
		right, ok := models[relation.toModel]
		if !ok {
			return "", dataModelRuntime{}, "", "", fmt.Errorf("relation model %q is not declared", relation.toModel)
		}
		if _, alreadyJoined := joined[relation.toModel]; alreadyJoined {
			return "", dataModelRuntime{}, "", "", fmt.Errorf("model %q is already joined", relation.toModel)
		}
		return alias, right, relation.toField, relation.fromField, nil
	}
	if alias, ok := joined[relation.toModel]; ok {
		right, ok := models[relation.fromModel]
		if !ok {
			return "", dataModelRuntime{}, "", "", fmt.Errorf("relation model %q is not declared", relation.fromModel)
		}
		if _, alreadyJoined := joined[relation.fromModel]; alreadyJoined {
			return "", dataModelRuntime{}, "", "", fmt.Errorf("model %q is already joined", relation.fromModel)
		}
		return alias, right, relation.fromField, relation.toField, nil
	}
	return "", dataModelRuntime{}, "", "", fmt.Errorf("relation %q does not connect to joined models", relation.relation.Name)
}

func compileJoinSelect(models map[string]dataModelRuntime, aliases map[string]string, fields []string) (string, error) {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		modelName, fieldName, err := splitQualifiedField(field)
		if err != nil {
			return "", err
		}
		if err := ensureModelField(models, modelName, fieldName); err != nil {
			return "", err
		}
		alias, ok := aliases[modelName]
		if !ok {
			return "", fmt.Errorf("select model %q is not joined", modelName)
		}
		parts = append(parts, fmt.Sprintf("%s.%s AS %s", quoteIdent(alias), quoteIdent(fieldName), quoteIdent(modelName+"_"+fieldName)))
	}
	return strings.Join(parts, ", "), nil
}

func compileJoinWhere(models map[string]dataModelRuntime, aliases map[string]string, conditions []DataCondition) (string, []any, error) {
	if len(conditions) == 0 {
		return "", nil, nil
	}
	parts := make([]string, 0, len(conditions))
	args := make([]any, 0, len(conditions))
	for _, condition := range conditions {
		modelName, fieldName, err := splitQualifiedField(condition.Field)
		if err != nil {
			return "", nil, err
		}
		if err := ensureModelField(models, modelName, fieldName); err != nil {
			return "", nil, err
		}
		alias, ok := aliases[modelName]
		if !ok {
			return "", nil, fmt.Errorf("where model %q is not joined", modelName)
		}
		field := models[modelName].fields[fieldName]
		op := strings.TrimSpace(condition.Op)
		if op == "" {
			op = "eq"
		}
		sqlOp, values, err := compileDataOperator(field, op, condition.Value)
		if err != nil {
			return "", nil, err
		}
		parts = append(parts, fmt.Sprintf("%s.%s %s", quoteIdent(alias), quoteIdent(fieldName), sqlOp))
		args = append(args, values...)
	}
	return " WHERE " + strings.Join(parts, " AND "), args, nil
}

func compileJoinOrder(models map[string]dataModelRuntime, aliases map[string]string, orders []DataOrder) (string, error) {
	if len(orders) == 0 {
		return "", nil
	}
	parts := make([]string, 0, len(orders))
	for _, order := range orders {
		modelName, fieldName, err := splitQualifiedField(order.Field)
		if err != nil {
			return "", err
		}
		if err := ensureModelField(models, modelName, fieldName); err != nil {
			return "", err
		}
		alias, ok := aliases[modelName]
		if !ok {
			return "", fmt.Errorf("order model %q is not joined", modelName)
		}
		direction := strings.ToUpper(strings.TrimSpace(order.Direction))
		if direction != "ASC" && direction != "DESC" {
			direction = "ASC"
		}
		parts = append(parts, quoteIdent(alias)+"."+quoteIdent(fieldName)+" "+direction)
	}
	return " ORDER BY " + strings.Join(parts, ", "), nil
}

func splitQualifiedField(value string) (string, string, error) {
	parts := strings.Split(strings.TrimSpace(value), ".")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("field %q must be qualified as model.field", value)
	}
	modelName := strings.TrimSpace(parts[0])
	fieldName := strings.TrimSpace(parts[1])
	if !dataIdentifierPattern.MatchString(modelName) || !dataIdentifierPattern.MatchString(fieldName) {
		return "", "", fmt.Errorf("invalid qualified field %q", value)
	}
	return modelName, fieldName, nil
}

func ensureModelField(models map[string]dataModelRuntime, modelName string, fieldName string) error {
	model, ok := models[modelName]
	if !ok {
		return fmt.Errorf("model %q is not declared", modelName)
	}
	if _, ok := model.fields[fieldName]; !ok {
		return fmt.Errorf("field %q is not declared on model %q", fieldName, modelName)
	}
	return nil
}

func dataModelDDL(model dataModelRuntime, dialect string) (string, error) {
	ddls, err := dataModelDDLs(model, dialect)
	if err != nil {
		return "", err
	}
	return ddls[0], nil
}

func dataModelDDLs(model dataModelRuntime, dialect string) ([]string, error) {
	columns := []string{
		dataPrimaryKeyColumn(dialect),
		quoteIdent("uuid") + " VARCHAR(120) NOT NULL UNIQUE",
		quoteIdent("created_at") + " " + dataIntegerType(dialect) + " NOT NULL DEFAULT 0",
		quoteIdent("updated_at") + " " + dataIntegerType(dialect) + " NOT NULL DEFAULT 0",
	}
	fieldNames := sortedDataFieldNames(model.fields)
	for _, name := range fieldNames {
		if isSystemDataField(name) {
			continue
		}
		field := model.fields[name]
		columnSQL, err := dataColumnDefinition(field, false, dialect)
		if err != nil {
			return nil, err
		}
		columns = append(columns, columnSQL)
	}
	for _, index := range model.model.Indexes {
		index = strings.TrimSpace(index)
		if index == "" || isSystemDataField(index) {
			continue
		}
		if _, ok := model.fields[index]; !ok {
			return nil, fmt.Errorf("index field %q is not declared", index)
		}
		if !isSQLiteDialect(dialect) {
			columns = append(columns, fmt.Sprintf("INDEX %s (%s)", quoteIdent("idx_"+index), quoteIdent(index)))
		}
	}
	statements := []string{fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", quoteIdent(model.tableName), strings.Join(columns, ", "))}
	if isSQLiteDialect(dialect) {
		for _, index := range model.model.Indexes {
			index = strings.TrimSpace(index)
			if index == "" || isSystemDataField(index) {
				continue
			}
			statements = append(statements, dataCreateIndexSQL(dialect, model.tableName, "idx_"+index, index))
		}
	}
	return statements, nil
}

func dataColumnDefinition(field DataField, forceNullable bool, dialect string) (string, error) {
	columnType, err := dataColumnType(field, dialect)
	if err != nil {
		return "", err
	}
	nullable := " NULL"
	if field.Required && !forceNullable {
		nullable = " NOT NULL"
	}
	return fmt.Sprintf("%s %s%s", quoteIdent(field.Name), columnType, nullable), nil
}

func dataColumnType(field DataField, dialect string) (string, error) {
	switch normalizeDataFieldType(field.Type) {
	case "string", "enum":
		length := field.MaxLength
		if length <= 0 || length > 1000 {
			length = 255
		}
		return fmt.Sprintf("VARCHAR(%d)", length), nil
	case "text":
		return "TEXT", nil
	case "int", "id":
		return dataIntegerType(dialect), nil
	case "float":
		if isSQLiteDialect(dialect) {
			return "REAL", nil
		}
		return "DOUBLE", nil
	case "bool":
		if isSQLiteDialect(dialect) {
			return "INTEGER", nil
		}
		return "TINYINT(1)", nil
	case "datetime":
		return dataIntegerType(dialect), nil
	default:
		return "", fmt.Errorf("unsupported data field type %q", field.Type)
	}
}

func dataPrimaryKeyColumn(dialect string) string {
	if isSQLiteDialect(dialect) {
		return quoteIdent("id") + " INTEGER PRIMARY KEY AUTOINCREMENT"
	}
	return quoteIdent("id") + " BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY"
}

func dataIntegerType(dialect string) string {
	if isSQLiteDialect(dialect) {
		return "INTEGER"
	}
	return "BIGINT"
}

func dataCreateIndexSQL(dialect string, tableName string, indexName string, columnName string) string {
	if isSQLiteDialect(dialect) {
		return fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)", quoteIdent(indexName), quoteIdent(tableName), quoteIdent(columnName))
	}
	return fmt.Sprintf("ALTER TABLE %s ADD INDEX %s (%s)", quoteIdent(tableName), quoteIdent(indexName), quoteIdent(columnName))
}

func isSQLiteDialect(dialect string) bool {
	return dialect == "" || dialect == gormutil.DialectSQLite
}

func validateDataRecord(model dataModelRuntime, input map[string]any, partial bool) (map[string]any, error) {
	if input == nil {
		return nil, errors.New("record is required")
	}
	out := make(map[string]any)
	for name, field := range model.fields {
		if isSystemDataField(name) {
			continue
		}
		value, exists := input[name]
		if !exists {
			if field.Required && !partial {
				return nil, fmt.Errorf("field %q is required", name)
			}
			continue
		}
		if err := validateDataValue(field, value); err != nil {
			return nil, err
		}
		out[name] = value
	}
	for name := range input {
		if _, ok := model.fields[name]; !ok {
			return nil, fmt.Errorf("field %q is not declared", name)
		}
		if isSystemDataField(name) {
			return nil, fmt.Errorf("field %q is managed by platform", name)
		}
	}
	return out, nil
}

func validateDataValue(field DataField, value any) error {
	if value == nil {
		if field.Required {
			return fmt.Errorf("field %q is required", field.Name)
		}
		return nil
	}
	fieldType := normalizeDataFieldType(field.Type)
	switch fieldType {
	case "string", "text", "enum":
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("field %q must be string", field.Name)
		}
		if field.MaxLength > 0 && len([]rune(text)) > field.MaxLength {
			return fmt.Errorf("field %q exceeds max length %d", field.Name, field.MaxLength)
		}
		if fieldType == "enum" && len(field.Values) > 0 && !stringInSlice(text, field.Values) {
			return fmt.Errorf("field %q value is not allowed", field.Name)
		}
	case "int", "id", "float", "datetime":
		number, ok := numberToFloat(value)
		if !ok {
			return fmt.Errorf("field %q must be number", field.Name)
		}
		if field.Min != nil && number < *field.Min {
			return fmt.Errorf("field %q must be >= %v", field.Name, *field.Min)
		}
		if field.Max != nil && number > *field.Max {
			return fmt.Errorf("field %q must be <= %v", field.Name, *field.Max)
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field %q must be bool", field.Name)
		}
	}
	return nil
}

func compileDataWhere(model dataModelRuntime, conditions []DataCondition) (string, []any, error) {
	if len(conditions) == 0 {
		return "", nil, nil
	}
	parts := make([]string, 0, len(conditions))
	args := make([]any, 0, len(conditions))
	for _, condition := range conditions {
		field, ok := model.fields[condition.Field]
		if !ok {
			return "", nil, fmt.Errorf("where field %q is not declared", condition.Field)
		}
		op := strings.TrimSpace(condition.Op)
		if op == "" {
			op = "eq"
		}
		sqlOp, value, err := compileDataOperator(field, op, condition.Value)
		if err != nil {
			return "", nil, err
		}
		parts = append(parts, fmt.Sprintf("%s %s", quoteIdent(condition.Field), sqlOp))
		args = append(args, value...)
	}
	return " WHERE " + strings.Join(parts, " AND "), args, nil
}

func compileDataOperator(field DataField, op string, value any) (string, []any, error) {
	switch op {
	case "eq":
		return "= ?", []any{value}, nil
	case "ne":
		return "<> ?", []any{value}, nil
	case "gt":
		return "> ?", []any{value}, nil
	case "gte":
		return ">= ?", []any{value}, nil
	case "lt":
		return "< ?", []any{value}, nil
	case "lte":
		return "<= ?", []any{value}, nil
	case "contains":
		if normalizeDataFieldType(field.Type) != "string" && normalizeDataFieldType(field.Type) != "text" {
			return "", nil, fmt.Errorf("field %q does not support contains", field.Name)
		}
		return "LIKE ?", []any{"%" + fmt.Sprint(value) + "%"}, nil
	default:
		return "", nil, fmt.Errorf("unsupported operator %q", op)
	}
}

func compileDataOrder(model dataModelRuntime, orders []DataOrder) (string, error) {
	if len(orders) == 0 {
		return " ORDER BY `id` DESC", nil
	}
	parts := make([]string, 0, len(orders))
	for _, order := range orders {
		if _, ok := model.fields[order.Field]; !ok {
			return "", fmt.Errorf("order field %q is not declared", order.Field)
		}
		direction := strings.ToUpper(strings.TrimSpace(order.Direction))
		if direction != "ASC" && direction != "DESC" {
			direction = "ASC"
		}
		parts = append(parts, quoteIdent(order.Field)+" "+direction)
	}
	return " ORDER BY " + strings.Join(parts, ", "), nil
}

func dataTableName(appID string, modelName string) string {
	sum := sha1.Sum([]byte(appID))
	return "func_app_" + hex.EncodeToString(sum[:])[:12] + "__" + strings.ToLower(modelName)
}

func quoteIdent(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

func quoteIdentList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, quoteIdent(value))
	}
	return strings.Join(quoted, ", ")
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedDataFieldNames(values map[string]DataField) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func normalizeDataLimit(limit int) int {
	if limit <= 0 {
		return defaultDataLimit
	}
	if limit > maxDataLimit {
		return maxDataLimit
	}
	return limit
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func normalizeDataFieldType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "integer" {
		return "int"
	}
	if value == "number" {
		return "float"
	}
	return value
}

func isSystemDataField(name string) bool {
	switch name {
	case "id", "uuid", "created_at", "updated_at":
		return true
	default:
		return false
	}
}

func stringInSlice(value string, values []string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func numberToFloat(value any) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float64:
		return typed, true
	case json.Number:
		number, err := typed.Float64()
		return number, err == nil
	default:
		return 0, false
	}
}
