package tools

import (
	"goblog/auth"
	"goblog/blog"
	"gorm.io/gorm"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// fixBlogUsersTable migrates the blog_users table from the old schema
// (id varchar(255) with table-level PRIMARY KEY) to the new schema
// (id integer PRIMARY KEY AUTOINCREMENT). The old schema was created by
// an older GORM version and causes "more than one primary key" errors
// during AutoMigrate.
func fixBlogUsersTable(db *gorm.DB) error {
	var createSQL string
	row := db.Raw("SELECT sql FROM sqlite_master WHERE type = 'table' AND tbl_name = 'blog_users' AND name = 'blog_users'").Row()
	if row == nil {
		return nil
	}
	if err := row.Scan(&createSQL); err != nil {
		return nil // table doesn't exist yet
	}

	// Only fix if the id column is varchar (old schema)
	if !strings.Contains(strings.ToLower(createSQL), "varchar") {
		return nil
	}

	log.Println("Migrating blog_users from old varchar schema to integer primary key")

	return db.Transaction(func(tx *gorm.DB) error {
		// Create the new table with correct schema
		if err := tx.Exec(`CREATE TABLE blog_users_new (
			"id" integer PRIMARY KEY AUTOINCREMENT,
			"login" text,
			"avatar_url" text,
			"name" text,
			"email" text,
			"access_token" text
		)`).Error; err != nil {
			return err
		}

		// Copy data. Use CAST for id, and assign new ids for rows with NULL/empty id.
		// Insert rows with valid numeric ids first, preserving their ids.
		if err := tx.Exec(`INSERT INTO blog_users_new (id, login, avatar_url, name, email, access_token)
			SELECT CAST(id AS INTEGER), login, avatar_url, name, email, access_token
			FROM blog_users
			WHERE id IS NOT NULL AND id != '' AND CAST(id AS INTEGER) > 0`).Error; err != nil {
			return err
		}

		// Insert rows with NULL/empty ids, letting SQLite auto-assign ids.
		if err := tx.Exec(`INSERT INTO blog_users_new (login, avatar_url, name, email, access_token)
			SELECT login, avatar_url, name, email, access_token
			FROM blog_users
			WHERE id IS NULL OR id = '' OR CAST(id AS INTEGER) = 0`).Error; err != nil {
			return err
		}

		if err := tx.Exec("DROP TABLE blog_users").Error; err != nil {
			return err
		}
		if err := tx.Exec("ALTER TABLE blog_users_new RENAME TO blog_users").Error; err != nil {
			return err
		}
		return nil
	})
}

// fixTableLevelPrimaryKey checks if a table's DDL has a table-level PRIMARY KEY
// constraint and no inline PRIMARY KEY. If so, it recreates the table without
// the table-level constraint so GORM's AutoMigrate can safely add the inline one.
func fixTableLevelPrimaryKey(db *gorm.DB, table string) error {
	var createSQL string
	row := db.Raw("SELECT sql FROM sqlite_master WHERE type = 'table' AND tbl_name = ? AND name = ?", table, table).Row()
	if row == nil {
		return nil
	}
	if err := row.Scan(&createSQL); err != nil {
		return nil
	}

	upper := strings.ToUpper(createSQL)

	// Skip if already has inline PRIMARY KEY (nothing to fix)
	if strings.Contains(upper, "PRIMARY KEY AUTOINCREMENT") {
		return nil
	}

	body := createSQL[strings.Index(createSQL, "(")+1 : strings.LastIndex(createSQL, ")")]
	parts := splitDDLFields(body)
	tablePKIndex := -1
	for i, part := range parts {
		trimmed := strings.TrimSpace(part)
		trimmedUpper := strings.ToUpper(trimmed)
		if strings.HasPrefix(trimmedUpper, "PRIMARY KEY") {
			tablePKIndex = i
			break
		}
	}

	if tablePKIndex == -1 {
		return nil
	}

	log.Printf("Fixing table-level PRIMARY KEY in table %s", table)

	// Remove the table-level PRIMARY KEY constraint
	parts = append(parts[:tablePKIndex], parts[tablePKIndex+1:]...)
	head := createSQL[:strings.Index(createSQL, "(")+1]
	newDDL := head + strings.Join(parts, ",") + ")"

	// Replace table name with temp name
	tempTable := table + "__fix"
	newDDL = strings.Replace(newDDL, "`"+table+"`", "`"+tempTable+"`", 1)
	newDDL = strings.Replace(newDDL, "\""+table+"\"", "`"+tempTable+"`", 1)

	var columns []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		trimmedUpper := strings.ToUpper(trimmed)
		if strings.HasPrefix(trimmedUpper, "CONSTRAINT") || strings.HasPrefix(trimmedUpper, "CHECK") {
			continue
		}
		name := extractColumnName(trimmed)
		if name != "" {
			columns = append(columns, "`"+name+"`")
		}
	}
	colList := strings.Join(columns, ",")

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(newDDL).Error; err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO `"+tempTable+"`("+colList+") SELECT "+colList+" FROM `"+table+"`").Error; err != nil {
			return err
		}
		if err := tx.Exec("DROP TABLE `" + table + "`").Error; err != nil {
			return err
		}
		if err := tx.Exec("ALTER TABLE `" + tempTable + "` RENAME TO `" + table + "`").Error; err != nil {
			return err
		}
		return nil
	})
}

func splitDDLFields(body string) []string {
	var parts []string
	depth := 0
	start := 0
	for i, c := range body {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, body[start:i])
				start = i + 1
			}
		}
	}
	if start < len(body) {
		parts = append(parts, body[start:])
	}
	return parts
}

