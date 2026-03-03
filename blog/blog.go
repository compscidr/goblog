package blog

import (
	"errors"
	"fmt"
	"regexp"
	"sort"

	scholar "github.com/compscidr/scholar"
	"goblog/auth"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/ikeikeikeike/go-sitemap-generator/v2/stm"
)

// Blog API handles non-admin functions of the blog like listing posts, tags
// comments, etc.
type Blog struct {
	db             **gorm.DB // needs a double pointer to be able to update the db
	auth           auth.IAuth
	Version        string
	scholar        *scholar.Scholar
	commentLimiter map[string]time.Time
	limiterMu      sync.Mutex
}

// New constructs an Admin API
func New(db *gorm.DB, auth auth.IAuth, version string, scholar *scholar.Scholar) Blog {
	api := Blog{
		db:             &db,
		auth:           auth,
		Version:        version,
		scholar:        scholar,
		commentLimiter: make(map[string]time.Time),
	}
	return api
}

func (b *Blog) UpdateDb(db *gorm.DB) {
	b.db = &db
}

func (b *Blog) IsDbNil() bool {
	return (*b.db) == nil
}

// sortArticlesByDateDesc sorts scholar articles by publication date in descending order.
func sortArticlesByDateDesc(articles []*scholar.Article) {
	sort.Slice(articles, func(i, j int) bool {
		if articles[i].Year != articles[j].Year {
			return articles[i].Year > articles[j].Year
		}
		if articles[i].Month != articles[j].Month {
			return articles[i].Month > articles[j].Month
		}
		return articles[i].Day > articles[j].Day
	})
}

// Generic Functions (not JSON or HTML)
func (b *Blog) GetPosts(drafts bool) []Post {
	var posts []Post
	if !drafts {
		(*b.db).Preload("Tags").Preload("PostType").Order("created_at desc").Find(&posts, "draft = ?", drafts)
	} else {
		(*b.db).Preload("Tags").Preload("PostType").Order("created_at desc").Find(&posts)
	}
	return posts
}

func (b *Blog) GetLatest() Post {
	var post Post
	(*b.db).Preload("Tags").Preload("PostType").Order("created_at desc").First(&post)
	return post
}

func (b *Blog) getTags() []Tag {
	var tags []Tag
	(*b.db).Preload("Posts").Order("name asc").Find(&tags)
	return tags
}

func (b *Blog) getArchivesByYear() map[string][]Post {
	archive := make(map[string][]Post)
	posts := b.GetPosts(false)
	for _, post := range posts {
		year := strconv.Itoa(post.CreatedAt.Year())
		if _, ok := archive[year]; !ok {
			archive[year] = make([]Post, 0)
		}
		archive[year] = append(archive[year], post)
	}
	return archive
}

func (b *Blog) getArchivesByYearMonth() map[string][]Post {
	archive := make(map[string][]Post)
	posts := b.GetPosts(false)
	for _, post := range posts {
		year := strconv.Itoa(post.CreatedAt.Year())
		month := strconv.Itoa(int(post.CreatedAt.Month()))
		yearMonth := year + "/" + month
		if _, ok := archive[yearMonth]; !ok {
			archive[yearMonth] = make([]Post, 0)
		}
		archive[yearMonth] = append(archive[yearMonth], post)
	}
	return archive
}

func (b *Blog) GetPostObject(c *gin.Context) (*Post, error) {
	var post Post
	year, err := strconv.Atoi(c.Param("yyyy"))
	if err != nil {
		return nil, errors.New("year must be an integer")
	}
	month, err := strconv.Atoi(c.Param("mm"))
	if err != nil {
		return nil, errors.New("month must be an integer")
	}
	day, err := strconv.Atoi(c.Param("dd"))
	if err != nil {
		return nil, errors.New("day must be an integer")
	}
	slug := c.Param("slug")
	slug = url.QueryEscape(slug)

	log.Println("Looking for post: ", year, "/", month, "/", day, "/", slug)

	if err := (*b.db).Preload("Tags").Preload("PostType").Where("created_at > ? AND slug LIKE ?", time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), slug).First(&post).Error; err != nil {
		return nil, errors.New("No post at " + strconv.Itoa(year) + "/" + strconv.Itoa(month) + "/" + strconv.Itoa(day) + "/" + slug)
	}

	//b.db.Model(&post).Related(&post.Tags, "Tags")
	log.Println("Found: ", post.Title, " TAGS: ", post.Tags)
	return &post, nil
}

func (b *Blog) getPostByParams(year int, month int, day int, slug string) (*Post, error) {
	log.Println("trying: " + strconv.Itoa(year) + "/" + strconv.Itoa(month) + "/" + strconv.Itoa(day) + "/" + slug)
	var post Post
	slug = url.QueryEscape(slug)
	if err := (*b.db).Preload("Tags").Preload("PostType").Where("created_at > ? AND slug LIKE ?", time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), slug).First(&post).Error; err != nil {
		log.Println("NOT FOUND")
		return nil, errors.New("No post at " + strconv.Itoa(year) + "/" + strconv.Itoa(month) + "/" + strconv.Itoa(day) + "/" + slug)
	}
	log.Println("Found: ", post.Title, " TAGS: ", post.Tags)
	return &post, nil
}

