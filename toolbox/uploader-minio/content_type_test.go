package uploader_minio

import "testing"

func TestGetContentTypeRecognizesSourceFiles(t *testing.T) {
	uploader := &MinioUploader{}
	for postfix, want := range map[string]string{
		".go":  "text/x-go",
		".js":  "text/javascript",
		".ts":  "text/typescript",
		".py":  "text/x-python",
		".md":  "text/markdown",
		".csv": "text/csv",
	} {
		if got := uploader.getContentType(postfix); got != want {
			t.Errorf("getContentType(%q) = %q, want %q", postfix, got, want)
		}
	}
}
