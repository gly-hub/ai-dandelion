//go:build wasip1

package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ---------- 行数据处理辅助 ----------

func fieldString(row map[string]any, key string) string {
	if v, ok := row[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

func toInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case float32:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case uint64:
		return int(t)
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	case string:
		n := 0
		fmt.Sscanf(t, "%d", &n)
		return n
	}
	return 0
}

func toUint64(v any) uint64 {
	switch t := v.(type) {
	case float64:
		return uint64(t)
	case float32:
		return uint64(t)
	case uint64:
		return t
	case int:
		if t < 0 {
			return 0
		}
		return uint64(t)
	case int64:
		if t < 0 {
			return 0
		}
		return uint64(t)
	case json.Number:
		n, _ := t.Int64()
		if n < 0 {
			return 0
		}
		return uint64(n)
	case string:
		n := uint64(0)
		fmt.Sscanf(t, "%d", &n)
		return n
	}
	return 0
}

// availableCount 返回可借数量 = 总库存 - 借出数量。
func availableCount(row map[string]any) int {
	total := toInt(row["total_stock"])
	borrowed := toInt(row["borrowed_count"])
	if borrowed < 0 {
		borrowed = 0
	}
	available := total - borrowed
	if available < 0 {
		return 0
	}
	return available
}

// computeStatus 按文档规则计算展示状态：下架状态下不可借出保持下架；
// 否则可借数量为 0 时为借出，否则为在馆。
func computeStatus(row map[string]any) string {
	status := fieldString(row, "status")
	if status == StatusOffshelf {
		return StatusOffshelf
	}
	if availableCount(row) <= 0 {
		return StatusLent
	}
	return StatusOnshelf
}

// normalizeRow 在响应中补齐派生值 available_count 与 status。
func normalizeRow(row map[string]any) map[string]any {
	if row == nil {
		return row
	}
	row["available_count"] = availableCount(row)
	row["status"] = computeStatus(row)
	return row
}

func rowMatchesKeyword(row map[string]any, keyword string) bool {
	for _, key := range []string{"title", "isbn", "author"} {
		if strings.Contains(strings.ToLower(fieldString(row, key)), keyword) {
			return true
		}
	}
	return false
}

// checkISBNUnique 全库校验 ISBN 唯一（排除自身 id）。
func checkISBNUnique(isbn string, excludeID uint64) (bool, string) {
	result := dataList(DataListRequest{
		Model: ModelBook,
		Where: []DataWhere{
			{Field: "isbn", Op: "eq", Value: isbn},
		},
		Page: DataPage{Limit: 100},
	})
	if result.Error != "" {
		return false, result.Error
	}
	for _, row := range result.Rows {
		if toUint64(row["id"]) != excludeID {
			return false, "ISBN 已存在"
		}
	}
	return true, ""
}

// fetchByISBN 按 ISBN 取回完整记录（用于新建后回读）。
func fetchByISBN(isbn string) (map[string]any, bool) {
	result := dataList(DataListRequest{
		Model: ModelBook,
		Where: []DataWhere{
			{Field: "isbn", Op: "eq", Value: isbn},
		},
		Page: DataPage{Limit: 1},
	})
	if result.Error != "" || len(result.Rows) == 0 {
		return nil, false
	}
	return result.Rows[0], true
}

// ---------- action handlers ----------

func handleList(raw []byte) uint64 {
	var params ListParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return storeResponse(InvokeResponse{Error: "参数解析失败: " + err.Error()})
		}
	}
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 10
	}

	var where []DataWhere
	if params.Category != "" {
		where = append(where, DataWhere{Field: "category", Op: "eq", Value: params.Category})
	}
	if params.Status != "" {
		where = append(where, DataWhere{Field: "status", Op: "eq", Value: params.Status})
	}

	result := dataList(DataListRequest{
		Model: ModelBook,
		Where: where,
		OrderBy: []DataOrder{
			{Field: "updated_at", Direction: "desc"},
		},
		Page: DataPage{Limit: 100},
	})
	if result.Error != "" {
		return storeResponse(InvokeResponse{
			Success: false,
			Error:   "图书列表加载失败: " + result.Error,
		})
	}

	keyword := strings.ToLower(strings.TrimSpace(params.Keyword))
	filtered := make([]map[string]any, 0, len(result.Rows))
	for _, row := range result.Rows {
		if keyword != "" && !rowMatchesKeyword(row, keyword) {
			continue
		}
		filtered = append(filtered, normalizeRow(row))
	}

	total := len(filtered)
	offset := (params.Page - 1) * params.PageSize
	end := offset + params.PageSize
	var pageRows []map[string]any
	if offset >= total {
		pageRows = []map[string]any{}
	} else if end >= total {
		pageRows = filtered[offset:]
	} else {
		pageRows = filtered[offset:end]
	}

	return storeResponse(InvokeResponse{
		Success: true,
		Rows:    pageRows,
		Total:   total,
	})
}