func (b *Blog) getPostsByTag(c *gin.Context) ([]Post, error) {
	var posts []Post
	var tag Tag
	name := strings.TrimPrefix(c.Param("name"), "/")
	if err := (*b.db).Where("name = ?", name).First(&tag).Error; err != nil {
		return nil, errors.New("No tag named " + name)
	}

	(*b.db).Model(&tag).Order("created_at desc").Association("Posts").Find(&posts)
	// Batch-load PostType for all posts to avoid N+1 queries
	if len(posts) > 0 {
		ids := make([]uint, len(posts))
		for i, p := range posts {
			ids[i] = p.ID
		}
		(*b.db).Preload("PostType").Where("id IN ?", ids).Order("created_at desc").Find(&posts)
	}
	log.Print("POSTS: ", posts)
	return posts, nil
}

func (b *Blog) GetSettings() map[string]Setting {
	var settings []Setting
	(*b.db).Find(&settings)

	settingsMap := make(map[string]Setting)
	for _, setting := range settings {
		settingsMap[setting.Key] = setting
	}
	return settingsMap
}

func (b *Blog) SearchPosts(query string) []Post {
	var posts []Post
	escaped := strings.ReplaceAll(query, "!", "!!")
	escaped = strings.ReplaceAll(escaped, "%", "!%")
	escaped = strings.ReplaceAll(escaped, "_", "!_")
	q := "%" + escaped + "%"
	(*b.db).Preload("Tags").Preload("PostType").Where("draft = ? AND (title LIKE ? ESCAPE '!' OR content LIKE ? ESCAPE '!')", false, q, q).Order("created_at desc").Find(&posts)
	return posts
}

// reInternalLink matches internal post URLs like /posts/2024/01/15/my-slug, /notes/2024/01/15/my-slug, or /2024/01/15/my-slug
var reInternalLink = regexp.MustCompile(`\]\(/(?:[a-z0-9-]+/)?(\d{4})/(\d{1,2})/(\d{1,2})/([^)\s]+)\)`)

// ComputeBacklinks parses a post's content for internal links and upserts backlink records.
func (b *Blog) ComputeBacklinks(post *Post) {
	// Clear existing backlinks for this source post
	(*b.db).Where("source_post_id = ?", post.ID).Delete(&Backlink{})

	matches := reInternalLink.FindAllStringSubmatch(post.Content, -1)
	seen := make(map[uint]bool)
	for _, match := range matches {
		year, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		month, err := strconv.Atoi(match[2])
		if err != nil {
			continue
		}
		day, err := strconv.Atoi(match[3])
		if err != nil {
			continue
		}
		slug := match[4]

		// Use exact slug match and bounded date range
		var target Post
		startOfDay := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		endOfDay := startOfDay.Add(24 * time.Hour)
		if err := (*b.db).Preload("Tags").
			Where("slug = ? AND created_at >= ? AND created_at < ?", slug, startOfDay, endOfDay).
			First(&target).Error; err != nil {
			continue
		}
		if target.ID == post.ID || seen[target.ID] {
			continue
		}
		seen[target.ID] = true
		(*b.db).Create(&Backlink{SourcePostID: post.ID, TargetPostID: target.ID})
	}
}

// GetBacklinks returns posts that link TO the given post.
func (b *Blog) GetBacklinks(postID uint) []Post {
	var backlinks []Backlink
	(*b.db).Where("target_post_id = ?", postID).Find(&backlinks)
	if len(backlinks) == 0 {
		return nil
	}
	ids := make([]uint, len(backlinks))
	for i, bl := range backlinks {
		ids[i] = bl.SourcePostID
	}
	var posts []Post
	(*b.db).Preload("PostType").Where("id IN ? AND deleted_at IS NULL", ids).Order("created_at desc").Find(&posts)
	return posts
}

// GetOutboundLinks returns posts that the given post links TO.
func (b *Blog) GetOutboundLinks(postID uint) []Post {
	var backlinks []Backlink
	(*b.db).Where("source_post_id = ?", postID).Find(&backlinks)
	if len(backlinks) == 0 {
		return nil
	}
	ids := make([]uint, len(backlinks))
	for i, bl := range backlinks {
		ids[i] = bl.TargetPostID
	}
	var posts []Post
	(*b.db).Preload("PostType").Where("id IN ? AND deleted_at IS NULL", ids).Order("created_at desc").Find(&posts)
	return posts
}

// GetExternalBacklinks returns external referers for a given post.
func (b *Blog) GetExternalBacklinks(postID uint) []ExternalBacklink {
	var backlinks []ExternalBacklink
	(*b.db).Where("post_id = ?", postID).Order("hit_count desc").Find(&backlinks)
	return backlinks
}

