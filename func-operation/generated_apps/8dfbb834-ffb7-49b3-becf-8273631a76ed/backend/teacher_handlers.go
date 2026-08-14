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

func rowMatchesKeyword(row map[string]any, keyword string) bool {
	for _, key := range []string{"name", "employee_no", "phone"} {
		if strings.Contains(strings.ToLower(fieldString(row, key)), keyword) {
			return true
		}
	}
	return false
}

// checkEmployeeNoUnique 全库校验工号唯一（排除自身 id）。
func checkEmployeeNoUnique(employeeNo string, excludeID uint64) (bool, string) {
	result := dataList(DataListRequest{
		Model: ModelTeacher,
		Where: []DataWhere{
			{Field: "employee_no", Op: "eq", Value: employeeNo},
		},
		Page: DataPage{Limit: 100},
	})
	if result.Error != "" {
		return false, "工号唯一性检查失败: " + result.Error
	}
	for _, row := range result.Rows {
		if toUint64(row["id"]) != excludeID {
			return false, "工号已存在，请检查"
		}
	}
	return true, ""
}

// fetchByEmployeeNo 按工号取回完整记录（用于新建后回读）。
func fetchByEmployeeNo(employeeNo string) (map[string]any, bool) {
	result := dataList(DataListRequest{
		Model: ModelTeacher,
		Where: []DataWhere{
			{Field: "employee_no", Op: "eq", Value: employeeNo},
		},
		Page: DataPage{Limit: 1},
	})
	if result.Error != "" || len(result.Rows) == 0 {
		return nil, false
	}
	return result.Rows[0], true
}

// ---------- action handlers ----------

func handleTeacherList(raw []byte) uint64 {
	var params TeacherListParams
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
	if params.Country != "" {
		where = append(where, DataWhere{Field: "country", Op: "eq", Value: params.Country})
	}
	if params.Department != "" {
		where = append(where, DataWhere{Field: "department", Op: "eq", Value: params.Department})
	}
	if params.Status != "" {
		where = append(where, DataWhere{Field: "status", Op: "eq", Value: params.Status})
	}
	if params.Education != "" {
		where = append(where, DataWhere{Field: "education", Op: "eq", Value: params.Education})
	}
	if params.Title != "" {
		where = append(where, DataWhere{Field: "title", Op: "eq", Value: params.Title})
	}

	result := dataList(DataListRequest{
		Model: ModelTeacher,
		Where: where,
		OrderBy: []DataOrder{
			{Field: "updated_at", Direction: "desc"},
		},
		Page: DataPage{Limit: 100},
	})
	if result.Error != "" {
		return storeResponse(InvokeResponse{
			Success: false,
			Error:   "教师列表加载失败: " + result.Error,
		})
	}

	keyword := strings.ToLower(strings.TrimSpace(params.Keyword))
	filtered := make([]map[string]any, 0, len(result.Rows))
	departmentSet := make(map[string]bool)
	titleSet := make(map[string]bool)
	for _, row := range result.Rows {
		if keyword != "" && !rowMatchesKeyword(row, keyword) {
			continue
		}
		if dep := fieldString(row, "department"); dep != "" {
			departmentSet[dep] = true
		}
		if t := fieldString(row, "title"); t != "" {
			titleSet[t] = true
		}
		filtered = append(filtered, row)
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
		Success:           true,
		Rows:              pageRows,
		Total:             total,
		DepartmentOptions: sortedSetKeys(departmentSet),
		TitleOptions:      sortedSetKeys(titleSet),
	})
}

func handleTeacherDetail(raw []byte) uint64 {
	var params DetailParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return storeResponse(InvokeResponse{Error: "参数解析失败: " + err.Error()})
		}
	}
	if err := validateID(uint64(params.ID)); err != nil {
		return storeResponse(InvokeResponse{Success: false, Error: err.Message})
	}

	result := dataGet(ModelTeacher, uint64(params.ID))
	if result.Error != "" {
		return storeResponse(InvokeResponse{
			Success: false,
			Error:   "加载教师详情失败: " + result.Error,
		})
	}
	if result.Record == nil {
		return storeResponse(InvokeResponse{Success: false, Error: "教师记录不存在"})
	}
	return storeResponse(InvokeResponse{Success: true, Data: result.Record})
}

