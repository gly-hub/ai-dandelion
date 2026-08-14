package generatedapp

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	ArtifactStoreDriverLocal = "local"
	ArtifactStoreDriverS3    = "s3"
)

type ArtifactStoreConfig struct {
	Driver string
	S3     S3ArtifactStoreConfig
}

type S3ArtifactStoreConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Prefix    string
	Region    string
	UseSSL    bool
}

type ArtifactBundle struct {
	SHA256 string
	Files  map[string][]byte
}

type ArtifactStore interface {
	Promote(context.Context, ArtifactBundle) error
	Materialize(context.Context, string) (string, error)
	Reconcile(context.Context, map[string]struct{}, time.Duration) (ArtifactReconcileResult, error)
}

func NewArtifactStore(rootDir string, config ArtifactStoreConfig) (ArtifactStore, error) {
	driver := strings.ToLower(strings.TrimSpace(config.Driver))
	if driver == "" {
		driver = ArtifactStoreDriverLocal
	}
	switch driver {
	case ArtifactStoreDriverLocal:
		return NewLocalArtifactStore(rootDir), nil
	case ArtifactStoreDriverS3:
		return NewS3ArtifactStore(rootDir, config.S3)
	default:
		return nil, errors.New("unsupported artifact store driver")
	}
}
