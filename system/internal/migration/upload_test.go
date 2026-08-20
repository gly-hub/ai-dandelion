package migration

import (
	"testing"

	"github.com/gly-hub/ai-dandelion/system/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEnsureUploadOwnershipReplacesLegacyMD5Index(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec("CREATE TABLE uploads (uuid TEXT PRIMARY KEY, md5 TEXT NOT NULL)").Error; err != nil {
		t.Fatalf("create legacy upload table: %v", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX idx_uploads_md5 ON uploads(md5)").Error; err != nil {
		t.Fatalf("create legacy MD5 index: %v", err)
	}
	if err := EnsureUploadOwnership(db); err != nil {
		t.Fatalf("migrate upload ownership: %v", err)
	}
	if db.Migrator().HasIndex(&model.Upload{}, "idx_uploads_md5") {
		t.Fatal("legacy global MD5 index still exists")
	}
	if !db.Migrator().HasIndex(&model.Upload{}, "uk_upload_user_md5") {
		t.Fatal("per-user MD5 index was not created")
	}
	for _, item := range []model.Upload{
		{UUID: "upload-a", UserID: "user-a", MD5: "same"},
		{UUID: "upload-b", UserID: "user-b", MD5: "same"},
	} {
		if err := db.Create(&item).Error; err != nil {
			t.Fatalf("create per-user upload: %v", err)
		}
	}
}
