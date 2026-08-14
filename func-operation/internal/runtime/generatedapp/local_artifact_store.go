package generatedapp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LocalArtifactStore struct {
	releaseDir string
}

func NewLocalArtifactStore(rootDir string) *LocalArtifactStore {
	return &LocalArtifactStore{releaseDir: filepath.Join(rootDir, ".releases")}
}

func (s *LocalArtifactStore) Promote(_ context.Context, bundle ArtifactBundle) error {
	if !artifactSHA256Pattern.MatchString(bundle.SHA256) {
		return errors.New("invalid artifact hash")
	}
	target := filepath.Join(s.releaseDir, bundle.SHA256)
	if _, err := os.Stat(target); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(s.releaseDir, 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(s.releaseDir, ".staging-")
	if err != nil {
		return err
	}
	defer func() {
		_ = makeArtifactDirWritable(tmp)
		_ = os.RemoveAll(tmp)
	}()
	if err := writeArtifactBundle(tmp, bundle); err != nil {
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		if _, statErr := os.Stat(target); statErr == nil {
			return nil
		}
		return err
	}
	return sealArtifactDir(target)
}

func (s *LocalArtifactStore) Materialize(_ context.Context, artifactSHA string) (string, error) {
	if !artifactSHA256Pattern.MatchString(artifactSHA) {
		return "", errors.New("invalid artifact hash")
	}
	dir := filepath.Join(s.releaseDir, artifactSHA)
	info, err := os.Stat(dir)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("local artifact %q is not a directory", artifactSHA)
	}
	return dir, nil
}

func (s *LocalArtifactStore) Reconcile(_ context.Context, referenced map[string]struct{}, staleStagingAfter time.Duration) (ArtifactReconcileResult, error) {
	for hash := range referenced {
		if !artifactSHA256Pattern.MatchString(hash) {
			return ArtifactReconcileResult{}, errors.New("invalid referenced artifact hash")
		}
	}
	entries, err := os.ReadDir(s.releaseDir)
	if errors.Is(err, os.ErrNotExist) {
		return ArtifactReconcileResult{}, nil
	}
	if err != nil {
		return ArtifactReconcileResult{}, fmt.Errorf("read local release store: %w", err)
	}
	result := ArtifactReconcileResult{}
	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(s.releaseDir, name)
		if strings.HasPrefix(name, ".staging-") {
			info, statErr := entry.Info()
			if statErr != nil {
				return result, statErr
			}
			if staleStagingAfter > 0 && time.Since(info.ModTime()) >= staleStagingAfter {
				if err := removeLocalArtifact(path); err != nil {
					return result, fmt.Errorf("remove stale artifact staging %q: %w", name, err)
				}
				result.RemovedStaging++
			}
			continue
		}
		if !artifactSHA256Pattern.MatchString(name) {
			continue
		}
		if _, keep := referenced[name]; keep {
			continue
		}
		if err := removeLocalArtifact(path); err != nil {
			return result, fmt.Errorf("remove orphaned artifact %q: %w", name, err)
		}
		result.RemovedOrphans++
	}
	return result, nil
}

func writeArtifactBundle(target string, bundle ArtifactBundle) error {
	for name, contents := range bundle.Files {
		if err := validateArtifactFileName(name); err != nil {
			return err
		}
		cleanName := filepath.ToSlash(filepath.Clean(name))
		path := filepath.Join(target, filepath.FromSlash(cleanName))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, contents, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func removeLocalArtifact(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return os.Remove(path)
	}
	if err := makeArtifactDirWritable(path); err != nil {
		return err
	}
	return os.RemoveAll(path)
}
