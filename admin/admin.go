package admin

import (
	"fmt"
	"goblog/auth"
	"goblog/blog"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// WWWFolder and UploadsFolder configures where the file uploads should be stored. This is
// mostly used for testing
var WWWFolder = "www/"
var UploadsFolder = "uploads/"

// Admin handles admin requests
type Admin struct {
	db      **gorm.DB // needs a double pointer to be able to update the db
	auth    auth.IAuth
	b       *blog.Blog
	version string
}

// New constructs an Admin API
func New(db *gorm.DB, auth auth.IAuth, b *blog.Blog, version string) Admin {
	api := Admin{&db, auth, b, version}
	return api
}

func (a *Admin) UpdateDb(db *gorm.DB) {
	a.db = &db
}

// ////JSON API///////
func safeSlug(slug string) string {
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "/", "")
	slug = strings.ReplaceAll(slug, ".", "-")
	return url.QueryEscape(slug)
}

// CreatePost adds a post if the user has permission
func (a *Admin) CreatePost(c *gin.Context) {
	contentType := c.Request.Header.Get("content-type")
	if contentType != "application/json" {
		c.JSON(http.StatusUnsupportedMediaType, "Expecting application/json")
		return
	}

	if !a.auth.IsAdmin(c) {
		log.Println("IS ADMIN RETURNED FALSE")
		c.JSON(http.StatusUnauthorized, "Not Authorized")
		return
	}

	var requestPost blog.Post
	err := c.BindJSON(&requestPost)
	if err != nil {
		log.Println("MALFORMED REQ: " + err.Error())
		c.JSON(http.StatusBadRequest, "Malformed request")
		return
	}

	if requestPost.Title == "" || requestPost.Content == "" {
		c.JSON(http.StatusBadRequest, "Missing Title or Content")
		return
	}

	//todo: make tags work - need to get the relations working
	requestPost.Slug = safeSlug(requestPost.Title)
	log.Print("CREATING POST: ", requestPost)
	(*a.db).Create(&requestPost)

	a.b.ComputeBacklinks(&requestPost)

	log.Println("POST CREATED: ", requestPost)
	c.JSON(http.StatusCreated, requestPost)
}

// UploadFile is the endpoint for storing files on the server
// https://github.com/gin-gonic/examples/blob/master/upload-file/single/main.go
func (a *Admin) UploadFile(c *gin.Context) {
	log.Println("Upload file API hit")

	if !a.auth.IsAdmin(c) {
		log.Println("IS ADMIN RETURNED FALSE")
		c.JSON(http.StatusUnauthorized, "Not Authorized")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		log.Println(fmt.Sprintf("FormFile erorr: %s", err.Error()))
		c.JSON(http.StatusBadRequest, fmt.Sprintf("get form err: %s", err.Error()))
		return
	}

	filename := UploadsFolder + filepath.Base(file.Filename)
	if err := c.SaveUploadedFile(file, WWWFolder+filename); err != nil {
		log.Println(fmt.Sprintf("Save Upload File Error erorr: %s", err.Error()))
		c.JSON(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()))
		return
	}

	log.Println("Saved file okay: " + filename)

	c.JSON(http.StatusOK, map[string]interface{}{"filename": "/" + filename})
}

// UpdatePost modifies an existing post
// Requires the ID of the post, title and content to not be empty
func (a *Admin) UpdatePost(c *gin.Context) {
	contentType := c.Request.Header.Get("content-type")
	if contentType != "application/json" {
		c.JSON(http.StatusUnsupportedMediaType, "Expecting application/json")
		return
	}

	if !a.auth.IsAdmin(c) {
		log.Println("IS ADMIN RETURNED FALSE")
		c.JSON(http.StatusUnauthorized, "Not Authorized")
		return
	}

	var requestPost blog.Post
	e := c.BindJSON(&requestPost)
	if e != nil {
		log.Println("MALFORMED REQUEST: " + e.Error())
		c.JSON(http.StatusBadRequest, "Malformed request, missing some information")
		return
	}
	log.Println("REQUEST POST: ", requestPost)

	if requestPost.Title == "" || requestPost.Content == "" || requestPost.ID < 0 {
		c.JSON(http.StatusBadRequest, "Missing ID, Title or Content")
		return
	}
	requestPost.Slug = safeSlug(requestPost.Slug)

	var existingPost blog.Post
	err := (*a.db).Where("id = ?", requestPost.ID).First(&existingPost).Error

	if err != nil {
		c.JSON(http.StatusBadRequest, "Existing post with ID "+fmt.Sprint(requestPost.ID)+" not found")
		return
	}

	//clear old associations
	(*a.db).Model(&existingPost).Association("Tags").Clear()

	log.Println("UPDATING DRAFT AS: ", requestPost.Draft)

	existingPost.Title = requestPost.Title
	existingPost.Content = requestPost.Content
	existingPost.Slug = requestPost.Slug
	existingPost.Tags = requestPost.Tags
	existingPost.CreatedAt = requestPost.CreatedAt
	existingPost.Draft = requestPost.Draft
	(*a.db).Model(&existingPost).Where("id = ?", requestPost.ID).Updates(&existingPost)

	//https://stackoverflow.com/questions/56653423/gorm-doesnt-update-boolean-field-to-false
	if !requestPost.Draft {
		(*a.db).Model(&existingPost).Select("draft").Update("draft", false)
	}

	a.b.ComputeBacklinks(&existingPost)

	log.Println("POST UPDATED: ", existingPost)
	c.JSON(http.StatusAccepted, existingPost)
}

