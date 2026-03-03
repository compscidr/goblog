package blog_test

import (
	"bytes"
	"encoding/json"
	scholar "github.com/compscidr/scholar"
	"goblog/admin"
	"goblog/auth"
	"goblog/blog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Auth struct {
	mock.Mock
}

func (m *Auth) IsAdmin(c *gin.Context) bool {
	args := m.Called(c)
	return args.Bool(0)
}

func (m *Auth) IsLoggedIn(c *gin.Context) bool {
	args := m.Called(c)
	return args.Bool(0)
}

func TestBlogWorkflow(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"))
	db.AutoMigrate(&auth.BlogUser{})
	db.AutoMigrate(&blog.Post{})
	db.AutoMigrate(&blog.Tag{})
	db.AutoMigrate(&blog.Comment{})
	a := &Auth{}
	sch := scholar.New("profiles.json", "articles.json")
	b := blog.New(db, a, "test", sch)
	admin := admin.New(db, a, &b, "test")

	router := gin.Default()
	store := cookie.NewStore([]byte("changelater"))
	router.Use(sessions.Sessions("www.jasonernst.com", store))

	//json requests
	router.POST("/api/v1/posts", admin.CreatePost)
	router.GET("/api/v1/posts", b.ListPosts)
	router.GET("/api/v1/posts/:yyyy/:mm/:dd/:slug", b.GetPost)

	//html requests
	router.GET("/posts/:yyyy/:mm/:dd/:slug", b.Post)
	router.POST("/comments", b.SubmitComment)
	router.GET("/tag/*name", b.Tag)
	router.GET("/posts", b.Posts)
	router.GET("/tags", b.Tags)
	router.GET("/", b.Home)
	router.NoRoute(b.NoRoute)
	router.GET("/presentations", b.Speaking)
	router.GET("/projects", b.Projects)
	router.GET("/about", b.About)

	router.GET("/search", b.Search)
	router.GET("/login", b.Login)
	router.GET("/logout", b.Logout)

	//list all posts, should be empty
	jsonValue, _ := json.Marshal("")
	req, _ := http.NewRequest("GET", "/api/v1/posts", bytes.NewBuffer(jsonValue))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	expected := `[]`
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}
	if w.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", w.Body.String(), expected)
	}

	//create valid post
	testTag := blog.Tag{
		Name: "test",
	}
	testPost := blog.Post{
		Title:   "Test title",
		Content: "This is some test content",
		Tags:    []blog.Tag{testTag},
	}
	jsonValue, _ = json.Marshal(testPost)
	req, _ = http.NewRequest("POST", "/api/v1/posts", bytes.NewBuffer(jsonValue))
	req.Header.Add("Content-Type", "application/json")
	a.On("IsAdmin", mock.Anything).Return(true).Once()
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusCreated, w.Code)
	}

	//list all posts, should not be empty
	jsonValue, _ = json.Marshal("")
	req, _ = http.NewRequest("GET", "/api/v1/posts", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), testPost.Title) {
		t.Errorf("Expected to see a post with title: %s but didn't", testPost.Title)
	}

	//get specific post
	var posts []blog.Post
	err := json.Unmarshal(w.Body.Bytes(), &posts)
	if err != nil {
		t.Errorf("Couldn't parse the posts")
	}
	post := posts[0]
	jsonValue, _ = json.Marshal(post)
	req, _ = http.NewRequest("GET", "/api/v1/posts/"+strconv.Itoa(post.CreatedAt.Year())+"/"+strconv.Itoa(int(post.CreatedAt.Month()))+"/"+strconv.Itoa(post.CreatedAt.Day())+"/"+post.Slug, bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), testPost.Title) {
		t.Errorf("Expected to see a post with title: %s but didn't", testPost.Title)
	}

	//bad year
	jsonValue, _ = json.Marshal("")
	req, _ = http.NewRequest("GET", "/api/v1/posts/zfaq/12/12/slug", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusBadRequest, w.Code)
	}

	//bad Month
	jsonValue, _ = json.Marshal("")
	req, _ = http.NewRequest("GET", "/api/v1/posts/2020/zq/12/slug", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusBadRequest, w.Code)
	}

	//bad day
	jsonValue, _ = json.Marshal("")
	req, _ = http.NewRequest("GET", "/api/v1/posts/2020/12/qf/slug", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusBadRequest, w.Code)
	}

	//everything good but non-existant
	jsonValue, _ = json.Marshal("")
	req, _ = http.NewRequest("GET", "/api/v1/posts/2020/12/12/slug", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusBadRequest, w.Code)
	}

	//html tests

	//get tag
	router.LoadHTMLGlob("../templates/*")
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	a.On("IsLoggedIn", mock.Anything).Return(false)
	jsonValue, _ = json.Marshal("")
	req, _ = http.NewRequest("GET", "/tag/test", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	//get not found tag
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	req, _ = http.NewRequest("GET", "/tag/blah", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusNotFound, w.Code)
	}

	//get all tags
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	req, _ = http.NewRequest("GET", "/tags", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	//get all posts
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	req, _ = http.NewRequest("GET", "/posts", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	//get home
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	req, _ = http.NewRequest("GET", "/", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	//no route
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	req, _ = http.NewRequest("GET", "/dfadfasdf", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusNotFound, w.Code)
	}

	//html post as normal user
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	req, _ = http.NewRequest("GET", "/posts/"+strconv.Itoa(post.CreatedAt.Year())+"/"+strconv.Itoa(int(post.CreatedAt.Month()))+"/"+strconv.Itoa(post.CreatedAt.Day())+"/"+post.Slug, bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}
	if !strings.Contains(w.Body.String(), testPost.Title) {
		t.Errorf("Expected to see a post with title: %s but didn't", testPost.Title)
	}

	//html post as admin - TODO: fix this test, it doesn't seem to recognize the IsAdmin true mock
	a.On("IsAdmin", mock.Anything).Return(true).Once()
	req, _ = http.NewRequest("GET", "/posts/"+strconv.Itoa(post.CreatedAt.Year())+"/"+strconv.Itoa(int(post.CreatedAt.Month()))+"/"+strconv.Itoa(post.CreatedAt.Day())+"/"+post.Slug, bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}
	if !strings.Contains(w.Body.String(), testPost.Title) {
		t.Errorf("Expected to see a post with title: %s but didn't", testPost.Title)
	}

	//html post not found
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	req, _ = http.NewRequest("GET", "/posts/2020/12/12/slug", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusNotFound, w.Code)
	}

	//get projects
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	req, _ = http.NewRequest("GET", "/projects", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	//get presentations
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	req, _ = http.NewRequest("GET", "/presentations", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	//get about
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	req, _ = http.NewRequest("GET", "/about", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	//search with matching query
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	req, _ = http.NewRequest("GET", "/search?q=Test", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}
	if !strings.Contains(w.Body.String(), testPost.Title) {
		t.Errorf("Expected search results to contain post title: %s", testPost.Title)
	}

	//search should exclude draft posts
	draftPost := blog.Post{
		Title:   "Draft Secret Post",
		Content: "This draft content should not appear in search",
		Draft:   true,
	}
	db.Create(&draftPost)

	a.On("IsAdmin", mock.Anything).Return(false).Once()
	req, _ = http.NewRequest("GET", "/search?q=Draft+Secret", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}
	if !strings.Contains(w.Body.String(), "0 results found") {
		t.Errorf("Expected '0 results found' for draft-only search query")
	}

	//search with non-matching query
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	req, _ = http.NewRequest("GET", "/search?q=zzzznonexistent", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}
	if !strings.Contains(w.Body.String(), "No results found") {
		t.Errorf("Expected 'No results found' message in empty search results")
	}
	if !strings.Contains(w.Body.String(), "0 results found") {
		t.Errorf("Expected '0 results found' in empty search results")
	}

	//logout
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	req, _ = http.NewRequest("GET", "/logout", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)
	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusTemporaryRedirect, w.Code)
	}

	//login (note: doesn't test actual login, just showing the login form)
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	req, _ = http.NewRequest("GET", "/login", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	//login without the .env file
	os.Rename("local.env", "local.env.old")
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	req, _ = http.NewRequest("GET", "/login", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusInternalServerError, w.Code)
	}
	os.Rename("local.env.old", "local.env")

	// Comment tests

	// Valid comment submission -> 303 redirect
	formData := "post_id=" + strconv.Itoa(int(post.ID)) + "&name=TestUser&content=Great+post!&redirect=" + url.QueryEscape(post.Permalink())
	req, _ = http.NewRequest("POST", "/comments", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("Expected status %d for valid comment but got %d", http.StatusSeeOther, w.Code)
	}
	location := w.Header().Get("Location")
	if !strings.Contains(location, "#comment-") {
		t.Errorf("Expected redirect to contain #comment- anchor but got: %s", location)
	}

	// Missing fields -> error redirect
	formData = "post_id=" + strconv.Itoa(int(post.ID)) + "&name=&content=&redirect=" + url.QueryEscape(post.Permalink())
	req, _ = http.NewRequest("POST", "/comments", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("Expected status %d for missing fields but got %d", http.StatusSeeOther, w.Code)
	}
	location = w.Header().Get("Location")
	if !strings.Contains(location, "comment_error=missing_fields") {
		t.Errorf("Expected redirect with missing_fields error but got: %s", location)
	}

	// Honeypot filled -> silent redirect (no error)
	formData = "post_id=" + strconv.Itoa(int(post.ID)) + "&name=Bot&content=spam&website=http://spam.com&redirect=" + url.QueryEscape(post.Permalink())
	req, _ = http.NewRequest("POST", "/comments", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("Expected status %d for honeypot but got %d", http.StatusSeeOther, w.Code)
	}
	location = w.Header().Get("Location")
	if strings.Contains(location, "comment_error") {
		t.Errorf("Honeypot should silently redirect without error, but got: %s", location)
	}

	// Rate limiting -> error redirect (already posted above from same IP)
	formData = "post_id=" + strconv.Itoa(int(post.ID)) + "&name=TestUser2&content=Another+comment&redirect=" + url.QueryEscape(post.Permalink())
	req, _ = http.NewRequest("POST", "/comments", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("Expected status %d for rate limit but got %d", http.StatusSeeOther, w.Code)
	}
	location = w.Header().Get("Location")
	if !strings.Contains(location, "comment_error=rate_limit") {
		t.Errorf("Expected redirect with rate_limit error but got: %s", location)
	}
}

func TestBacklinks(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"))
	db.AutoMigrate(&auth.BlogUser{}, &blog.Post{}, &blog.Tag{}, &blog.Backlink{}, &blog.ExternalBacklink{})
	a := &Auth{}
	sch := scholar.New("profiles.json", "articles.json")
	b := blog.New(db, a, "test", sch)

	// Create two posts. Post B will link to Post A.
	postA := blog.Post{
		Title:   "Post A",
		Content: "This is Post A content",
		Slug:    "post-a",
	}
	db.Create(&postA)

	// Post B links to Post A using a markdown link
	postB := blog.Post{
		Title:   "Post B",
		Content: "Check out [Post A](/posts/" + postA.CreatedAt.Format("2006/1/2") + "/post-a) for more info",
		Slug:    "post-b",
	}
	db.Create(&postB)

	// Compute backlinks for Post B
	b.ComputeBacklinks(&postB)

	// Post A should have Post B as a backlink
	backlinks := b.GetBacklinks(postA.ID)
	if len(backlinks) != 1 {
		t.Fatalf("Expected 1 backlink for Post A, got %d", len(backlinks))
	}
	if backlinks[0].ID != postB.ID {
		t.Errorf("Expected backlink from Post B (ID %d), got ID %d", postB.ID, backlinks[0].ID)
	}

	// Post B should have Post A as an outbound link
	outbound := b.GetOutboundLinks(postB.ID)
	if len(outbound) != 1 {
		t.Fatalf("Expected 1 outbound link for Post B, got %d", len(outbound))
	}
	if outbound[0].ID != postA.ID {
		t.Errorf("Expected outbound link to Post A (ID %d), got ID %d", postA.ID, outbound[0].ID)
	}

	// Post A should have no outbound links
	outboundA := b.GetOutboundLinks(postA.ID)
	if len(outboundA) != 0 {
		t.Errorf("Expected 0 outbound links for Post A, got %d", len(outboundA))
	}

	// Post B should have no backlinks
	backlinksB := b.GetBacklinks(postB.ID)
	if len(backlinksB) != 0 {
		t.Errorf("Expected 0 backlinks for Post B, got %d", len(backlinksB))
	}

	// Update Post B to remove the link, backlinks should be cleared
	postB.Content = "Updated content with no links"
	db.Save(&postB)
	b.ComputeBacklinks(&postB)

	backlinks = b.GetBacklinks(postA.ID)
	if len(backlinks) != 0 {
		t.Errorf("Expected 0 backlinks after removing link, got %d", len(backlinks))
	}
}

func TestExternalBacklinks(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"))
	db.AutoMigrate(&auth.BlogUser{}, &blog.Post{}, &blog.Tag{}, &blog.Backlink{}, &blog.ExternalBacklink{})
	a := &Auth{}
	sch := scholar.New("profiles.json", "articles.json")
	b := blog.New(db, a, "test", sch)

	post := blog.Post{
		Title:   "Test Post",
		Content: "Some content",
		Slug:    "test-post",
	}
	db.Create(&post)

	router := gin.Default()
	store := cookie.NewStore([]byte("changelater"))
	router.Use(sessions.Sessions("test", store))

	// Test external referer is tracked
	router.GET("/track", func(c *gin.Context) {
		b.TrackReferer(c, post.ID)
		c.String(http.StatusOK, "ok")
	})

	// Request with external referer
	req, _ := http.NewRequest("GET", "/track", nil)
	req.Header.Set("Referer", "https://example.com/some-page")
	req.Host = "myblog.com"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	backlinks := b.GetExternalBacklinks(post.ID)
	if len(backlinks) != 1 {
		t.Fatalf("Expected 1 external backlink, got %d", len(backlinks))
	}
	if backlinks[0].Referer != "https://example.com/some-page" {
		t.Errorf("Expected referer 'https://example.com/some-page', got '%s'", backlinks[0].Referer)
	}
	if backlinks[0].HitCount != 1 {
		t.Errorf("Expected hit count 1, got %d", backlinks[0].HitCount)
	}

	// Second request from same referer should increment hit count
	req, _ = http.NewRequest("GET", "/track", nil)
	req.Header.Set("Referer", "https://example.com/some-page")
	req.Host = "myblog.com"
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	backlinks = b.GetExternalBacklinks(post.ID)
	if len(backlinks) != 1 {
		t.Fatalf("Expected 1 external backlink after second hit, got %d", len(backlinks))
	}
	if backlinks[0].HitCount != 2 {
		t.Errorf("Expected hit count 2 after second hit, got %d", backlinks[0].HitCount)
	}

	// Self-referral should be skipped
	req, _ = http.NewRequest("GET", "/track", nil)
	req.Header.Set("Referer", "https://myblog.com/other-page")
	req.Host = "myblog.com"
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	backlinks = b.GetExternalBacklinks(post.ID)
	if len(backlinks) != 1 {
		t.Errorf("Expected self-referral to be skipped, got %d backlinks", len(backlinks))
	}

	// Empty referer should be skipped
	req, _ = http.NewRequest("GET", "/track", nil)
	req.Host = "myblog.com"
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	backlinks = b.GetExternalBacklinks(post.ID)
	if len(backlinks) != 1 {
		t.Errorf("Expected empty referer to be skipped, got %d backlinks", len(backlinks))
	}
}