func handleTeacherCreate(raw []byte) uint64 {
	var params TeacherFormParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return storeResponse(InvokeResponse{Error: "参数解析失败: " + err.Error()})
		}
	}

	errors := validateTeacherForm(params.Name, params.EmployeeNo, params.Country, params.Gender, params.BirthDate, params.Education, params.Department, params.Title, params.Phone, params.Email, params.HireDate)
	if len(errors) > 0 {
		return storeResponse(InvokeResponse{Success: false, Error: errors[0].Message})
	}

	employeeNo := strings.TrimSpace(params.EmployeeNo)
	if unique, msg := checkEmployeeNoUnique(employeeNo, 0); !unique {
		return storeResponse(InvokeResponse{Success: false, Error: msg})
	}

	record := map[string]any{
		"name":        strings.TrimSpace(params.Name),
		"employee_no": employeeNo,
		"gender":      normalizeGender(params.Gender),
		"department":  strings.TrimSpace(params.Department),
		"title":       strings.TrimSpace(params.Title),
		"phone":       strings.TrimSpace(params.Phone),
		"email":       strings.TrimSpace(params.Email),
		"status":      StatusActive,
	}
	if country := strings.TrimSpace(params.Country); country != "" {
		record["country"] = country
	}
	if params.BirthDate != "" {
		record["birth_date"] = params.BirthDate
	}
	if params.Education != "" {
		record["education"] = params.Education
	}
	if params.HireDate != "" {
		record["hire_date"] = params.HireDate
	}

	result := dataCreate(DataWriteRequest{Model: ModelTeacher, Record: record})
	if result.Error != "" {
		return storeResponse(InvokeResponse{
			Success: false,
			Error:   "新建教师失败: " + result.Error,
		})
	}

	if row, ok := fetchByEmployeeNo(employeeNo); ok {
		return storeResponse(InvokeResponse{Success: true, Data: row})
	}
	return storeResponse(InvokeResponse{Success: true, Data: record})
}

func handleTeacherUpdate(raw []byte) uint64 {
	var params TeacherUpdateParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return storeResponse(InvokeResponse{Error: "参数解析失败: " + err.Error()})
		}
	}
	if err := validateID(uint64(params.ID)); err != nil {
		return storeResponse(InvokeResponse{Success: false, Error: err.Message})
	}

	existing := dataGet(ModelTeacher, uint64(params.ID))
	if existing.Error != "" || existing.Record == nil {
		return storeResponse(InvokeResponse{Success: false, Error: "教师记录不存在"})
	}

	errors := validateTeacherForm(params.Name, params.EmployeeNo, params.Country, params.Gender, params.BirthDate, params.Education, params.Department, params.Title, params.Phone, params.Email, params.HireDate)
	if len(errors) > 0 {
		return storeResponse(InvokeResponse{Success: false, Error: errors[0].Message})
	}

	employeeNo := strings.TrimSpace(params.EmployeeNo)
	if unique, msg := checkEmployeeNoUnique(employeeNo, uint64(params.ID)); !unique {
		return storeResponse(InvokeResponse{Success: false, Error: msg})
	}

	patch := map[string]any{
		"name":        strings.TrimSpace(params.Name),
		"employee_no": employeeNo,
		"gender":      normalizeGender(params.Gender),
		"department":  strings.TrimSpace(params.Department),
		"title":       strings.TrimSpace(params.Title),
		"phone":       strings.TrimSpace(params.Phone),
		"email":       strings.TrimSpace(params.Email),
	}
	// 可选字段：为空时置 nil 清空存储值。
	if country := strings.TrimSpace(params.Country); country != "" {
		patch["country"] = country
	} else {
		patch["country"] = nil
	}
	if params.BirthDate != "" {
		patch["birth_date"] = params.BirthDate
	} else {
		patch["birth_date"] = nil
	}
	if params.Education != "" {
		patch["education"] = params.Education
	} else {
		patch["education"] = nil
	}
	if params.HireDate != "" {
		patch["hire_date"] = params.HireDate
	} else {
		patch["hire_date"] = nil
	}

	result := dataUpdate(DataWriteRequest{Model: ModelTeacher, ID: uint64(params.ID), Patch: patch})
	if result.Error != "" {
		return storeResponse(InvokeResponse{
			Success: false,
			Error:   "更新教师失败: " + result.Error,
		})
	}

	getResult := dataGet(ModelTeacher, uint64(params.ID))
	if getResult.Error == "" && getResult.Record != nil {
		return storeResponse(InvokeResponse{Success: true, Data: getResult.Record})
	}
	return storeResponse(InvokeResponse{Success: true, Data: patch})
}