// DeletePost deletes a post from the database
func (a *Admin) DeletePost(c *gin.Context) {
	contentType := c.Request.Header.Get("content-type")
	if contentType != "application/json" {
		c.JSON(http.StatusUnsupportedMediaType, "Expecting application/json")
		return
	}

	if !a.auth.IsAdmin(c) {
		log.Println("IS ADMIN RETURNED FALSE")
		c.JSON(http.StatusUnauthorized, "Not Authorized")
		return
	}

	var requestPost blog.Post
	if c.BindJSON(&requestPost) != nil {
		c.JSON(http.StatusBadRequest, "Malformed request, missing some information")
		return
	}

	(*a.db).Where("id = ?", requestPost.ID).Delete(&blog.Post{})

	c.JSON(http.StatusOK, "")
}

// DeleteComment deletes a comment from the database
func (a *Admin) DeleteComment(c *gin.Context) {
	contentType := c.Request.Header.Get("content-type")
	if contentType != "application/json" {
		c.JSON(http.StatusUnsupportedMediaType, "Expecting application/json")
		return
	}

	if !a.auth.IsAdmin(c) {
		c.JSON(http.StatusUnauthorized, "Not Authorized")
		return
	}

	var requestComment blog.Comment
	if c.BindJSON(&requestComment) != nil {
		c.JSON(http.StatusBadRequest, "Malformed request, missing some information")
		return
	}

	(*a.db).Where("id = ?", requestComment.ID).Delete(&blog.Comment{})

	c.JSON(http.StatusOK, "")
}

func (a *Admin) PublishPost(c *gin.Context) {
	id := c.Param("id")
	log.Println("Publishing post: ", id)

	var post blog.Post
	(*a.db).Where("id = ?", id).First(&post)
	(*a.db).Model(&post).Select("draft").Update("draft", false)
	c.JSON(http.StatusAccepted, post)
}

func (a *Admin) DraftPost(c *gin.Context) {
	id := c.Param("id")
	log.Println("Drafting post: ", id)
	var post blog.Post
	(*a.db).Where("id = ?", id).First(&post)
	(*a.db).Model(&post).Select("draft").Update("draft", true)
	c.JSON(http.StatusAccepted, post)
}

func (a *Admin) AddSetting(c *gin.Context) {
	contentType := c.Request.Header.Get("content-type")
	if contentType != "application/json" {
		c.JSON(http.StatusUnsupportedMediaType, "Expecting application/json")
		return
	}

	if !a.auth.IsAdmin(c) {
		log.Println("IS ADMIN RETURNED FALSE")
		c.JSON(http.StatusUnauthorized, "Not Authorized")
		return
	}

	var requestSetting blog.Setting
	err := c.BindJSON(&requestSetting)
	if err != nil {
		log.Println("MALFORMED REQ: " + err.Error())
		c.JSON(http.StatusBadRequest, "Malformed request")
		return
	}

	if requestSetting.Key == "" {
		c.JSON(http.StatusBadRequest, "Missing Key Name for Setting")
		return
	}

	log.Print("CREATING SETTING: ", requestSetting)
	(*a.db).Create(&requestSetting)

	log.Println("SETTING CREATED: ", requestSetting)
	c.JSON(http.StatusCreated, requestSetting)
}

