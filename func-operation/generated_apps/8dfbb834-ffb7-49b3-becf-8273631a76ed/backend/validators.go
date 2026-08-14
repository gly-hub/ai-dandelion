//go:build wasip1

package main

import (
	"regexp"
	"strings"
	"time"
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func validateRequired(value, field, message string) *ValidationError {
	if strings.TrimSpace(value) == "" {
		return &ValidationError{Field: field, Message: message}
	}
	return nil
}

func validateMaxLength(value string, maxLen int, field, message string) *ValidationError {
	if len([]rune(value)) > maxLen {
		return &ValidationError{Field: field, Message: message}
	}
	return nil
}

func validateEnum(value string, valid []string, field, message string) *ValidationError {
	if value == "" {
		return nil
	}
	for _, v := range valid {
		if v == value {
			return nil
		}
	}
	return &ValidationError{Field: field, Message: message}
}

func validateID(id uint64) *ValidationError {
	if id <= 0 {
		return &ValidationError{Field: "id", Message: "记录不存在"}
	}
	return nil
}

var employeeNoPattern = regexp.MustCompile(`^[A-Za-z0-9]{3,30}$`)
var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func isValidEmployeeNo(value string) bool {
	return employeeNoPattern.MatchString(value)
}

// isValidPhone 手机/座机：去除空格、横线、括号与 + 后，数字位数在 7-15 位。
func isValidPhone(value string) bool {
	digits := phoneDigits(value)
	return len(digits) >= 7 && len(digits) <= 15
}

func phoneDigits(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isValidEmail(value string) bool {
	return emailPattern.MatchString(value)
}

// isValidDate 校验 yyyy-MM-dd 为真实存在的日期。
func isValidDate(value string) bool {
	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		return false
	}
	return t.Format("2006-01-02") == value
}

// validateTeacherForm 校验新建/编辑教师表单。返回非空即存在校验错误。
func validateTeacherForm(name, employeeNo, country, gender, birthDate, education, department, title, phone, email, hireDate string) []ValidationError {
	var errors []ValidationError

	if err := validateRequired(name, "name", "请输入姓名"); err != nil {
		errors = append(errors, *err)
	} else if err := validateMaxLength(name, 50, "name", "姓名需在 1-50 字之间"); err != nil {
		errors = append(errors, *err)
	}

	if err := validateRequired(employeeNo, "employee_no", "请输入工号"); err != nil {
		errors = append(errors, *err)
	} else if !isValidEmployeeNo(employeeNo) {
		errors = append(errors, ValidationError{Field: "employee_no", Message: "工号需为 3-30 位字母或数字"})
	}

	// 国籍为可选字段；选项来源为全局配置键 country（前端读取），
	// WASM 侧无法访问公共配置，仅做长度边界校验，选项合法性由前端约束。
	if err := validateMaxLength(country, 50, "country", "国籍需在 50 字以内"); err != nil {
		errors = append(errors, *err)
	}

	if err := validateEnum(gender, genderValues, "gender", "请选择正确的性别"); err != nil {
		errors = append(errors, *err)
	}

	if birthDate != "" && !isValidDate(birthDate) {
		errors = append(errors, ValidationError{Field: "birth_date", Message: "请输入正确的出生日期"})
	}

	if err := validateEnum(education, educationValues, "education", "请选择正确的学历"); err != nil {
		errors = append(errors, *err)
	}

	if err := validateRequired(department, "department", "请输入所属院系"); err != nil {
		errors = append(errors, *err)
	} else if err := validateMaxLength(department, 50, "department", "所属院系需在 1-50 字之间"); err != nil {
		errors = append(errors, *err)
	}

	if err := validateMaxLength(title, 50, "title", "职称需在 50 字以内"); err != nil {
		errors = append(errors, *err)
	}

	if err := validateRequired(phone, "phone", "请输入联系电话"); err != nil {
		errors = append(errors, *err)
	} else if !isValidPhone(phone) {
		errors = append(errors, ValidationError{Field: "phone", Message: "请输入正确的联系电话"})
	}

	if email != "" {
		if err := validateMaxLength(email, 100, "email", "邮箱需在 100 字以内"); err != nil {
			errors = append(errors, *err)
		} else if !isValidEmail(email) {
			errors = append(errors, ValidationError{Field: "email", Message: "请输入正确的电子邮箱"})
		}
	}

	if hireDate != "" && !isValidDate(hireDate) {
		errors = append(errors, ValidationError{Field: "hire_date", Message: "请输入正确的入职日期"})
	}

	return errors
}

// normalizeGender 空值回退为 male。
func normalizeGender(value string) string {
	for _, v := range genderValues {
		if v == value {
			return v
		}
	}
	return GenderMale
}
