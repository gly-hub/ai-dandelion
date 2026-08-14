package logic

import (
	"testing"

	"github.com/gly-hub/ai-dandelion/system/internal/model"
)

func TestUploadSessionRetainsPresignedUploadURL(t *testing.T) {
	session := uploadSession(&model.Upload{
		UUID:        "upload-uuid",
		Mode:        "put",
		ContentType: "image/jpeg",
	}, false, "https://storage.example.com/upload-uuid?signature=value")

	if session.GetUrl() == "" {
		t.Fatal("upload session should include the presigned PUT URL")
	}
}
