//go:build wasip1

package main

import (
	"encoding/json"
	"sort"
	"unsafe"
)

//go:wasmimport platform result_store
func resultStore(reqPtr, reqLen uint32) uint64

//go:wasmimport platform data_list
func hostDataList(reqPtr, reqLen uint32) uint64

//go:wasmimport platform data_get
func hostDataGet(reqPtr, reqLen uint32) uint64

//go:wasmimport platform data_create
func hostDataCreate(reqPtr, reqLen uint32) uint64

//go:wasmimport platform data_update
func hostDataUpdate(reqPtr, reqLen uint32) uint64

//go:wasmimport platform data_delete
func hostDataDelete(reqPtr, reqLen uint32) uint64

//go:wasmimport platform data_join_query
func hostDataJoinQuery(reqPtr, reqLen uint32) uint64

//go:wasmimport platform data_run_query
func hostDataRunQuery(reqPtr, reqLen uint32) uint64

//go:wasmimport platform call_capability
func hostCallCapability(reqPtr, reqLen uint32) uint64

//go:wasmimport platform result_len
func resultLen(handle uint64) uint32

//go:wasmimport platform result_read
func resultRead(handle uint64, outPtr uint32) uint32

func storeResponse(data any) uint64 {
	raw, _ := json.Marshal(data)
	return resultStore(uint32(uintptr(unsafe.Pointer(&raw[0]))), uint32(len(raw)))
}

type DataListRequest struct {
	Model   string      `json:"model"`
	Where   []DataWhere `json:"where,omitempty"`
	OrderBy []DataOrder `json:"orderBy,omitempty"`
	Page    DataPage    `json:"page,omitempty"`
}

type DataWhere struct {
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

type DataWriteRequest struct {
	Model  string         `json:"model"`
	ID     any            `json:"id,omitempty"`
	Record map[string]any `json:"record,omitempty"`
	Patch  map[string]any `json:"patch,omitempty"`
}

type DataJoinRequest struct {
	From    string      `json:"from"`
	Joins   []DataJoin  `json:"joins,omitempty"`
	Select  []string    `json:"select"`
	Where   []DataWhere `json:"where,omitempty"`
	OrderBy []DataOrder `json:"orderBy,omitempty"`
	Limit   int         `json:"limit,omitempty"`
}

type DataJoin struct {
	Relation string `json:"relation"`
	Type     string `json:"type,omitempty"`
}

type DataRunQueryRequest struct {
	Query  string         `json:"query"`
	Params map[string]any `json:"params,omitempty"`
}

type CapabilityCallRequest struct {
	AppID      string         `json:"appId"`
	Capability string         `json:"capability"`
	Params     map[string]any `json:"params,omitempty"`
}

type DataListResult struct {
	Rows  []map[string]any `json:"rows"`
	Total int              `json:"total"`
	Error string           `json:"error,omitempty"`
}

type DataGetResult struct {
	Record map[string]any `json:"record,omitempty"`
	Error  string         `json:"error,omitempty"`
}

type DataWriteResult struct {
	ID           any            `json:"id,omitempty"`
	RowsAffected int64          `json:"rowsAffected,omitempty"`
	Record       map[string]any `json:"record,omitempty"`
	Error        string         `json:"error,omitempty"`
}

func dataList(req DataListRequest) DataListResult {
	var result DataListResult
	callPlatformJSON(hostDataList, req, &result)
	return result
}

func dataGet(model string, id any) DataGetResult {
	var result DataGetResult
	callPlatformJSON(hostDataGet, map[string]any{"model": model, "id": id}, &result)
	return result
}

func dataCreate(req DataWriteRequest) DataWriteResult {
	var result DataWriteResult
	callPlatformJSON(hostDataCreate, req, &result)
	return result
}

func dataUpdate(req DataWriteRequest) DataWriteResult {
	var result DataWriteResult
	callPlatformJSON(hostDataUpdate, req, &result)
	return result
}

func dataDelete(model string, id any) DataWriteResult {
	var result DataWriteResult
	callPlatformJSON(hostDataDelete, map[string]any{"model": model, "id": id}, &result)
	return result
}

func dataJoinQuery(req DataJoinRequest) DataListResult {
	var result DataListResult
	callPlatformJSON(hostDataJoinQuery, req, &result)
	return result
}

func dataRunQuery(req DataRunQueryRequest) DataListResult {
	var result DataListResult
	callPlatformJSON(hostDataRunQuery, req, &result)
	return result
}

func callCapability(req CapabilityCallRequest) DataListResult {
	var result DataListResult
	callPlatformJSON(hostCallCapability, req, &result)
	return result
}

func callPlatformJSON(call func(uint32, uint32) uint64, req any, out any) {
	raw, _ := json.Marshal(req)
	handle := call(uint32(uintptr(unsafe.Pointer(&raw[0]))), uint32(len(raw)))
	length := resultLen(handle)
	if length == 0 {
		return
	}
	buf := make([]byte, length)
	resultRead(handle, uint32(uintptr(unsafe.Pointer(&buf[0]))))
	_ = json.Unmarshal(buf, out)
}

// sortedKeys 返回任意 string 键的 map 的排序键列表。
func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// sortedSetKeys 返回 bool 集合 map 的排序键列表。
func sortedSetKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
