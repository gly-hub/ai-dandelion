package generatedapp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/dao"
	"github.com/tetratelabs/wazero"
)

func TestArtifactSnapshotChangesWhenExecutableArtifactChanges(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	appID := "3658889f-3b80-45a9-88e5-8de80fa287b5"
	appDir := filepath.Join(root, appID)
	if err := os.MkdirAll(filepath.Join(appDir, "frontend"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"schemaVersion":"v2","id":"` + appID + `","name":"Test","version":"v1","export":"handle","frontendFile":"frontend.js","backendModule":"backend.wasm","dataModels":[]}`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "frontend.js"), []byte("export const version = 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "frontend", "ui.js"), []byte("export const ui = 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	minimalWASM := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	if err := os.WriteFile(filepath.Join(appDir, "backend.wasm"), minimalWASM, 0o644); err != nil {
		t.Fatal(err)
	}

	service := &Service{rootDir: root, runtime: wazero.NewRuntime(context.Background())}
	defer service.runtime.Close(context.Background())
	first, err := service.ArtifactSnapshot(appID)
	if err != nil {
		t.Fatalf("ArtifactSnapshot() error = %v", err)
	}
	if len(first.SHA256) != 64 {
		t.Fatalf("unexpected SHA256 %q", first.SHA256)
	}
	if err := os.WriteFile(filepath.Join(appDir, "frontend", "ui.js"), []byte("export const ui = 2"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := service.ArtifactSnapshot(appID)
	if err != nil {
		t.Fatalf("ArtifactSnapshot() after change error = %v", err)
	}
	if first.SHA256 == second.SHA256 {
		t.Fatal("artifact hash did not change after executable frontend module changed")
	}
}

func TestScaffoldPlatformDoesNotExposeRawSQLImports(t *testing.T) {
	t.Parallel()

	platform := backendPlatformTemplate()
	if strings.Contains(platform, "db_query") || strings.Contains(platform, "db_exec") {
		t.Fatalf("generated platform still exposes raw SQL imports: %s", platform)
	}
}

func TestArtifactSnapshotRejectsFrontendSymlink(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	appID := "3658889f-3b80-45a9-88e5-8de80fa287b5"
	writeExecutableArtifact(t, root, appID, "export const version = 1")
	if err := os.Symlink(filepath.Join(root, appID, "frontend.js"), filepath.Join(root, appID, "frontend", "linked.js")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	service := &Service{rootDir: root, runtime: wazero.NewRuntime(context.Background())}
	defer service.runtime.Close(context.Background())
	if _, err := service.ArtifactSnapshot(appID); err == nil {
		t.Fatal("ArtifactSnapshot accepted a frontend symlink")
	}
}

func TestValidateCapabilitiesRejectsUnknownQuery(t *testing.T) {
	err := validateCapabilities(manifest{ID: "3658889f-3b80-45a9-88e5-8de80fa287b5", Provides: []CapabilityProvide{{Name: "customer.search", Query: "missing"}}})
	if err == nil {
		t.Fatal("expected undeclared capability query to be rejected")
	}
}

func TestPromotedArtifactIsolatedFromDraftChanges(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	appID := "3658889f-3b80-45a9-88e5-8de80fa287b5"
	writeExecutableArtifact(t, root, appID, "export const version = 1")
	service, err := NewService(ctx, root, dao.NewGeneratedApp(nil))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close(ctx)

	snapshot, err := service.ArtifactSnapshot(appID)
	if err != nil {
		t.Fatalf("ArtifactSnapshot() error = %v", err)
	}
	if err := service.PromoteArtifact(appID, snapshot.SHA256); err != nil {
		t.Fatalf("PromoteArtifact() error = %v", err)
	}
	defer makeArtifactDirWritable(filepath.Join(root, ".releases", snapshot.SHA256))
	if err := os.WriteFile(filepath.Join(root, appID, "frontend.js"), []byte("export const version = 2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.PublishedArtifactSnapshot(appID, snapshot.SHA256); err != nil {
		t.Fatalf("PublishedArtifactSnapshot() error = %v", err)
	}
	if err := service.LoadApprovedArtifacts(ctx, map[string]string{appID: snapshot.SHA256}); err != nil {
		t.Fatalf("LoadApprovedArtifacts() error = %v", err)
	}
	frontend, err := service.FrontendCode(appID, "frontend.js")
	if err != nil {
		t.Fatalf("FrontendCode() error = %v", err)
	}
	if frontend != "export const version = 1" {
		t.Fatalf("published frontend = %q, want draft-independent content", frontend)
	}
}

func TestDraftFrontendBundlePreservesFrontendRelativePaths(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	appID := "3658889f-3b80-45a9-88e5-8de80fa287b5"
	writeExecutableArtifact(t, root, appID, "import { ui } from './frontend/ui.js'; export function render() { return ui }")
	service, err := NewService(ctx, root, dao.NewGeneratedApp(nil), WithDraftRuntime())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close(ctx)

	if _, err := service.LoadDraftApp(ctx, appID); err != nil {
		t.Fatalf("LoadDraftApp() error = %v", err)
	}
	_, entry, modules, err := service.FrontendBundle(appID)
	if err != nil {
		t.Fatalf("FrontendBundle() error = %v", err)
	}
	if entry != "frontend.js" {
		t.Fatalf("bundle entry = %q, want frontend.js", entry)
	}
	if _, ok := modules["frontend.js"]; !ok {
		t.Fatal("bundle is missing frontend.js")
	}
	if _, ok := modules["frontend/ui.js"]; !ok {
		t.Fatal("bundle is missing the frontend-relative import path")
	}
}

func TestPublishedArtifactValidationDoesNotEvictActiveModule(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	appID := "49621f74-331e-4be9-83c4-2b3cadbb8dc0"
	writeExecutableArtifact(t, root, appID, "export const version = 1")
	service, err := NewService(ctx, root, dao.NewGeneratedApp(nil))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close(ctx)

	snapshot, err := service.ArtifactSnapshot(appID)
	if err != nil {
		t.Fatalf("ArtifactSnapshot() error = %v", err)
	}
	if err := service.PromoteArtifact(appID, snapshot.SHA256); err != nil {
		t.Fatalf("PromoteArtifact() error = %v", err)
	}
	defer makeArtifactDirWritable(filepath.Join(root, ".releases", snapshot.SHA256))
	if err := service.LoadApprovedArtifacts(ctx, map[string]string{appID: snapshot.SHA256}); err != nil {
		t.Fatalf("LoadApprovedArtifacts() error = %v", err)
	}
	if _, err := service.PublishedArtifactSnapshot(appID, snapshot.SHA256); err != nil {
		t.Fatalf("PublishedArtifactSnapshot() error = %v", err)
	}
	_, err = service.Invoke(ctx, appID, nil)
	if err == nil || !strings.Contains(err.Error(), "missing export") {
		t.Fatalf("Invoke() error = %v, want missing export after successful instantiation", err)
	}
	if strings.Contains(err.Error(), "source module must be compiled") {
		t.Fatalf("validation evicted the active compiled module: %v", err)
	}
}

func TestLoadApprovedArtifactsRejectsChangedRelease(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	appID := "3658889f-3b80-45a9-88e5-8de80fa287b5"
	writeExecutableArtifact(t, root, appID, "export const version = 1")
	service, err := NewService(ctx, root, dao.NewGeneratedApp(nil))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close(ctx)

	snapshot, err := service.ArtifactSnapshot(appID)
	if err != nil {
		t.Fatalf("ArtifactSnapshot() error = %v", err)
	}
	if err := service.PromoteArtifact(appID, snapshot.SHA256); err != nil {
		t.Fatalf("PromoteArtifact() error = %v", err)
	}
	defer makeArtifactDirWritable(filepath.Join(root, ".releases", snapshot.SHA256))
	releaseFrontend := filepath.Join(root, ".releases", snapshot.SHA256, "frontend.js")
	if err := os.Chmod(releaseFrontend, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(releaseFrontend, []byte("export const version = changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := service.LoadApprovedArtifacts(ctx, map[string]string{appID: snapshot.SHA256}); err == nil {
		t.Fatal("LoadApprovedArtifacts() accepted a modified release")
	}
}

func TestReconcileLocalArtifactsRemovesOnlyUnreferencedArtifacts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	appID := "3658889f-3b80-45a9-88e5-8de80fa287b5"
	writeExecutableArtifact(t, root, appID, "export const version = 1")
	service, err := NewService(ctx, root, dao.NewGeneratedApp(nil))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer service.Close(ctx)

	snapshot, err := service.ArtifactSnapshot(appID)
	if err != nil {
		t.Fatalf("ArtifactSnapshot() error = %v", err)
	}
	if err := service.PromoteArtifact(appID, snapshot.SHA256); err != nil {
		t.Fatalf("PromoteArtifact() error = %v", err)
	}
	defer makeArtifactDirWritable(filepath.Join(root, ".releases", snapshot.SHA256))

	orphanSHA := strings.Repeat("a", 64)
	orphanDir := filepath.Join(root, ".releases", orphanSHA)
	if err := os.MkdirAll(orphanDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stagingDir := filepath.Join(root, ".releases", ".staging-abandoned")
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(stagingDir, old, old); err != nil {
		t.Fatal(err)
	}

	result, err := service.ReconcileLocalArtifacts(map[string]struct{}{snapshot.SHA256: {}}, time.Hour)
	if err != nil {
		t.Fatalf("ReconcileLocalArtifacts() error = %v", err)
	}
	if result.RemovedOrphans != 1 || result.RemovedStaging != 1 {
		t.Fatalf("reconcile result = %#v, want one orphan and one staging directory removed", result)
	}
	if _, err := os.Stat(filepath.Join(root, ".releases", snapshot.SHA256)); err != nil {
		t.Fatalf("referenced artifact was removed: %v", err)
	}
	if _, err := os.Stat(orphanDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("orphan artifact still exists, err=%v", err)
	}
	if _, err := os.Stat(stagingDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale staging directory still exists, err=%v", err)
	}
}

func writeExecutableArtifact(t *testing.T, root, appID, frontend string) {
	t.Helper()
	appDir := filepath.Join(root, appID)
	if err := os.MkdirAll(filepath.Join(appDir, "frontend"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"schemaVersion":"v2","id":"` + appID + `","name":"Test","version":"v1","export":"handle","frontendFile":"frontend.js","backendModule":"backend.wasm","dataModels":[]}`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "frontend.js"), []byte(frontend), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "frontend", "ui.js"), []byte("export const ui = 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	minimalWASM := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	if err := os.WriteFile(filepath.Join(appDir, "backend.wasm"), minimalWASM, 0o644); err != nil {
		t.Fatal(err)
	}
}
