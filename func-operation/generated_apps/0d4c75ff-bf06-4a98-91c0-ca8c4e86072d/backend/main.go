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
	case "list":
		return handleList(req.Data)
	case "detail":
		return handleDetail(req.Data)
	case "create":
		return handleCreate(req.Data)
	case "update":
		return handleUpdate(req.Data)
	case "delete":
		return handleDelete(req.Data)
	case "lend":
		return handleLend(req.Data)
	case "return_book":
		return handleReturnBook(req.Data)
	case "offshelf":
		return handleOffshelf(req.Data)
	case "onshelf":
		return handleOnshelf(req.Data)
	default:
		return storeResponse(InvokeResponse{
			Success: false,
			Error:   "unknown action: " + req.Action,
		})
	}
}
