package uploader_minio

import "time"

const (
	PreassignedUploadPartUrlExpireTime = time.Hour * 24 * 7
	DefaultPresignedUploadExpireTime   = time.Hour
	DefaultPresignedPreviewExpireTime  = time.Hour
	DefaultPresignedDownloadExpireTime = time.Hour
)

const (
	UploadModePut       = "put"
	UploadModeMultipart = "multipart"
)

const (
	IllegalTotalChunkCounts = "totalChunkCounts is illegal"
	IllegalSize             = "size is illegal"
	IllegalFileName         = "file name is illegal"
)
