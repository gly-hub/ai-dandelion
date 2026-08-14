package generatedapp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const s3ArtifactCompleteMarker = ".artifact-complete"
const s3ArtifactCacheMarker = ".artifact-ready"

type S3ArtifactStore struct {
	client   *minio.Client
	bucket   string
	prefix   string
	cacheDir string
}

func NewS3ArtifactStore(rootDir string, config S3ArtifactStoreConfig) (*S3ArtifactStore, error) {
	if strings.TrimSpace(config.Endpoint) == "" || strings.TrimSpace(config.AccessKey) == "" || strings.TrimSpace(config.SecretKey) == "" || strings.TrimSpace(config.Bucket) == "" {
		return nil, errors.New("s3 artifact store endpoint, access key, secret key, and bucket are required")
	}
	client, err := minio.New(strings.TrimSpace(config.Endpoint), &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: config.UseSSL,
		Region: strings.TrimSpace(config.Region),
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 artifact client: %w", err)
	}
	prefix := strings.Trim(strings.TrimSpace(config.Prefix), "/")
	if prefix == "" {
		prefix = "func-operation/releases"
	}
	return &S3ArtifactStore{
		client:   client,
		bucket:   strings.TrimSpace(config.Bucket),
		prefix:   prefix,
		cacheDir: filepath.Join(rootDir, ".artifact-cache"),
	}, nil
}

func (s *S3ArtifactStore) Promote(ctx context.Context, bundle ArtifactBundle) error {
	if !artifactSHA256Pattern.MatchString(bundle.SHA256) || len(bundle.Files) == 0 {
		return errors.New("invalid artifact bundle")
	}
	if err := s.requireBucket(ctx); err != nil {
		return err
	}
	markerKey := s.objectKey(bundle.SHA256, s3ArtifactCompleteMarker)
	if _, err := s.client.StatObject(ctx, s.bucket, markerKey, minio.StatObjectOptions{}); err == nil {
		return nil
	} else if minio.ToErrorResponse(err).Code != "NoSuchKey" && minio.ToErrorResponse(err).Code != "NoSuchObject" {
		return fmt.Errorf("stat artifact completion marker: %w", err)
	}
	for name, contents := range bundle.Files {
		if err := validateArtifactFileName(name); err != nil {
			return err
		}
		if _, err := s.client.PutObject(ctx, s.bucket, s.objectKey(bundle.SHA256, name), bytes.NewReader(contents), int64(len(contents)), minio.PutObjectOptions{}); err != nil {
			return fmt.Errorf("upload artifact %q: %w", name, err)
		}
	}
	if _, err := s.client.PutObject(ctx, s.bucket, markerKey, strings.NewReader(bundle.SHA256), int64(len(bundle.SHA256)), minio.PutObjectOptions{ContentType: "text/plain"}); err != nil {
		return fmt.Errorf("write artifact completion marker: %w", err)
	}
	return nil
}