func extractColumnName(field string) string {
	field = strings.TrimSpace(field)
	if field == "" {
		return ""
	}
	if field[0] == '`' || field[0] == '"' || field[0] == '\'' {
		quote := field[0]
		end := strings.IndexByte(field[1:], quote)
		if end >= 0 {
			return field[1 : end+1]
		}
	}
	end := strings.IndexAny(field, " \t")
	if end > 0 {
		return field[:end]
	}
	return field
}

// seedDefaultSettings inserts default settings if the settings table is empty.
// This handles upgrades from older versions that didn't have a settings table.
func seedDefaultSettings(db *gorm.DB) {
	var count int64
	db.Raw("SELECT count(*) FROM settings").Scan(&count)
	if count > 0 {
		return
	}

	log.Println("Seeding default settings")
	defaults := []blog.Setting{
		{Key: "site_title", Type: "text", Value: "Jason Ernst"},
		{Key: "site_subtitle", Type: "text", Value: "Software Engineer"},
		{Key: "site_logo_letters", Type: "text", Value: "JE"},
		{Key: "site_tags", Type: "text", Value: "Decentralization, Mesh Net"},
		{Key: "landing_page_image", Type: "file", Value: "/img/profile.png"},
		{Key: "favicon", Type: "file", Value: "/img/favicon.ico"},
		{Key: "github_url", Type: "text", Value: "https://www.github.com/compscidr"},
		{Key: "linkedin_url", Type: "text", Value: "https://www.linkedin.com/in/jasonernst/"},
		{Key: "x_url", Type: "text", Value: "https://www.x.com/compscidr"},
		{Key: "keybase_url", Type: "text", Value: "https://keybase.io/compscidr"},
		{Key: "instagram_url", Type: "text", Value: "https://www.instagram.com/compscidr"},
		{Key: "facebook_url", Type: "text", Value: "https://www.facebook.com/jason.b.ernst"},
		{Key: "strava_url", Type: "text", Value: "https://www.strava.com/athletes/2021127"},
		{Key: "spotify_url", Type: "text", Value: "https://open.spotify.com/user/csgrad"},
		{Key: "xbox_url", Type: "text", Value: "https://account.xbox.com/en-us/profile?gamertag=Compscidr"},
		{Key: "steam_url", Type: "text", Value: "https://steamcommunity.com/id/compscidr"},
	}
	for _, s := range defaults {
		db.Create(&s)
	}
}

var reInternalLink = regexp.MustCompile(`\]\(/(?:posts/)?(\d{4})/(\d{1,2})/(\d{1,2})/([^)\s]+)\)`)

// seedBacklinks computes internal backlinks for all existing posts.
// Only runs if the backlinks table is empty (first migration).
func seedBacklinks(db *gorm.DB) {
	var count int64
	db.Model(&blog.Backlink{}).Count(&count)
	if count > 0 {
		return
	}

	var posts []blog.Post
	db.Find(&posts)
	if len(posts) == 0 {
		return
	}

	log.Println("Computing backlinks for all existing posts")
	for _, post := range posts {
		matches := reInternalLink.FindAllStringSubmatch(post.Content, -1)
		seen := make(map[uint]bool)
		for _, match := range matches {
			year, err := strconv.Atoi(match[1])
			if err != nil {
				log.Printf("seedBacklinks: invalid year %q in post %d: %v", match[1], post.ID, err)
				continue
			}
			month, err := strconv.Atoi(match[2])
			if err != nil {
				log.Printf("seedBacklinks: invalid month %q in post %d: %v", match[2], post.ID, err)
				continue
			}
			day, err := strconv.Atoi(match[3])
			if err != nil {
				log.Printf("seedBacklinks: invalid day %q in post %d: %v", match[3], post.ID, err)
				continue
			}
			slug := match[4]

			// Use exact slug match and bounded date range
			var target blog.Post
			startOfDay := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
			endOfDay := startOfDay.Add(24 * time.Hour)
			if err := db.Where("slug = ? AND created_at >= ? AND created_at < ?", slug, startOfDay, endOfDay).
				First(&target).Error; err != nil {
				continue
			}
			if target.ID == post.ID || seen[target.ID] {
				continue
			}
			seen[target.ID] = true
			if err := db.Create(&blog.Backlink{SourcePostID: post.ID, TargetPostID: target.ID}).Error; err != nil {
				log.Printf("seedBacklinks: error creating backlink from post %d to %d: %v", post.ID, target.ID, err)
			}
		}
	}
}

func Migrate(db *gorm.DB) error {
	// Fix blog_users table: convert old varchar(255) id to integer autoincrement
	if err := fixBlogUsersTable(db); err != nil {
		log.Printf("Warning: could not fix blog_users table: %v", err)
	}

	// Fix other tables that may have table-level PRIMARY KEY constraints
	for _, table := range []string{"tags", "posts", "admin_users", "settings"} {
		if err := fixTableLevelPrimaryKey(db, table); err != nil {
			log.Printf("Warning: could not fix table %s: %v", table, err)
		}
	}

	err := db.AutoMigrate(&auth.BlogUser{}, &blog.Post{}, &blog.Tag{}, &auth.AdminUser{}, &blog.Setting{}, &blog.Comment{}, &blog.Backlink{}, &blog.ExternalBacklink{})
	if err != nil {
		log.Println("Error migrating tables: " + err.Error())
		return err
	}

	seedDefaultSettings(db)
	seedBacklinks(db)

	return nil
}
