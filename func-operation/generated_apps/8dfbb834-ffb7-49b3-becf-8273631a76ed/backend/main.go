//go:build wasip1

package main

import (
	"encoding/json"
	"unsafe"
)

var requestBuffer []byte

func main() {}

//go:wasmexport alloc
func alloc(size uint32) uint32 {
	requestBuffer = make([]byte, size)
	return uint32(uintptr(unsafe.Pointer(&requestBuffer[0])))
}

//go:wasmexport handle
func handle(reqPtr, reqLen uint32) uint64 {
	reqBytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(reqPtr))), reqLen)
	var req InvokeRequest
	if len(reqBytes) > 0 {
		if err := json.Unmarshal(reqBytes, &req); err != nil {
			return storeResponse(InvokeResponse{Error: "JSON解析失败: " + err.Error()})
		}
	}

	switch req.Action {
	case "teacher_list":
		return handleTeacherList(req.Data)
	case "teacher_detail":
		return handleTeacherDetail(req.Data)
	case "teacher_create":
		return handleTeacherCreate(req.Data)
	case "teacher_update":
		return handleTeacherUpdate(req.Data)
	case "teacher_change_status":
		return handleTeacherChangeStatus(req.Data)
	case "teacher_delete":
		return handleTeacherDelete(req.Data)
	default:
		return storeResponse(InvokeResponse{
			Success: false,
			Error:   "unknown action: " + req.Action,
		})
	}
}
