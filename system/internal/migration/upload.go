package migration

import (
	"fmt"

	"github.com/gly-hub/ai-dandelion/system/internal/model"
	"gorm.io/gorm"
)

// EnsureUploadOwnership upgrades the legacy global MD5 index to a per-user
// uniqueness constraint. It is intentionally explicit because production boot
// does not AutoMigrate already-existing tables.
func EnsureUploadOwnership(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("upload migration database is nil")
	}
	migrator := db.Migrator()
	if migrator.HasIndex(&model.Upload{}, "idx_uploads_md5") {
		if err := migrator.DropIndex(&model.Upload{}, "idx_uploads_md5"); err != nil {
			return fmt.Errorf("drop legacy upload MD5 index: %w", err)
		}
	}
	if err := db.AutoMigrate(&model.Upload{}); err != nil {
		return fmt.Errorf("migrate upload ownership: %w", err)
	}
	return nil
}