func (s *S3ArtifactStore) Materialize(ctx context.Context, artifactSHA string) (string, error) {
	if !artifactSHA256Pattern.MatchString(artifactSHA) {
		return "", errors.New("invalid artifact hash")
	}
	cacheTarget := filepath.Join(s.cacheDir, artifactSHA)
	if cachedArtifactReady(cacheTarget, artifactSHA) {
		return cacheTarget, nil
	}
	if _, err := os.Lstat(cacheTarget); err == nil {
		if err := removeLocalArtifact(cacheTarget); err != nil {
			return "", fmt.Errorf("remove incomplete artifact cache: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := s.requireBucket(ctx); err != nil {
		return "", err
	}
	if _, err := s.client.StatObject(ctx, s.bucket, s.objectKey(artifactSHA, s3ArtifactCompleteMarker), minio.StatObjectOptions{}); err != nil {
		return "", fmt.Errorf("stat artifact completion marker: %w", err)
	}
	if err := os.MkdirAll(s.cacheDir, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp(s.cacheDir, ".staging-")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = makeArtifactDirWritable(tmp)
		_ = os.RemoveAll(tmp)
	}()
	found := false
	for object := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Prefix: s.objectPrefix(artifactSHA), Recursive: true}) {
		if object.Err != nil {
			return "", fmt.Errorf("list artifact objects: %w", object.Err)
		}
		relative := strings.TrimPrefix(object.Key, s.objectPrefix(artifactSHA))
		if relative == s3ArtifactCompleteMarker {
			continue
		}
		if err := validateArtifactFileName(relative); err != nil {
			return "", err
		}
		reader, err := s.client.GetObject(ctx, s.bucket, object.Key, minio.GetObjectOptions{})
		if err != nil {
			return "", err
		}
		contents, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil {
			return "", readErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		path := filepath.Join(tmp, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(path, contents, 0o644); err != nil {
			return "", err
		}
		found = true
	}
	if !found {
		return "", errors.New("s3 artifact contains no files")
	}
	if err := os.WriteFile(filepath.Join(tmp, s3ArtifactCacheMarker), []byte(artifactSHA), 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, cacheTarget); err != nil {
		if cachedArtifactReady(cacheTarget, artifactSHA) {
			return cacheTarget, nil
		}
		return "", err
	}
	if err := sealArtifactDir(cacheTarget); err != nil {
		return "", err
	}
	return cacheTarget, nil
}

func (s *S3ArtifactStore) Reconcile(ctx context.Context, referenced map[string]struct{}, staleStagingAfter time.Duration) (ArtifactReconcileResult, error) {
	if err := s.requireBucket(ctx); err != nil {
		return ArtifactReconcileResult{}, err
	}
	for hash := range referenced {
		if !artifactSHA256Pattern.MatchString(hash) {
			return ArtifactReconcileResult{}, errors.New("invalid referenced artifact hash")
		}
	}
	type artifactObjects struct {
		keys       []string
		completed  bool
		lastChange time.Time
	}
	artifacts := make(map[string]*artifactObjects)
	for object := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Prefix: s.prefix + "/", Recursive: true}) {
		if object.Err != nil {
			return ArtifactReconcileResult{}, object.Err
		}
		relative := strings.TrimPrefix(object.Key, s.prefix+"/")
		parts := strings.SplitN(relative, "/", 2)
		if len(parts) != 2 || !artifactSHA256Pattern.MatchString(parts[0]) {
			continue
		}
		item := artifacts[parts[0]]
		if item == nil {
			item = &artifactObjects{}
			artifacts[parts[0]] = item
		}
		item.keys = append(item.keys, object.Key)
		item.completed = item.completed || parts[1] == s3ArtifactCompleteMarker
		if object.LastModified.After(item.lastChange) {
			item.lastChange = object.LastModified
		}
	}
	result := ArtifactReconcileResult{}
	for hash, item := range artifacts {
		if _, keep := referenced[hash]; keep {
			continue
		}
		if !item.completed && staleStagingAfter > 0 && time.Since(item.lastChange) < staleStagingAfter {
			continue
		}
		for _, key := range item.keys {
			if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
				return result, fmt.Errorf("remove s3 artifact %q: %w", hash, err)
			}
		}
		if item.completed {
			result.RemovedOrphans++
		} else {
			result.RemovedStaging++
		}
	}
	return result, nil
}

func (s *S3ArtifactStore) requireBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check s3 artifact bucket: %w", err)
	}
	if !exists {
		return fmt.Errorf("s3 artifact bucket %q does not exist", s.bucket)
	}
	return nil
}

func (s *S3ArtifactStore) objectPrefix(artifactSHA string) string {
	return s.prefix + "/" + artifactSHA + "/"
}

func (s *S3ArtifactStore) objectKey(artifactSHA, name string) string {
	return s.objectPrefix(artifactSHA) + strings.TrimPrefix(name, "/")
}

func validateArtifactFileName(name string) error {
	original := filepath.ToSlash(name)
	cleanName := filepath.ToSlash(filepath.Clean(name))
	if original == "" || strings.HasPrefix(original, "/") || cleanName == "." || strings.HasPrefix(cleanName, "../") || filepath.IsAbs(name) || cleanName != original {
		return errors.New("invalid artifact file path")
	}
	return nil
}

func cachedArtifactReady(dir, artifactSHA string) bool {
	raw, err := os.ReadFile(filepath.Join(dir, s3ArtifactCacheMarker))
	return err == nil && strings.TrimSpace(string(raw)) == artifactSHA
}