// TrackReferer records external referers for a post.
func (b *Blog) TrackReferer(c *gin.Context, postID uint) {
	referer := c.Request.Referer()
	if referer == "" {
		return
	}

	parsed, err := url.Parse(referer)
	if err != nil {
		return
	}

	// Skip self-referrals (compare hostnames without ports)
	reqHost := c.Request.Host
	if i := strings.LastIndex(reqHost, ":"); i != -1 {
		reqHost = reqHost[:i]
	}
	if strings.EqualFold(parsed.Hostname(), reqHost) {
		return
	}

	// Skip non-HTTP(S) schemes
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return
	}

	now := time.Now()
	var existing ExternalBacklink
	result := (*b.db).Where("post_id = ? AND referer = ?", postID, referer).First(&existing)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			log.Printf("Error querying external backlinks: %v", result.Error)
			return
		}
		// New referer
		(*b.db).Create(&ExternalBacklink{
			PostID:    postID,
			Referer:   referer,
			FirstSeen: now,
			LastSeen:  now,
			HitCount:  1,
		})
	} else {
		(*b.db).Model(&existing).Updates(map[string]interface{}{
			"last_seen": now,
			"hit_count": gorm.Expr("hit_count + ?", 1),
		})
	}
}
// GetNavPages returns enabled pages that should show in the navigation, ordered by nav_order.
func (b *Blog) GetNavPages() []Page {
	var pages []Page
	(*b.db).Where("enabled = ? AND show_in_nav = ?", true, true).Order("nav_order asc").Find(&pages)
	return pages
}

// GetPageBySlug returns a page by its slug. Returns error if not found or disabled.
func (b *Blog) GetPageBySlug(slug string) (*Page, error) {
	var page Page
	if err := (*b.db).Where("slug = ? AND enabled = ?", slug, true).First(&page).Error; err != nil {
		return nil, errors.New("page not found: " + slug)
	}
	return &page, nil
}

// GetPostTypes returns all post types
func (b *Blog) GetPostTypes() []PostType {
	var types []PostType
	(*b.db).Order("name asc").Find(&types)
	return types
}

// GetPostTypeBySlug returns a post type by its slug
func (b *Blog) GetPostTypeBySlug(slug string) (*PostType, error) {
	var pt PostType
	if err := (*b.db).Where("slug = ?", slug).First(&pt).Error; err != nil {
		return nil, errors.New("post type not found: " + slug)
	}
	return &pt, nil
}

// GetPostsByType returns posts filtered by post type
func (b *Blog) GetPostsByType(postTypeID uint, drafts bool) []Post {
	var posts []Post
	if !drafts {
		(*b.db).Preload("Tags").Preload("PostType").Where("post_type_id = ? AND draft = ?", postTypeID, false).Order("created_at desc").Find(&posts)
	} else {
		(*b.db).Preload("Tags").Preload("PostType").Where("post_type_id = ?", postTypeID).Order("created_at desc").Find(&posts)
	}
	return posts
}

// getPostByTypeAndParams finds a post by type slug and date/slug params
func (b *Blog) getPostByTypeAndParams(typeSlug string, year int, month int, day int, slug string) (*Post, error) {
	pt, err := b.GetPostTypeBySlug(typeSlug)
	if err != nil {
		return nil, err
	}
	var post Post
	slug = url.QueryEscape(slug)
	if err := (*b.db).Preload("Tags").Preload("PostType").
		Where("post_type_id = ? AND created_at > ? AND slug LIKE ?", pt.ID, time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), slug).
		First(&post).Error; err != nil {
		return nil, errors.New("No post at " + typeSlug + "/" + strconv.Itoa(year) + "/" + strconv.Itoa(month) + "/" + strconv.Itoa(day) + "/" + slug)
	}
	return &post, nil
}

// PostTypeListing renders the listing page for a post type
func (b *Blog) PostTypeListing(c *gin.Context, pt *PostType) {
	posts := b.GetPostsByType(pt.ID, false)
	c.HTML(http.StatusOK, "post_type_listing.html", gin.H{
		"logged_in":  b.auth.IsLoggedIn(c),
		"is_admin":   b.auth.IsAdmin(c),
		"post_type":  pt,
		"posts":      posts,
		"version":    b.Version,
		"title":      pt.Name,
		"recent":     b.GetLatest(),
		"admin_page": false,
		"settings":   b.GetSettings(),
		"nav_pages":  b.GetNavPages(),
	})
}

