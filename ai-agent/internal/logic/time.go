package logic

import "time"

func nowUnixMicro() int64 {
	return time.Now().UnixMicro()
}
