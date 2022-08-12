package tools_test

import (
	"goblog/tools"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func TestMigration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to open sqlite db: 'test.db'")
	}
	tools.Migrate(db)
}