func (a *Admin) UpdateSettings(c *gin.Context) {
	contentType := c.Request.Header.Get("content-type")
	if contentType != "application/json" {
		c.JSON(http.StatusUnsupportedMediaType, "Expecting application/json")
		return
	}

	if !a.auth.IsAdmin(c) {
		log.Println("IS ADMIN RETURNED FALSE")
		c.JSON(http.StatusUnauthorized, "Not Authorized")
		return
	}

	var requestSettings []blog.Setting
	err := c.BindJSON(&requestSettings)
	if err != nil {
		log.Println("MALFORMED REQ: " + err.Error())
		c.JSON(http.StatusBadRequest, "Malformed request")
		return
	}

	for _, setting := range requestSettings {
		if setting.Key == "" {
			c.JSON(http.StatusBadRequest, "Missing Key Name for Setting")
			return
		}
		(*a.db).Save(&setting)
	}
	c.JSON(http.StatusAccepted, requestSettings)
}

func (a *Admin) GetSetting(c *gin.Context) {
	if !a.auth.IsAdmin(c) {
		log.Println("IS ADMIN RETURNED FALSE")
		c.JSON(http.StatusUnauthorized, "Not Authorized")
		return
	}

	key := c.Param("key")
	log.Println("Getting setting: ", key)

	err := (*a.db).Where("key = ?", key).First(&blog.Setting{}).Error
	if err != nil {
		c.JSON(http.StatusNotFound, "Setting not found")
	} else {
		c.JSON(http.StatusOK, blog.Setting{})
	}
}

func (a *Admin) GetSettings(c *gin.Context) {
	if !a.auth.IsAdmin(c) {
		log.Println("IS ADMIN RETURNED FALSE")
		c.JSON(http.StatusUnauthorized, "Not Authorized")
		return
	}

	settingsMap := a.b.GetSettings()
	c.JSON(http.StatusOK, settingsMap)
}

//////HTML API///////

// Admin is the admin dashboard of the website
func (a *Admin) Admin(c *gin.Context) {
	c.HTML(http.StatusOK, "admin.html", gin.H{
		"posts":      a.b.GetPosts(true),
		"logged_in":  a.auth.IsLoggedIn(c),
		"is_admin":   a.auth.IsAdmin(c),
		"version":    a.version,
		"recent":     a.b.GetLatest(),
		"admin_page": true,
		"settings":   a.b.GetSettings(),
		"nav_pages":  a.b.GetNavPages(),
	})
}

func (a *Admin) AdminDashboard(c *gin.Context) {
	recentComments := a.b.GetRecentComments(10)
	var postIDs []uint
	for _, comment := range recentComments {
		postIDs = append(postIDs, comment.PostID)
	}
	commentPosts := a.b.GetPostsByIDs(postIDs)
	c.HTML(http.StatusOK, "admin_dashboard.html", gin.H{
		"posts":           a.b.GetPosts(true),
		"logged_in":       a.auth.IsLoggedIn(c),
		"is_admin":        a.auth.IsAdmin(c),
		"version":         a.version,
		"recent":          a.b.GetLatest(),
		"admin_page":      true,
		"settings":        a.b.GetSettings(),
		"recent_comments": recentComments,
		"comment_posts":   commentPosts,
		"nav_pages":       a.b.GetNavPages(),
	})
}

func (a *Admin) AdminPosts(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_all_posts.html", gin.H{
		"posts":      a.b.GetPosts(true),
		"logged_in":  a.auth.IsLoggedIn(c),
		"is_admin":   a.auth.IsAdmin(c),
		"version":    a.version,
		"recent":     a.b.GetLatest(),
		"admin_page": true,
		"settings":   a.b.GetSettings(),
		"nav_pages":  a.b.GetNavPages(),
	})
}

func (a *Admin) AdminNewPost(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_new_post.html", gin.H{
		"posts":      a.b.GetPosts(true),
		"logged_in":  a.auth.IsLoggedIn(c),
		"is_admin":   a.auth.IsAdmin(c),
		"version":    a.version,
		"recent":     a.b.GetLatest(),
		"admin_page": true,
		"settings":   a.b.GetSettings(),
		"nav_pages":  a.b.GetNavPages(),
	})
}

