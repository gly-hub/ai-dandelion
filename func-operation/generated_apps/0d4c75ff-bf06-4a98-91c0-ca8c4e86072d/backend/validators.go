//go:build wasip1

package main

import "strings"

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

func normalizeISBN(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "-", "")
	return value
}

func isValidISBN(value string) bool {
	v := normalizeISBN(value)
	if len(v) == 10 {
		return isValidISBN10(v)
	}
	if len(v) == 13 {
		return isValidISBN13(v)
	}
	return false
}

func isValidISBN10(v string) bool {
	sum := 0
	for i := 0; i < 10; i++ {
		c := v[i]
		var d int
		if i == 9 && c == 'X' {
			d = 10
		} else if c >= '0' && c <= '9' {
			d = int(c - '0')
		} else {
			return false
		}
		sum += d * (10 - i)
	}
	return sum%11 == 0
}

func isValidISBN13(v string) bool {
	sum := 0
	for i := 0; i < 13; i++ {
		c := v[i]
		if c < '0' || c > '9' {
			return false
		}
		d := int(c - '0')
		if i%2 == 0 {
			sum += d
		} else {
			sum += d * 3
		}
	}
	return sum%10 == 0
}

func validateBookForm(title, isbn, author, category, publisher, location string, publishYear *int, totalStock int) []ValidationError {
	var errors []ValidationError

	if err := validateRequired(title, "title", "请输入书名"); err != nil {
		errors = append(errors, *err)
	} else if err := validateMaxLength(title, 120, "title", "书名需在 1-120 字之间"); err != nil {
		errors = append(errors, *err)
	}

	if err := validateRequired(isbn, "isbn", "请输入 ISBN"); err != nil {
		errors = append(errors, *err)
	} else if !isValidISBN(isbn) {
		errors = append(errors, ValidationError{Field: "isbn", Message: "ISBN 格式不正确"})
	}

	if err := validateMaxLength(author, 80, "author", "作者需在 80 字以内"); err != nil {
		errors = append(errors, *err)
	}

	if err := validateEnum(category, categoryValues, "category", "请选择有效的分类"); err != nil {
		errors = append(errors, *err)
	}

	if err := validateMaxLength(publisher, 80, "publisher", "出版社需在 80 字以内"); err != nil {
		errors = append(errors, *err)
	}

	if publishYear != nil {
		if *publishYear < 1000 || *publishYear > 2100 {
			errors = append(errors, ValidationError{Field: "publish_year", Message: "出版年份需在 1000-2100 之间"})
		}
	}

	if err := validateMaxLength(location, 60, "location", "馆藏位置需在 60 字以内"); err != nil {
		errors = append(errors, *err)
	}

	if totalStock < 1 {
		errors = append(errors, ValidationError{Field: "total_stock", Message: "总库存需为不小于 1 的整数"})
	}

	return errors
}
