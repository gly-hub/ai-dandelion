package uploader_minio

import "testing"

func TestPublicURLUsesConfiguredDomain(t *testing.T) {
	uploader := &MinioUploader{
		minioBucket: "resource",
		basePath:    "uploads",
		domain:      "s3.example.com",
	}

	url, err := uploader.PublicURL("4abec9fc-2b0d-42a6-8500-426eb599a558.jpeg")
	if err != nil {
		t.Fatalf("create public URL: %v", err)
	}
	const want = "https://s3.example.com/resource/uploads/4/a/4abec9fc-2b0d-42a6-8500-426eb599a558.jpeg"
	if url != want {
		t.Fatalf("public URL = %q, want %q", url, want)
	}
}

func TestPublicURLAllowsPresignedFallback(t *testing.T) {
	uploader := &MinioUploader{minioBucket: "resource", basePath: "uploads"}

	url, err := uploader.PublicURL("4abec9fc-2b0d-42a6-8500-426eb599a558.jpeg")
	if err != nil {
		t.Fatalf("resolve public URL: %v", err)
	}
	if url != "" {
		t.Fatalf("public URL = %q, want empty fallback", url)
	}
}
