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
}

const (
	ModelBook       = "Book"
	QueryBookList   = "book_list"
	QueryBookByISBN = "book_by_isbn"

	StatusOnshelf  = "onshelf"
	StatusLent     = "lent"
	StatusOffshelf = "offshelf"
)

var categoryValues = []string{"文学", "科技", "历史", "艺术", "经济", "教育", "生活", "其他", "未分类"}

// Book 是图书的行结构体。后端实际以 map[string]any 行数据返回，
// 该结构体用于文档契约中的字段语义说明。
type Book struct {
	ID             uint64 `json:"id"`
	Title          string `json:"title"`
	ISBN           string `json:"isbn"`
	Author         string `json:"author"`
	Category       string `json:"category"`
	Publisher      string `json:"publisher"`
	PublishYear    *int   `json:"publish_year"`
	Location       string `json:"location"`
	TotalStock     int    `json:"total_stock"`
	BorrowedCount  int    `json:"borrowed_count"`
	AvailableCount int    `json:"available_count"`
	Status         string `json:"status"`
	UpdatedAt      string `json:"updated_at"`
}

// FlexID 兼容「数字」与「字符串」两种 id 入参。
// 前端列表/详情中的 id 多来自 HTML data-* 属性，传递时是字符串；
// 直接反序列化为 uint64 会报 "cannot unmarshal string into Go struct field"。
type FlexID uint64

func (id *FlexID) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		*id = 0
		return nil
	}
	// 数字形式
	var n uint64
	if err := json.Unmarshal(b, &n); err == nil {
		*id = FlexID(n)
		return nil
	}
	// 字符串形式
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

type ListParams struct {
	Keyword  string `json:"keyword"`
	Category string `json:"category"`
	Status   string `json:"status"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

type DetailParams struct {
	ID FlexID `json:"id"`
}

type BookFormParams struct {
	Title       string `json:"title"`
	ISBN        string `json:"isbn"`
	Author      string `json:"author"`
	Category    string `json:"category"`
	Publisher   string `json:"publisher"`
	PublishYear *int   `json:"publish_year"`
	Location    string `json:"location"`
	TotalStock  int    `json:"total_stock"`
}

type UpdateParams struct {
	ID          FlexID `json:"id"`
	Title       string `json:"title"`
	ISBN        string `json:"isbn"`
	Author      string `json:"author"`
	Category    string `json:"category"`
	Publisher   string `json:"publisher"`
	PublishYear *int   `json:"publish_year"`
	Location    string `json:"location"`
	TotalStock  int    `json:"total_stock"`
}

type QuantityParams struct {
	ID       FlexID `json:"id"`
	Quantity int    `json:"quantity"`
}

type DeleteParams struct {
	ID FlexID `json:"id"`
}
