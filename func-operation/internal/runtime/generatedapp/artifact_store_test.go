package generatedapp

import "testing"

func TestNewArtifactStoreDefaultsToLocal(t *testing.T) {
	t.Parallel()

	store, err := NewArtifactStore(t.TempDir(), ArtifactStoreConfig{})
	if err != nil {
		t.Fatalf("NewArtifactStore() error = %v", err)
	}
	if _, ok := store.(*LocalArtifactStore); !ok {
		t.Fatalf("store type = %T, want *LocalArtifactStore", store)
	}
}

func TestNewArtifactStoreRejectsInvalidS3Config(t *testing.T) {
	t.Parallel()

	if _, err := NewArtifactStore(t.TempDir(), ArtifactStoreConfig{Driver: ArtifactStoreDriverS3}); err == nil {
		t.Fatal("NewArtifactStore() accepted incomplete S3 configuration")
	}
}

func TestNewArtifactStoreRejectsUnknownDriver(t *testing.T) {
	t.Parallel()

	if _, err := NewArtifactStore(t.TempDir(), ArtifactStoreConfig{Driver: "filesystem"}); err == nil {
		t.Fatal("NewArtifactStore() accepted an unknown driver")
	}
}

func TestValidateArtifactFileNameRejectsPathAliases(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"../manifest.json", "frontend/../manifest.json", "frontend//ui.js", "/manifest.json"} {
		if err := validateArtifactFileName(name); err == nil {
			t.Fatalf("validateArtifactFileName(%q) accepted a path alias", name)
		}
	}
}