// DynamicPage renders the appropriate template for a page based on its PageType.
func (b *Blog) DynamicPage(c *gin.Context, page *Page) {
	navPages := b.GetNavPages()
	switch page.PageType {
	case PageTypeWriting:
		var posts []Post
		if page.PostTypeID != nil {
			posts = b.GetPostsByType(*page.PostTypeID, false)
		} else {
			posts = b.GetPosts(false)
		}
		c.HTML(http.StatusOK, "page_writing.html", gin.H{
			"logged_in":  b.auth.IsLoggedIn(c),
			"is_admin":   b.auth.IsAdmin(c),
			"posts":      posts,
			"page":       page,
			"version":    b.Version,
			"title":      page.Title,
			"recent":     b.GetLatest(),
			"admin_page": false,
			"settings":   b.GetSettings(),
			"nav_pages":  navPages,
		})
	case PageTypeResearch:
		articles, err := b.scholar.QueryProfileWithMemoryCache(page.ScholarID, 50)
		if err == nil {
			sortArticlesByDateDesc(articles)
			b.scholar.SaveCache("profiles.json", "articles.json")
			c.HTML(http.StatusOK, "page_research.html", gin.H{
				"logged_in":  b.auth.IsLoggedIn(c),
				"is_admin":   b.auth.IsAdmin(c),
				"page":       page,
				"articles":   articles,
				"version":    b.Version,
				"title":      page.Title,
				"recent":     b.GetLatest(),
				"admin_page": false,
				"settings":   b.GetSettings(),
				"nav_pages":  navPages,
			})
		} else {
			c.HTML(http.StatusOK, "page_research.html", gin.H{
				"logged_in":  b.auth.IsLoggedIn(c),
				"is_admin":   b.auth.IsAdmin(c),
				"page":       page,
				"articles":   make([]interface{}, 0),
				"version":    b.Version,
				"title":      page.Title,
				"recent":     b.GetLatest(),
				"admin_page": false,
				"settings":   b.GetSettings(),
				"errors":     err.Error(),
				"nav_pages":  navPages,
			})
		}
	case PageTypeTags:
		c.HTML(http.StatusOK, "page_tags.html", gin.H{
			"logged_in":  b.auth.IsLoggedIn(c),
			"is_admin":   b.auth.IsAdmin(c),
			"tags":       b.getTags(),
			"page":       page,
			"version":    b.Version,
			"title":      page.Title,
			"recent":     b.GetLatest(),
			"admin_page": false,
			"settings":   b.GetSettings(),
			"nav_pages":  navPages,
		})
	case PageTypeArchives:
		c.HTML(http.StatusOK, "page_archives.html", gin.H{
			"logged_in":   b.auth.IsLoggedIn(c),
			"is_admin":    b.auth.IsAdmin(c),
			"byYear":      b.getArchivesByYear(),
			"byYearMonth": b.getArchivesByYearMonth(),
			"page":        page,
			"version":     b.Version,
			"title":       page.Title,
			"recent":      b.GetLatest(),
			"admin_page":  false,
			"settings":    b.GetSettings(),
			"nav_pages":   navPages,
		})
	default: // about, custom
		c.HTML(http.StatusOK, "page_content.html", gin.H{
			"logged_in":  b.auth.IsLoggedIn(c),
			"is_admin":   b.auth.IsAdmin(c),
			"page":       page,
			"version":    b.Version,
			"title":      page.Title,
			"recent":     b.GetLatest(),
			"admin_page": false,
			"settings":   b.GetSettings(),
			"nav_pages":  navPages,
		})
	}
}

// renderAdminPost renders the admin edit view for a post, used by NoRoute for admin type-prefixed URLs
func (b *Blog) renderAdminPost(c *gin.Context, post *Post) {
	if !b.auth.IsAdmin(c) {
		c.HTML(http.StatusUnauthorized, "error.html", gin.H{
			"error":       "Unauthorized",
			"description": "You are not authorized to view this page",
			"version":     b.Version,
			"title":       "Unauthorized",
			"recent":      b.GetLatest(),
			"admin_page":  true,
			"settings":    b.GetSettings(),
			"nav_pages":   b.GetNavPages(),
		})
		return
	}
	c.HTML(http.StatusOK, "post-admin.html", gin.H{
		"logged_in":          b.auth.IsLoggedIn(c),
		"is_admin":           b.auth.IsAdmin(c),
		"post":               post,
		"post_types":         b.GetPostTypes(),
		"version":            b.Version,
		"recent":             b.GetLatest(),
		"admin_page":         true,
		"settings":           b.GetSettings(),
		"backlinks":          b.GetBacklinks(post.ID),
		"outbound_links":     b.GetOutboundLinks(post.ID),
		"external_backlinks": b.GetExternalBacklinks(post.ID),
		"nav_pages":          b.GetNavPages(),
	})
}