func handleDetail(raw []byte) uint64 {
	var params DetailParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return storeResponse(InvokeResponse{Error: "参数解析失败: " + err.Error()})
		}
	}
	if err := validateID(uint64(params.ID)); err != nil {
		return storeResponse(InvokeResponse{Success: false, Error: err.Message})
	}

	result := dataGet(ModelBook, uint64(params.ID))
	if result.Error != "" {
		return storeResponse(InvokeResponse{
			Success: false,
			Error:   "加载图书详情失败: " + result.Error,
		})
	}
	if result.Record == nil {
		return storeResponse(InvokeResponse{Success: false, Error: "记录不存在"})
	}
	return storeResponse(InvokeResponse{
		Success: true,
		Data:    normalizeRow(result.Record),
	})
}

func handleCreate(raw []byte) uint64 {
	var params BookFormParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return storeResponse(InvokeResponse{Error: "参数解析失败: " + err.Error()})
		}
	}

	errors := validateBookForm(params.Title, params.ISBN, params.Author, params.Category, params.Publisher, params.Location, params.PublishYear, params.TotalStock)
	if len(errors) > 0 {
		return storeResponse(InvokeResponse{Success: false, Error: errors[0].Message})
	}

	isbn := normalizeISBN(params.ISBN)
	if unique, msg := checkISBNUnique(isbn, 0); !unique {
		return storeResponse(InvokeResponse{Success: false, Error: msg})
	}

	category := params.Category
	if category == "" {
		category = "未分类"
	}

	record := map[string]any{
		"title":          strings.TrimSpace(params.Title),
		"isbn":           isbn,
		"author":         strings.TrimSpace(params.Author),
		"category":       category,
		"publisher":      strings.TrimSpace(params.Publisher),
		"location":       strings.TrimSpace(params.Location),
		"total_stock":    params.TotalStock,
		"borrowed_count": 0,
		"status":         StatusOnshelf,
	}
	if params.PublishYear != nil {
		record["publish_year"] = *params.PublishYear
	}

	result := dataCreate(DataWriteRequest{Model: ModelBook, Record: record})
	if result.Error != "" {
		return storeResponse(InvokeResponse{
			Success: false,
			Error:   "新建图书失败: " + result.Error,
		})
	}

	if row, ok := fetchByISBN(isbn); ok {
		return storeResponse(InvokeResponse{Success: true, Data: normalizeRow(row)})
	}
	return storeResponse(InvokeResponse{Success: true, Data: record})
}

func handleUpdate(raw []byte) uint64 {
	var params UpdateParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return storeResponse(InvokeResponse{Error: "参数解析失败: " + err.Error()})
		}
	}
	if err := validateID(uint64(params.ID)); err != nil {
		return storeResponse(InvokeResponse{Success: false, Error: err.Message})
	}

	existing := dataGet(ModelBook, uint64(params.ID))
	if existing.Error != "" || existing.Record == nil {
		return storeResponse(InvokeResponse{Success: false, Error: "记录不存在"})
	}

	borrowed := toInt(existing.Record["borrowed_count"])
	if params.TotalStock < borrowed {
		return storeResponse(InvokeResponse{Success: false, Error: "总库存不能小于已借出数量"})
	}

	errors := validateBookForm(params.Title, params.ISBN, params.Author, params.Category, params.Publisher, params.Location, params.PublishYear, params.TotalStock)
	if len(errors) > 0 {
		return storeResponse(InvokeResponse{Success: false, Error: errors[0].Message})
	}

	isbn := normalizeISBN(params.ISBN)
	if unique, msg := checkISBNUnique(isbn, uint64(params.ID)); !unique {
		return storeResponse(InvokeResponse{Success: false, Error: msg})
	}

	category := params.Category
	if category == "" {
		category = "未分类"
	}

	patch := map[string]any{
		"title":       strings.TrimSpace(params.Title),
		"isbn":        isbn,
		"author":      strings.TrimSpace(params.Author),
		"category":    category,
		"publisher":   strings.TrimSpace(params.Publisher),
		"location":    strings.TrimSpace(params.Location),
		"total_stock": params.TotalStock,
	}
	if params.PublishYear != nil {
		patch["publish_year"] = *params.PublishYear
	}

	if fieldString(existing.Record, "status") != StatusOffshelf {
		available := params.TotalStock - borrowed
		if available <= 0 {
			patch["status"] = StatusLent
		} else {
			patch["status"] = StatusOnshelf
		}
	}

	result := dataUpdate(DataWriteRequest{Model: ModelBook, ID: uint64(params.ID), Patch: patch})
	if result.Error != "" {
		return storeResponse(InvokeResponse{
			Success: false,
			Error:   "更新图书失败: " + result.Error,
		})
	}

	getResult := dataGet(ModelBook, uint64(params.ID))
	if getResult.Error == "" && getResult.Record != nil {
		return storeResponse(InvokeResponse{Success: true, Data: normalizeRow(getResult.Record)})
	}
	return storeResponse(InvokeResponse{Success: true, Data: patch})
}

