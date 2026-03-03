package tools_test

import (
	"goblog/tools"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
	"testing"
)

func TestMigration(t *testing.T) {
	os.Remove("test.db")
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to open sqlite db: 'test.db'")
	}
	t.Cleanup(func() { os.Remove("test.db") })
	err = tools.Migrate(db)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}
}

// TestMigrationWithOldSchema reproduces the production issue where blog_users
// was created with id varchar(255) and a table-level PRIMARY KEY by an older
// GORM version. This caused "more than one primary key" when AutoMigrate tried
// to alter the column to integer PRIMARY KEY AUTOINCREMENT.
func TestMigrationWithOldSchema(t *testing.T) {
	os.Remove("test_old_schema.db")
	db, err := gorm.Open(sqlite.Open("test_old_schema.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	t.Cleanup(func() { os.Remove("test_old_schema.db") })

	// Exact DDL from the production goblog.db
	if err := db.Exec(`CREATE TABLE "blog_users" ("id" varchar(255),"github_id" integer,
		"login" text,"avatar_url" text,"name" text,"email" text,"access_token" text,
		PRIMARY KEY ("id"))`).Error; err != nil {
		t.Fatalf("failed to create old schema: %v", err)
	}

	// Also create tags with the old schema (table-level PK)
	if err := db.Exec(`CREATE TABLE "tags" ("name" varchar(255), PRIMARY KEY ("name"))`).Error; err != nil {
		t.Fatalf("failed to create old tags schema: %v", err)
	}

	// Insert test data with a mix of valid ids and NULL ids (as seen in production)
	if err := db.Exec(`INSERT INTO blog_users (id, github_id, login, name) VALUES
		('23049896', 23049896, 'testuser', 'Test User'),
		(NULL, NULL, 'nulluser', 'Null User')`).Error; err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	if err := db.Exec(`INSERT INTO tags (name) VALUES ('golang'), ('sqlite')`).Error; err != nil {
		t.Fatalf("failed to insert tag data: %v", err)
	}

	err = tools.Migrate(db)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Verify user data survived
	var userCount int64
	db.Raw("SELECT count(*) FROM blog_users").Scan(&userCount)
	if userCount != 2 {
		t.Fatalf("expected 2 users, got %d", userCount)
	}

	// Verify the numeric id was preserved
	var login string
	db.Raw("SELECT login FROM blog_users WHERE id = 23049896").Scan(&login)
	if login != "testuser" {
		t.Fatalf("expected login 'testuser' for id 23049896, got '%s'", login)
	}

	// Verify tag data survived
	var tagCount int64
	db.Raw("SELECT count(*) FROM tags").Scan(&tagCount)
	if tagCount != 2 {
		t.Fatalf("expected 2 tags, got %d", tagCount)
	}
}