// renderPost renders a single post page, used by NoRoute handlers
func (b *Blog) renderPost(c *gin.Context, post *Post) {
	b.TrackReferer(c, post.ID)
	if b.auth.IsAdmin(c) {
		c.HTML(http.StatusOK, "post-admin.html", gin.H{
			"logged_in":          b.auth.IsLoggedIn(c),
			"is_admin":           b.auth.IsAdmin(c),
			"post":               post,
			"post_types":         b.GetPostTypes(),
			"version":            b.Version,
			"recent":             b.GetLatest(),
			"admin_page":         false,
			"settings":           b.GetSettings(),
			"comments":           b.getCommentsByPostID(post.ID),
			"comment_error":      c.Query("comment_error"),
			"backlinks":          b.GetBacklinks(post.ID),
			"outbound_links":     b.GetOutboundLinks(post.ID),
			"external_backlinks": b.GetExternalBacklinks(post.ID),
			"nav_pages":          b.GetNavPages(),
		})
	} else {
		c.HTML(http.StatusOK, "post.html", gin.H{
			"logged_in":     b.auth.IsLoggedIn(c),
			"is_admin":      b.auth.IsAdmin(c),
			"post":          post,
			"version":       b.Version,
			"recent":        b.GetLatest(),
			"admin_page":    false,
			"settings":      b.GetSettings(),
			"comments":      b.getCommentsByPostID(post.ID),
			"comment_error": c.Query("comment_error"),
			"nav_pages":     b.GetNavPages(),
		})
	}
}

//////JSON API///////

// ListPosts lists all blog posts
func (b *Blog) ListPosts(c *gin.Context) {
	c.JSON(http.StatusOK, b.GetPosts(false))
}

// GetPost returns a post with yyyy/mm/dd/slug
func (b *Blog) GetPost(c *gin.Context) {
	post, err := b.GetPostObject(c)
	if err != nil {
		log.Println("Bad request in GetPost: " + err.Error())
		c.JSON(http.StatusBadRequest, err)
	}
	if post == nil {
		c.JSON(http.StatusNotFound, "Post Not Found")
	}
	c.JSON(http.StatusOK, post)
}

//////HTML API///////

// NoRoute returns a custom 404 page
func (b *Blog) NoRoute(c *gin.Context) {

	tokens := strings.Split(c.Request.URL.String(), "/")
	// for some reason, first token is empty

	// Try admin type-prefixed post URL: /admin/{type-slug}/{yyyy}/{mm}/{dd}/{post-slug}
	if len(tokens) >= 7 && tokens[1] == "admin" {
		typeSlug := tokens[2]
		year, yerr := strconv.Atoi(tokens[3])
		month, merr := strconv.Atoi(tokens[4])
		day, derr := strconv.Atoi(tokens[5])
		if yerr == nil && merr == nil && derr == nil {
			post, err := b.getPostByTypeAndParams(typeSlug, year, month, day, tokens[6])
			if err == nil && post != nil {
				b.renderAdminPost(c, post)
				return
			}
		}
	}

	// Try type-prefixed post URL: /{type-slug}/{yyyy}/{mm}/{dd}/{post-slug}
	if len(tokens) >= 6 {
		typeSlug := tokens[1]
		year, yerr := strconv.Atoi(tokens[2])
		month, merr := strconv.Atoi(tokens[3])
		day, derr := strconv.Atoi(tokens[4])
		if yerr == nil && merr == nil && derr == nil {
			post, err := b.getPostByTypeAndParams(typeSlug, year, month, day, tokens[5])
			if err == nil && post != nil {
				b.renderPost(c, post)
				return
			}
		}
	}

	// Backward compat: /{yyyy}/{mm}/{dd}/{slug} (any type)
	if len(tokens) >= 5 {
		year, _ := strconv.Atoi(tokens[1])
		month, _ := strconv.Atoi(tokens[2])
		day, _ := strconv.Atoi(tokens[3])
		post, err := b.getPostByParams(year, month, day, tokens[4])
		if err == nil && post != nil {
			b.renderPost(c, post)
			return
		}
	}

	// Try to resolve as a dynamic page or post type listing by slug
	path := strings.TrimPrefix(c.Request.URL.Path, "/")
	path = strings.TrimSuffix(path, "/")
	if path != "" && !strings.Contains(path, "/") {
		// Check dynamic page first (pages have hero content, edit links, etc.)
		page, err := b.GetPageBySlug(path)
		if err == nil && page != nil {
			b.DynamicPage(c, page)
			return
		}

		// Then try post type listing
		pt, err := b.GetPostTypeBySlug(path)
		if err == nil && pt != nil {
			b.PostTypeListing(c, pt)
			return
		}
	}

	c.HTML(http.StatusNotFound, "error.html", gin.H{
		"logged_in":   b.auth.IsLoggedIn(c),
		"is_admin":    b.auth.IsAdmin(c),
		"error":       "404: Page Not Found",
		"description": "The page at '" + c.Request.URL.String() + "' was not found",
		"version":     b.Version,
		"recent":      b.GetLatest(),
		"admin_page":  false,
		"settings":    b.GetSettings(),
		"nav_pages":   b.GetNavPages(),
	})
}