func (a *Admin) AdminSettings(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_settings.html", gin.H{
		"posts":      a.b.GetPosts(true),
		"logged_in":  a.auth.IsLoggedIn(c),
		"is_admin":   a.auth.IsAdmin(c),
		"version":    a.version,
		"recent":     a.b.GetLatest(),
		"admin_page": true,
		"settings":   a.b.GetSettings(),
		"nav_pages":  a.b.GetNavPages(),
	})
}

// Page CRUD API handlers

var reservedSlugs = map[string]bool{
	"admin": true, "api": true, "login": true, "logout": true,
	"wizard": true, "search": true, "tag": true, "tags": true,
	"archives": true, "sitemap.xml": true, "comments": true,
	"wp-content": true, "index.php": true,
}

var reSlugValid = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

func sanitizeSlug(slug string) string {
	slug = strings.ToLower(strings.TrimSpace(slug))
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "/", "")
	return slug
}

func isReservedSlug(slug string) bool {
	return reservedSlugs[slug]
}

// ListPages returns all pages as JSON
func (a *Admin) ListPages(c *gin.Context) {
	if !a.auth.IsAdmin(c) {
		c.JSON(http.StatusUnauthorized, "Not Authorized")
		return
	}

	var pages []blog.Page
	(*a.db).Order("nav_order asc").Find(&pages)
	c.JSON(http.StatusOK, pages)
}

// CreatePage creates a new page
func (a *Admin) CreatePage(c *gin.Context) {
	contentType := c.Request.Header.Get("content-type")
	if contentType != "application/json" {
		c.JSON(http.StatusUnsupportedMediaType, "Expecting application/json")
		return
	}

	if !a.auth.IsAdmin(c) {
		c.JSON(http.StatusUnauthorized, "Not Authorized")
		return
	}

	var page blog.Page
	if err := c.BindJSON(&page); err != nil {
		c.JSON(http.StatusBadRequest, "Malformed request")
		return
	}

	if page.Title == "" {
		c.JSON(http.StatusBadRequest, "Missing Title")
		return
	}

	page.Slug = sanitizeSlug(page.Slug)
	if page.Slug == "" {
		c.JSON(http.StatusBadRequest, "Missing or invalid Slug")
		return
	}

	if !reSlugValid.MatchString(page.Slug) {
		c.JSON(http.StatusBadRequest, "Slug contains invalid characters")
		return
	}

	if isReservedSlug(page.Slug) {
		c.JSON(http.StatusBadRequest, "Slug '"+page.Slug+"' is reserved")
		return
	}

	// Check uniqueness
	var existing blog.Page
	if err := (*a.db).Where("slug = ?", page.Slug).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, "A page with slug '"+page.Slug+"' already exists")
		return
	}

	(*a.db).Create(&page)
	c.JSON(http.StatusCreated, page)
}

// UpdatePage updates an existing page
func (a *Admin) UpdatePage(c *gin.Context) {
	contentType := c.Request.Header.Get("content-type")
	if contentType != "application/json" {
		c.JSON(http.StatusUnsupportedMediaType, "Expecting application/json")
		return
	}

	if !a.auth.IsAdmin(c) {
		c.JSON(http.StatusUnauthorized, "Not Authorized")
		return
	}

	var requestPage blog.Page
	if err := c.BindJSON(&requestPage); err != nil {
		c.JSON(http.StatusBadRequest, "Malformed request")
		return
	}

	if requestPage.ID == 0 {
		c.JSON(http.StatusBadRequest, "Missing page ID")
		return
	}

	var existingPage blog.Page
	if err := (*a.db).Where("id = ?", requestPage.ID).First(&existingPage).Error; err != nil {
		c.JSON(http.StatusNotFound, "Page not found")
		return
	}

	requestPage.Slug = sanitizeSlug(requestPage.Slug)
	if requestPage.Slug == "" {
		c.JSON(http.StatusBadRequest, "Missing or invalid Slug")
		return
	}

	if isReservedSlug(requestPage.Slug) {
		c.JSON(http.StatusBadRequest, "Slug '"+requestPage.Slug+"' is reserved")
		return
	}

	// Check slug uniqueness (excluding current page)
	var conflict blog.Page
	if err := (*a.db).Where("slug = ? AND id != ?", requestPage.Slug, requestPage.ID).First(&conflict).Error; err == nil {
		c.JSON(http.StatusConflict, "A page with slug '"+requestPage.Slug+"' already exists")
		return
	}

	// Update fields
	existingPage.Title = requestPage.Title
	existingPage.Slug = requestPage.Slug
	existingPage.Content = requestPage.Content
	existingPage.HeroURL = requestPage.HeroURL
	existingPage.HeroType = requestPage.HeroType
	existingPage.PageType = requestPage.PageType
	existingPage.NavOrder = requestPage.NavOrder
	existingPage.ScholarID = requestPage.ScholarID

	(*a.db).Model(&existingPage).Updates(&existingPage)

	// Handle GORM zero-value booleans
	(*a.db).Model(&existingPage).Select("show_in_nav").Update("show_in_nav", requestPage.ShowInNav)
	(*a.db).Model(&existingPage).Select("enabled").Update("enabled", requestPage.Enabled)

	c.JSON(http.StatusAccepted, existingPage)
}

