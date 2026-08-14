//go:build wasip1

package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type InvokeRequest struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data,omitempty"`
}

type InvokeResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Data    any    `json:"data,omitempty"`
	Rows    any    `json:"rows,omitempty"`
	Total   int    `json:"total,omitempty"`

	// DepartmentOptions / TitleOptions 为列表接口附带返回的筛选项来源。
	DepartmentOptions []string `json:"departmentOptions,omitempty"`
	TitleOptions      []string `json:"titleOptions,omitempty"`
}

const (
	ModelTeacher = "Teacher"

	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusResigned  = "resigned"

	GenderMale   = "male"
	GenderFemale = "female"

	EducationBachelor = "bachelor"
	EducationMaster   = "master"
	EducationDoctor   = "doctor"
	EducationOther    = "other"
)

var (
	statusValues   = []string{StatusActive, StatusSuspended, StatusResigned}
	genderValues   = []string{GenderMale, GenderFemale}
	educationValues = []string{EducationBachelor, EducationMaster, EducationDoctor, EducationOther}
)

// Teacher 是教师行结构体。后端实际以 map[string]any 行数据返回，
// 该结构体用于字段语义说明。
type Teacher struct {
	ID         uint64 `json:"id"`
	Name       string `json:"name"`
	EmployeeNo string `json:"employee_no"`
	Country    string `json:"country"`
	Gender     string `json:"gender"`
	BirthDate  string `json:"birth_date"`
	Education  string `json:"education"`
	Department string `json:"department"`
	Title      string `json:"title"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	HireDate   string `json:"hire_date"`
	Status     string `json:"status"`
	UpdatedAt  string `json:"updated_at"`
}

// FlexID 兼容「数字」与「字符串」两种 id 入参。
type FlexID uint64

func (id *FlexID) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		*id = 0
		return nil
	}
	var n uint64
	if err := json.Unmarshal(b, &n); err == nil {
		*id = FlexID(n)
		return nil
	}
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return fmt.Errorf("id 类型不受支持")
	}
	str = strings.TrimSpace(str)
	if str == "" {
		*id = 0
		return nil
	}
	n2, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return fmt.Errorf("无效的 id: %q", str)
	}
	*id = FlexID(n2)
	return nil
}

type TeacherListParams struct {
	Keyword    string `json:"keyword"`
	Country    string `json:"country"`
	Department string `json:"department"`
	Status     string `json:"status"`
	Education  string `json:"education"`
	Title      string `json:"title"`
	Page       int    `json:"page"`
	PageSize   int    `json:"pageSize"`
}

type DetailParams struct {
	ID FlexID `json:"id"`
}

type TeacherFormParams struct {
	Name       string `json:"name"`
	EmployeeNo string `json:"employee_no"`
	Country    string `json:"country"`
	Gender     string `json:"gender"`
	BirthDate  string `json:"birth_date"`
	Education  string `json:"education"`
	Department string `json:"department"`
	Title      string `json:"title"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	HireDate   string `json:"hire_date"`
}

type TeacherUpdateParams struct {
	ID         FlexID `json:"id"`
	Name       string `json:"name"`
	EmployeeNo string `json:"employee_no"`
	Country    string `json:"country"`
	Gender     string `json:"gender"`
	BirthDate  string `json:"birth_date"`
	Education  string `json:"education"`
	Department string `json:"department"`
	Title      string `json:"title"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	HireDate   string `json:"hire_date"`
}

type ChangeStatusParams struct {
	ID     FlexID `json:"id"`
	Status string `json:"status"`
}

type DeleteParams struct {
	ID FlexID `json:"id"`
}