// Home returns html of the home page using the template
// if people want to have different stuff show on the home page they probably
// need to modify this function
func (b *Blog) Home(c *gin.Context) {
	b.checkValidDb(c)
	c.HTML(http.StatusOK, "home.html", gin.H{
		"logged_in":  b.auth.IsLoggedIn(c),
		"is_admin":   b.auth.IsAdmin(c),
		"version":    b.Version,
		"title":      "Software Engineer",
		"recent":     b.GetLatest(),
		"admin_page": false,
		"settings":   b.GetSettings(),
		"nav_pages":  b.GetNavPages(),
	})
}

// Posts is the index page for blog posts
func (b *Blog) Posts(c *gin.Context) {
	c.HTML(http.StatusOK, "posts.html", gin.H{
		"logged_in":  b.auth.IsLoggedIn(c),
		"is_admin":   b.auth.IsAdmin(c),
		"posts":      b.GetPosts(false),
		"version":    b.Version,
		"title":      "Posts",
		"recent":     b.GetLatest(),
		"admin_page": false,
		"settings":   b.GetSettings(),
		"nav_pages":  b.GetNavPages(),
	})
}

// Post is the page for all individual posts
func (b *Blog) Post(c *gin.Context) {
	post, err := b.GetPostObject(c)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error":       "Post Not Found",
			"description": err.Error(),
			"version":     b.Version,
			"title":       "Post Not Found",
			"recent":      b.GetLatest(),
			"admin_page":  false,
			"settings":    b.GetSettings(),
			"nav_pages":   b.GetNavPages(),
		})
	} else {
		b.TrackReferer(c, post.ID)
		data := gin.H{
			"logged_in":     b.auth.IsLoggedIn(c),
			"is_admin":      b.auth.IsAdmin(c),
			"post":          post,
			"version":       b.Version,
			"recent":        b.GetLatest(),
			"admin_page":    false,
			"settings":      b.GetSettings(),
			"comments":      b.getCommentsByPostID(post.ID),
			"comment_error": c.Query("comment_error"),
			"nav_pages":     b.GetNavPages(),
		}
		if b.auth.IsAdmin(c) {
			data["backlinks"] = b.GetBacklinks(post.ID)
			data["outbound_links"] = b.GetOutboundLinks(post.ID)
			data["external_backlinks"] = b.GetExternalBacklinks(post.ID)
			data["post_types"] = b.GetPostTypes()
		}
		c.HTML(http.StatusOK, "post.html", data)
		//if b.auth.IsAdmin(c) {
		//	c.HTML(http.StatusOK, "post-admin.html", gin.H{
		//		"logged_in": b.auth.IsLoggedIn(c),
		//		"is_admin":  b.auth.IsAdmin(c),
		//		"post":      post,
		//		"version":   b.version,
		//	})
		//} else {
		//	c.HTML(http.StatusOK, "post.html", gin.H{
		//		"logged_in": b.auth.IsLoggedIn(c),
		//		"is_admin":  b.auth.IsAdmin(c),
		//		"post":      post,
		//		"version":   b.version,
		//	})
		//}
	}
}

// Search handles the search page
func (b *Blog) Search(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	var posts []Post
	if query != "" {
		posts = b.SearchPosts(query)
	}
	c.HTML(http.StatusOK, "search.html", gin.H{
		"logged_in":  b.auth.IsLoggedIn(c),
		"is_admin":   b.auth.IsAdmin(c),
		"posts":      posts,
		"query":      query,
		"version":    b.Version,
		"title":      "Search",
		"recent":     b.GetLatest(),
		"admin_page": false,
		"settings":   b.GetSettings(),
		"nav_pages":  b.GetNavPages(),
	})
}

// Tag lists all posts with a given tag
func (b *Blog) Tag(c *gin.Context) {
	tag := strings.TrimPrefix(c.Param("name"), "/")
	posts, err := b.getPostsByTag(c)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error":       "Tag '" + tag + "' Not Found",
			"description": err.Error(),
			"version":     b.Version,
			"title":       "Tag '" + tag + "' Not Found",
			"recent":      b.GetLatest(),
			"admin_page":  false,
			"settings":    b.GetSettings(),
			"nav_pages":   b.GetNavPages(),
		})
	} else {
		c.HTML(http.StatusOK, "tag.html", gin.H{
			"logged_in":  b.auth.IsLoggedIn(c),
			"is_admin":   b.auth.IsAdmin(c),
			"posts":      posts,
			"tag":        tag,
			"version":    b.Version,
			"title":      "Posts with Tag '" + tag + "'",
			"recent":     b.GetLatest(),
			"admin_page": false,
			"settings":   b.GetSettings(),
			"nav_pages":  b.GetNavPages(),
		})
	}
}

// Tags is the index page for all Tags
func (b *Blog) Tags(c *gin.Context) {
	c.HTML(http.StatusOK, "tags.html", gin.H{
		"version":    b.Version,
		"title":      "Tags",
		"tags":       b.getTags(),
		"recent":     b.GetLatest(),
		"admin_page": false,
		"settings":   b.GetSettings(),
		"nav_pages":  b.GetNavPages(),
	})
}

