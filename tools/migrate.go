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

// seedDefaultPostType creates the default "Post" post type and assigns existing posts to it.
func seedDefaultPostType(db *gorm.DB) {
	var count int64
	db.Model(&blog.PostType{}).Count(&count)
	if count > 0 {
		return
	}

	log.Println("Seeding default post type")
	defaultType := blog.PostType{
		Name:        "Post",
		Slug:        "posts",
		Description: "Blog posts",
	}
	if err := db.Create(&defaultType).Error; err != nil {
		log.Printf("Failed to seed default post type: %v", err)
		return
	}

	// Assign all existing posts with post_type_id = 0 to the default type
	if err := db.Model(&blog.Post{}).Where("post_type_id = 0 OR post_type_id IS NULL").Update("post_type_id", defaultType.ID).Error; err != nil {
		log.Printf("Failed to assign existing posts to default post type: %v", err)
	}
}

// seedDefaultSettings ensures all default settings exist.
// Uses FirstOrCreate so that new settings are added on upgrade without
// overwriting existing values.
func seedDefaultSettings(db *gorm.DB) {
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
		{Key: "custom_header_code", Type: "textarea", Value: ""},
		{Key: "custom_footer_code", Type: "textarea", Value: ""},
	}
	for _, s := range defaults {
		db.Where("key = ?", s.Key).FirstOrCreate(&s)
	}
}

var reInternalLink = regexp.MustCompile(`\]\(/(?:[a-z0-9-]+/)?(\d{4})/(\d{1,2})/(\d{1,2})/([^)\s]+)\)`)

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

	err := db.AutoMigrate(&auth.BlogUser{}, &blog.PostType{}, &blog.Post{}, &blog.Tag{}, &auth.AdminUser{}, &blog.Setting{}, &blog.Comment{}, &blog.Backlink{}, &blog.ExternalBacklink{}, &blog.Page{})
	if err != nil {
		log.Println("Error migrating tables: " + err.Error())
		return err
	}

	seedDefaultPostType(db)
	seedDefaultSettings(db)
	seedBacklinks(db)
	seedDefaultPages(db)
	linkWritingPagesToPostType(db)

	return nil
}

// seedDefaultPages creates the default pages (Writing, Research, About) if no pages exist.
func seedDefaultPages(db *gorm.DB) {
	var count int64
	db.Model(&blog.Page{}).Count(&count)
	if count > 0 {
		return
	}

	log.Println("Seeding default pages")

	aboutContent := `I'm currently Principal Software Engineer at a startup working on mobile networks on phones. At the University of Guelph in Canada I hold adjunct Professor status and serve on the committee of several graduate students who are studying wireless networks and occasionally still co-publish research papers.

Prior to this I was a Senior Software Engineer at two different robotics startups in San Francisco (Rapid Robotics and Osaro). I was also the CTO and first developer at a startup in Vancouver called RightMesh which was building a mesh networking library for Android phones. During this time I was also an adjunct professor at the University of Guelph and was the industrial PI of a [$2.13M MITACS grant to improve connectivity in Northern Canada](https://betakit.com/u-of-guelph-left-investing-2-13-million-in-rightmesh-project-improving-northern-connectivity/), specifically Rigolet. RightMesh raised $30M in an ICO in 2018.

Before that I was the CTO of [Redtree Robotics](https://montrealgazette.com/business/local-business/montreal-startup-ecosystem-fertile-playground-for-entrepreneurs) which was working on a robotics hardware software platform to enable plug-and-play swarm robotics. I started this company with a friend during grad school and we raised some seed funding from Real Ventures.

I've won, sponsored, and mentored [hackathons](https://uwaterloo.ca/news/waterloo-student-wins-national-hackathon). I love to give [talks](https://www.bbc.co.uk/programmes/w3csvpcr) and present papers.

I also enjoy driving, working on cars, video games, contributing to [open source](https://github.com/compscidr), cycling, running, and [travel](https://nomadlist.com/@compscidr).

[Tags](/tags) [Archives](/archives)`

	defaults := []blog.Page{
		{
			Title:     "Writing",
			Slug:      "posts",
			HeroURL:   "/vid/redtree.mp4",
			HeroType:  "video",
			PageType:  blog.PageTypeWriting,
			ShowInNav: true,
			NavOrder:  1,
			Enabled:   true,
		},
		{
			Title:     "Research",
			Slug:      "research",
			HeroURL:   "/img/aidecentralized.jpg",
			HeroType:  "image",
			PageType:  blog.PageTypeResearch,
			ShowInNav: true,
			NavOrder:  2,
			Enabled:   true,
			ScholarID: "SbUmSEAAAAAJ",
		},
		{
			Title:     "About",
			Slug:      "about",
			Content:   aboutContent,
			HeroURL:   "/img/hero_rigolet.jpg",
			HeroType:  "image",
			PageType:  blog.PageTypeAbout,
			ShowInNav: true,
			NavOrder:  3,
			Enabled:   true,
		},
		{
			Title:     "Tags",
			Slug:      "tags",
			PageType:  blog.PageTypeTags,
			ShowInNav: false,
			NavOrder:  4,
			Enabled:   true,
		},
		{
			Title:     "Archives",
			Slug:      "archives",
			PageType:  blog.PageTypeArchives,
			ShowInNav: false,
			NavOrder:  5,
			Enabled:   true,
		},
	}
	for _, p := range defaults {
		db.Create(&p)
	}
}

// linkWritingPagesToPostType sets PostTypeID on Writing pages that have it NULL.
// This handles existing databases where pages were created before post types existed.
func linkWritingPagesToPostType(db *gorm.DB) {
	var pages []blog.Page
	if err := db.Where("page_type = ? AND post_type_id IS NULL", blog.PageTypeWriting).Find(&pages).Error; err != nil || len(pages) == 0 {
		return
	}
	for _, page := range pages {
		var pt blog.PostType
		if err := db.Where("slug = ?", page.Slug).First(&pt).Error; err != nil {
			// No matching post type for this page's slug, try the default "posts" type
			if err := db.Where("slug = ?", "posts").First(&pt).Error; err != nil {
				continue
			}
		}
		log.Printf("Linking writing page %q to post type %q", page.Title, pt.Name)
		db.Model(&page).Update("post_type_id", pt.ID)
	}
}