func handleTeacherChangeStatus(raw []byte) uint64 {
	var params ChangeStatusParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return storeResponse(InvokeResponse{Error: "参数解析失败: " + err.Error()})
		}
	}
	if err := validateID(uint64(params.ID)); err != nil {
		return storeResponse(InvokeResponse{Success: false, Error: err.Message})
	}

	target := strings.TrimSpace(params.Status)
	if target == "" || !containsString(statusValues, target) {
		return storeResponse(InvokeResponse{Success: false, Error: "目标状态不合法"})
	}

	existing := dataGet(ModelTeacher, uint64(params.ID))
	if existing.Error != "" || existing.Record == nil {
		return storeResponse(InvokeResponse{Success: false, Error: "教师记录不存在"})
	}
	current := fieldString(existing.Record, "status")
	if current == StatusResigned {
		return storeResponse(InvokeResponse{Success: false, Error: "已离职教师不可再变更状态"})
	}
	if current == target {
		return storeResponse(InvokeResponse{Success: false, Error: "目标状态与当前状态相同"})
	}

	result := dataUpdate(DataWriteRequest{
		Model: ModelTeacher,
		ID:    uint64(params.ID),
		Patch: map[string]any{"status": target},
	})
	if result.Error != "" {
		return storeResponse(InvokeResponse{
			Success: false,
			Error:   "状态流转失败: " + result.Error,
		})
	}

	getResult := dataGet(ModelTeacher, uint64(params.ID))
	if getResult.Error == "" && getResult.Record != nil {
		return storeResponse(InvokeResponse{Success: true, Data: getResult.Record})
	}
	return storeResponse(InvokeResponse{Success: true, Data: map[string]any{"id": params.ID, "status": target}})
}

func handleTeacherDelete(raw []byte) uint64 {
	var params DeleteParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return storeResponse(InvokeResponse{Error: "参数解析失败: " + err.Error()})
		}
	}
	if err := validateID(uint64(params.ID)); err != nil {
		return storeResponse(InvokeResponse{Success: false, Error: err.Message})
	}

	existing := dataGet(ModelTeacher, uint64(params.ID))
	if existing.Error != "" || existing.Record == nil {
		return storeResponse(InvokeResponse{Success: false, Error: "教师记录不存在"})
	}
	current := fieldString(existing.Record, "status")
	if current != StatusResigned {
		if current == StatusActive {
			return storeResponse(InvokeResponse{Success: false, Error: "在职教师不可删除，请先办理离职"})
		}
		return storeResponse(InvokeResponse{Success: false, Error: "停用教师不可删除，请先办理离职"})
	}

	result := dataDelete(ModelTeacher, uint64(params.ID))
	if result.Error != "" {
		return storeResponse(InvokeResponse{
			Success: false,
			Error:   "删除教师失败: " + result.Error,
		})
	}
	return storeResponse(InvokeResponse{Success: true, Data: map[string]any{}})
}

func containsString(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}