// Speaking is the index page for presentations
func (b *Blog) Speaking(c *gin.Context) {
	c.HTML(http.StatusOK, "presentations.html", gin.H{
		"logged_in":  b.auth.IsLoggedIn(c),
		"is_admin":   b.auth.IsAdmin(c),
		"version":    b.Version,
		"title":      "Presentations and Speaking",
		"recent":     b.GetLatest(),
		"admin_page": false,
		"settings":   b.GetSettings(),
		"nav_pages":  b.GetNavPages(),
	})
}

// Speaking is the index page for research publications
func (b *Blog) Research(c *gin.Context) {
	articles, err := b.scholar.QueryProfileWithMemoryCache("SbUmSEAAAAAJ", 50)
	if err == nil {
		sortArticlesByDateDesc(articles)
		b.scholar.SaveCache("profiles.json", "articles.json")
		c.HTML(http.StatusOK, "research.html", gin.H{
			"logged_in":  b.auth.IsLoggedIn(c),
			"is_admin":   b.auth.IsAdmin(c),
			"version":    b.Version,
			"title":      "Research Publications",
			"recent":     b.GetLatest(),
			"articles":   articles,
			"admin_page": false,
			"settings":   b.GetSettings(),
			"nav_pages":  b.GetNavPages(),
		})
	} else {
		articles := make([]*scholar.Article, 0)
		c.HTML(http.StatusOK, "research.html", gin.H{
			"logged_in":  b.auth.IsLoggedIn(c),
			"is_admin":   b.auth.IsAdmin(c),
			"version":    b.Version,
			"title":      "Research Publications",
			"recent":     b.GetLatest(),
			"articles":   articles,
			"admin_page": false,
			"settings":   b.GetSettings(),
			"errors":     err.Error(),
			"nav_pages":  b.GetNavPages(),
		})
	}
}

// Projects is the index page for projects / code
func (b *Blog) Projects(c *gin.Context) {
	c.HTML(http.StatusOK, "projects.html", gin.H{
		"logged_in":  b.auth.IsLoggedIn(c),
		"is_admin":   b.auth.IsAdmin(c),
		"version":    b.Version,
		"title":      "Projects",
		"recent":     b.GetLatest(),
		"admin_page": false,
		"settings":   b.GetSettings(),
		"nav_pages":  b.GetNavPages(),
	})
}

// About is the about page
func (b *Blog) About(c *gin.Context) {
	c.HTML(http.StatusOK, "about.html", gin.H{
		"logged_in":  b.auth.IsLoggedIn(c),
		"is_admin":   b.auth.IsAdmin(c),
		"version":    b.Version,
		"title":      "About",
		"recent":     b.GetLatest(),
		"admin_page": false,
		"settings":   b.GetSettings(),
		"nav_pages":  b.GetNavPages(),
	})
}

// Archives shows the posts by year, month, etc.
func (b *Blog) Archives(c *gin.Context) {
	c.HTML(http.StatusOK, "archives.html", gin.H{
		"logged_in":   b.auth.IsLoggedIn(c),
		"is_admin":    b.auth.IsAdmin(c),
		"version":     b.Version,
		"title":       "Blog Archives",
		"byYear":      b.getArchivesByYear(),
		"byYearMonth": b.getArchivesByYearMonth(),
		"recent":      b.GetLatest(),
		"admin_page":  false,
		"settings":    b.GetSettings(),
		"nav_pages":   b.GetNavPages(),
	})
}

func (b *Blog) Sitemap(c *gin.Context) {
	sm := stm.NewSitemap(1)
	sm.SetDefaultHost("https://www.jasonernst.com")
	sm.Create()

	sm.Add(stm.URL{{"loc", "/"}, {"changefreq", "weekly"}, {"priority", 1.0}})
	sm.Add(stm.URL{{"loc", "/archives"}, {"changefreq", "weekly"}, {"priority", 0.8}})
	sm.Add(stm.URL{{"loc", "/tags"}, {"changefreq", "weekly"}, {"priority", 0.8}})

	// Add enabled pages to sitemap
	navPages := b.GetNavPages()
	for _, page := range navPages {
		sm.Add(stm.URL{{"loc", page.PagePermalink()}, {"changefreq", "weekly"}, {"priority", 0.7}})
	}

	// Add post type listing URLs
	postTypes := b.GetPostTypes()
	for _, pt := range postTypes {
		sm.Add(stm.URL{{"loc", pt.Permalink()}, {"changefreq", "weekly"}, {"priority", 0.7}})
	}

	posts := b.GetPosts(false)
	for _, post := range posts {
		if !post.Draft {
			sm.Add(stm.URL{{"loc", post.Permalink()}, {"changefreq", "yearly"}, {"priority", 0.55}})
		}
	}
	tags := b.getTags()
	for _, tag := range tags {
		if len(tag.Posts) > 0 {
			sm.Add(stm.URL{{"loc", tag.Permalink()}, {"changefreq", "weekly"}, {"priority", 0.55}})
		}
	}

	c.Data(http.StatusOK, "text/xml", sm.XMLContent())
}