// DeletePage deletes a page
func (a *Admin) DeletePage(c *gin.Context) {
	contentType := c.Request.Header.Get("content-type")
	if contentType != "application/json" {
		c.JSON(http.StatusUnsupportedMediaType, "Expecting application/json")
		return
	}

	if !a.auth.IsAdmin(c) {
		c.JSON(http.StatusUnauthorized, "Not Authorized")
		return
	}

	var requestPage blog.Page
	if err := c.BindJSON(&requestPage); err != nil {
		c.JSON(http.StatusBadRequest, "Malformed request")
		return
	}

	(*a.db).Where("id = ?", requestPage.ID).Delete(&blog.Page{})
	c.JSON(http.StatusOK, "")
}

// AdminPages renders the admin page listing
func (a *Admin) AdminPages(c *gin.Context) {
	var pages []blog.Page
	(*a.db).Order("nav_order asc").Find(&pages)
	c.HTML(http.StatusOK, "admin_pages.html", gin.H{
		"pages":      pages,
		"logged_in":  a.auth.IsLoggedIn(c),
		"is_admin":   a.auth.IsAdmin(c),
		"version":    a.version,
		"recent":     a.b.GetLatest(),
		"admin_page": true,
		"settings":   a.b.GetSettings(),
		"nav_pages":  a.b.GetNavPages(),
	})
}

// AdminEditPage renders the form to edit a single page
func (a *Admin) AdminEditPage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error":       "Invalid page ID",
			"description": "Page ID must be a number",
			"version":     a.version,
			"admin_page":  true,
			"settings":    a.b.GetSettings(),
			"nav_pages":   a.b.GetNavPages(),
		})
		return
	}

	var page blog.Page
	if err := (*a.db).Where("id = ?", id).First(&page).Error; err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error":       "Page Not Found",
			"description": "No page with ID " + idStr,
			"version":     a.version,
			"admin_page":  true,
			"settings":    a.b.GetSettings(),
			"nav_pages":   a.b.GetNavPages(),
		})
		return
	}

	c.HTML(http.StatusOK, "admin_edit_page.html", gin.H{
		"page":       page,
		"logged_in":  a.auth.IsLoggedIn(c),
		"is_admin":   a.auth.IsAdmin(c),
		"version":    a.version,
		"recent":     a.b.GetLatest(),
		"admin_page": true,
		"settings":   a.b.GetSettings(),
		"nav_pages":  a.b.GetNavPages(),
	})
}

func (a *Admin) Post(c *gin.Context) {
	if !a.auth.IsAdmin(c) {
		log.Println("IS ADMIN RETURNED FALSE")
		c.JSON(http.StatusUnauthorized, "Not Authorized")
		return
	}

	post, err := a.b.GetPostObject(c)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error":       "Post Not Found",
			"description": err.Error(),
			"version":     a.b.Version,
			"title":       "Post Not Found",
			"recent":      a.b.GetLatest(),
			"admin_page":  true,
			"settings":    a.b.GetSettings(),
			"nav_pages":   a.b.GetNavPages(),
		})
	} else {
		c.HTML(http.StatusOK, "post-admin.html", gin.H{
			"logged_in":          a.auth.IsLoggedIn(c),
			"is_admin":           a.auth.IsAdmin(c),
			"post":               post,
			"version":            a.b.Version,
			"recent":             a.b.GetLatest(),
			"admin_page":         true,
			"settings":           a.b.GetSettings(),
			"backlinks":          a.b.GetBacklinks(post.ID),
			"outbound_links":     a.b.GetOutboundLinks(post.ID),
			"external_backlinks": a.b.GetExternalBacklinks(post.ID),
			"nav_pages":          a.b.GetNavPages(),
		})
	}
}