func handleDelete(raw []byte) uint64 {
	var params DeleteParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return storeResponse(InvokeResponse{Error: "参数解析失败: " + err.Error()})
		}
	}
	if err := validateID(uint64(params.ID)); err != nil {
		return storeResponse(InvokeResponse{Success: false, Error: err.Message})
	}

	existing := dataGet(ModelBook, uint64(params.ID))
	if existing.Error != "" || existing.Record == nil {
		return storeResponse(InvokeResponse{Success: false, Error: "记录不存在"})
	}
	if toInt(existing.Record["borrowed_count"]) > 0 {
		return storeResponse(InvokeResponse{Success: false, Error: "存在借出记录，请先归还"})
	}

	result := dataDelete(ModelBook, uint64(params.ID))
	if result.Error != "" {
		return storeResponse(InvokeResponse{
			Success: false,
			Error:   "删除图书失败: " + result.Error,
		})
	}
	return storeResponse(InvokeResponse{Success: true, Data: map[string]any{}})
}

func handleLend(raw []byte) uint64 {
	var params QuantityParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return storeResponse(InvokeResponse{Error: "参数解析失败: " + err.Error()})
		}
	}
	if err := validateID(uint64(params.ID)); err != nil {
		return storeResponse(InvokeResponse{Success: false, Error: err.Message})
	}
	if params.Quantity < 1 {
		return storeResponse(InvokeResponse{Success: false, Error: "请输入不小于 1 的整数"})
	}

	existing := dataGet(ModelBook, uint64(params.ID))
	if existing.Error != "" || existing.Record == nil {
		return storeResponse(InvokeResponse{Success: false, Error: "记录不存在"})
	}
	if fieldString(existing.Record, "status") == StatusOffshelf {
		return storeResponse(InvokeResponse{Success: false, Error: "该图书已下架，不能借出"})
	}

	total := toInt(existing.Record["total_stock"])
	borrowed := toInt(existing.Record["borrowed_count"])
	available := total - borrowed
	if params.Quantity > available {
		return storeResponse(InvokeResponse{Success: false, Error: "超出可借数量"})
	}

	newBorrowed := borrowed + params.Quantity
	status := StatusOnshelf
	if total-newBorrowed <= 0 {
		status = StatusLent
	}

	result := dataUpdate(DataWriteRequest{
		Model: ModelBook,
		ID:    uint64(params.ID),
		Patch: map[string]any{"borrowed_count": newBorrowed, "status": status},
	})
	if result.Error != "" {
		return storeResponse(InvokeResponse{
			Success: false,
			Error:   "借出失败: " + result.Error,
		})
	}

	getResult := dataGet(ModelBook, uint64(params.ID))
	if getResult.Error == "" && getResult.Record != nil {
		return storeResponse(InvokeResponse{Success: true, Data: normalizeRow(getResult.Record)})
	}
	return storeResponse(InvokeResponse{Success: true, Data: map[string]any{}})
}