// Login to the blog
func (b *Blog) Login(c *gin.Context) {
	err := godotenv.Load(".env")
	if err != nil {
		//fall back to local config
		err = godotenv.Load("local.env")
		if err != nil {
			//todo: handle better - perhaps return error to browser
			c.HTML(http.StatusInternalServerError, "Error loading .env file: "+err.Error(), gin.H{
				"logged_in":  b.auth.IsLoggedIn(c),
				"is_admin":   b.auth.IsAdmin(c),
				"version":    b.Version,
				"title":      "Login Configuration Error",
				"recent":     b.GetLatest(),
				"admin_page": false,
				"settings":   b.GetSettings(),
				"nav_pages":  b.GetNavPages(),
			})
			return
		}
	}

	clientID := os.Getenv("client_id")
	c.HTML(http.StatusOK, "login.html", gin.H{
		"logged_in":  b.auth.IsLoggedIn(c),
		"is_admin":   b.auth.IsAdmin(c),
		"client_id":  clientID,
		"version":    b.Version,
		"title":      "Login",
		"recent":     b.GetLatest(),
		"admin_page": false,
		"settings":   b.GetSettings(),
		"nav_pages":  b.GetNavPages(),
	})
}

// Logout of the blog
func (b *Blog) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("token")
	session.Save()
	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (b *Blog) canComment(ip string) bool {
	b.limiterMu.Lock()
	defer b.limiterMu.Unlock()
	if last, ok := b.commentLimiter[ip]; ok {
		if time.Since(last) < time.Minute {
			return false
		}
	}
	return true
}

func (b *Blog) recordComment(ip string) {
	b.limiterMu.Lock()
	defer b.limiterMu.Unlock()
	b.commentLimiter[ip] = time.Now()
}

func (b *Blog) getCommentsByPostID(postID uint) []Comment {
	var comments []Comment
	(*b.db).Where("post_id = ?", postID).Order("created_at asc").Find(&comments)
	return comments
}

// GetRecentComments returns the most recent comments across all posts
func (b *Blog) GetRecentComments(limit int) []Comment {
	var comments []Comment
	(*b.db).Order("created_at desc").Limit(limit).Find(&comments)
	return comments
}

// GetPostsByIDs returns a map of post ID to Post for the given IDs
func (b *Blog) GetPostsByIDs(ids []uint) map[uint]Post {
	result := make(map[uint]Post)
	if len(ids) == 0 {
		return result
	}
	var posts []Post
	(*b.db).Preload("PostType").Where("id IN ?", ids).Find(&posts)
	for _, p := range posts {
		result[p.ID] = p
	}
	return result
}

// SubmitComment handles POST /comments form submissions
func (b *Blog) SubmitComment(c *gin.Context) {
	redirect := c.PostForm("redirect")
	if redirect == "" {
		redirect = "/"
	}

	// Honeypot check - if website field is filled, silently redirect
	if c.PostForm("website") != "" {
		c.Redirect(http.StatusSeeOther, redirect)
		return
	}

	postIDStr := c.PostForm("post_id")
	name := strings.TrimSpace(c.PostForm("name"))
	email := strings.TrimSpace(c.PostForm("email"))
	content := strings.TrimSpace(c.PostForm("content"))

	postID, err := strconv.ParseUint(postIDStr, 10, 64)
	if err != nil || postID == 0 {
		c.Redirect(http.StatusSeeOther, redirect+"?comment_error=invalid_post")
		return
	}

	if name == "" || content == "" {
		c.Redirect(http.StatusSeeOther, redirect+"?comment_error=missing_fields")
		return
	}

	if len(name) > 100 {
		c.Redirect(http.StatusSeeOther, redirect+"?comment_error=name_too_long")
		return
	}
	if len(email) > 254 {
		c.Redirect(http.StatusSeeOther, redirect+"?comment_error=email_too_long")
		return
	}
	if len(content) > 5000 {
		c.Redirect(http.StatusSeeOther, redirect+"?comment_error=content_too_long")
		return
	}

	ip := c.ClientIP()
	if !b.canComment(ip) {
		c.Redirect(http.StatusSeeOther, redirect+"?comment_error=rate_limit")
		return
	}

	comment := Comment{
		PostID:    uint(postID),
		Name:      name,
		Email:     email,
		Content:   content,
		IPAddress: ip,
	}
	(*b.db).Create(&comment)
	b.recordComment(ip)

	c.Redirect(http.StatusSeeOther, redirect+fmt.Sprintf("#comment-%d", comment.ID))
}

func (b *Blog) checkValidDb(c *gin.Context) {
	if b.db == nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error":       "Database Not Found",
			"description": "Database is not connected",
			"version":     b.Version,
			"title":       "Database Not Found",
			"admin_page":  false,
			"settings":    b.GetSettings(),
		})
	}
}