func handleReturnBook(raw []byte) uint64 {
	var params QuantityParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return storeResponse(InvokeResponse{Error: "参数解析失败: " + err.Error()})
		}
	}
	if err := validateID(uint64(params.ID)); err != nil {
		return storeResponse(InvokeResponse{Success: false, Error: err.Message})
	}
	if params.Quantity < 1 {
		return storeResponse(InvokeResponse{Success: false, Error: "请输入不小于 1 的整数"})
	}

	existing := dataGet(ModelBook, uint64(params.ID))
	if existing.Error != "" || existing.Record == nil {
		return storeResponse(InvokeResponse{Success: false, Error: "记录不存在"})
	}

	borrowed := toInt(existing.Record["borrowed_count"])
	if params.Quantity > borrowed {
		return storeResponse(InvokeResponse{Success: false, Error: "归还数量大于借出数量"})
	}

	newBorrowed := borrowed - params.Quantity
	status := fieldString(existing.Record, "status")
	if status != StatusOffshelf {
		total := toInt(existing.Record["total_stock"])
		if total-newBorrowed > 0 {
			status = StatusOnshelf
		} else {
			status = StatusLent
		}
	}

	result := dataUpdate(DataWriteRequest{
		Model: ModelBook,
		ID:    uint64(params.ID),
		Patch: map[string]any{"borrowed_count": newBorrowed, "status": status},
	})
	if result.Error != "" {
		return storeResponse(InvokeResponse{
			Success: false,
			Error:   "归还失败: " + result.Error,
		})
	}

	getResult := dataGet(ModelBook, uint64(params.ID))
	if getResult.Error == "" && getResult.Record != nil {
		return storeResponse(InvokeResponse{Success: true, Data: normalizeRow(getResult.Record)})
	}
	return storeResponse(InvokeResponse{Success: true, Data: map[string]any{}})
}

func handleOffshelf(raw []byte) uint64 {
	var params DetailParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return storeResponse(InvokeResponse{Error: "参数解析失败: " + err.Error()})
		}
	}
	if err := validateID(uint64(params.ID)); err != nil {
		return storeResponse(InvokeResponse{Success: false, Error: err.Message})
	}

	existing := dataGet(ModelBook, uint64(params.ID))
	if existing.Error != "" || existing.Record == nil {
		return storeResponse(InvokeResponse{Success: false, Error: "记录不存在"})
	}
	if fieldString(existing.Record, "status") == StatusOffshelf {
		return storeResponse(InvokeResponse{Success: false, Error: "已是下架状态"})
	}

	result := dataUpdate(DataWriteRequest{
		Model: ModelBook,
		ID:    uint64(params.ID),
		Patch: map[string]any{"status": StatusOffshelf},
	})
	if result.Error != "" {
		return storeResponse(InvokeResponse{
			Success: false,
			Error:   "下架失败: " + result.Error,
		})
	}

	getResult := dataGet(ModelBook, uint64(params.ID))
	if getResult.Error == "" && getResult.Record != nil {
		return storeResponse(InvokeResponse{Success: true, Data: normalizeRow(getResult.Record)})
	}
	return storeResponse(InvokeResponse{Success: true, Data: map[string]any{}})
}

func handleOnshelf(raw []byte) uint64 {
	var params DetailParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return storeResponse(InvokeResponse{Error: "参数解析失败: " + err.Error()})
		}
	}
	if err := validateID(uint64(params.ID)); err != nil {
		return storeResponse(InvokeResponse{Success: false, Error: err.Message})
	}

	existing := dataGet(ModelBook, uint64(params.ID))
	if existing.Error != "" || existing.Record == nil {
		return storeResponse(InvokeResponse{Success: false, Error: "记录不存在"})
	}
	if fieldString(existing.Record, "status") != StatusOffshelf {
		return storeResponse(InvokeResponse{Success: false, Error: "非下架状态，无需上架"})
	}

	total := toInt(existing.Record["total_stock"])
	borrowed := toInt(existing.Record["borrowed_count"])
	status := StatusOnshelf
	if total-borrowed <= 0 {
		status = StatusLent
	}

	result := dataUpdate(DataWriteRequest{
		Model: ModelBook,
		ID:    uint64(params.ID),
		Patch: map[string]any{"status": status},
	})
	if result.Error != "" {
		return storeResponse(InvokeResponse{
			Success: false,
			Error:   "上架失败: " + result.Error,
		})
	}

	getResult := dataGet(ModelBook, uint64(params.ID))
	if getResult.Error == "" && getResult.Record != nil {
		return storeResponse(InvokeResponse{Success: true, Data: normalizeRow(getResult.Record)})
	}
	return storeResponse(InvokeResponse{Success: true, Data: map[string]any{}})
}
