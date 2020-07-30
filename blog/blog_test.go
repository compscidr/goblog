package blog_test

import (
	"bytes"
	"encoding/json"
	"goblog/admin"
	"goblog/auth"
	"goblog/blog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite" // this is the db driver
	"github.com/stretchr/testify/mock"
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
	db, _ := gorm.Open("sqlite3", ":memory:")
	db.AutoMigrate(&auth.BlogUser{})
	db.AutoMigrate(&blog.Post{})
	db.AutoMigrate(&blog.Tag{})
	a := &Auth{}
	admin := admin.New(db, a, "test")
	b := blog.New(db, a, "test")

	router := gin.Default()
	store := cookie.NewStore([]byte("changelater"))
	router.Use(sessions.Sessions("www.jasonernst.com", store))

	//json requests
	router.POST("/api/v1/posts", admin.CreatePost)
	router.GET("/api/v1/posts", b.ListPosts)
	router.GET("/api/v1/posts/:yyyy/:mm/:dd/:slug", b.GetPost)

	//html requests
	router.GET("/posts/:yyyy/:mm/:dd/:slug", b.Post)
	router.GET("/tag/:name", b.Tag)
	router.GET("/posts", b.Posts)
	router.GET("/tags", b.Tags)
	router.GET("/", b.Home)
	router.NoRoute(b.NoRoute)

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
		t.Errorf("Expected to see a post with title: " + testPost.Title + " but didn't")
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
		t.Errorf("Expected to see a post with title: " + testPost.Title + " but didn't")
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
	a.On("IsAdmin", mock.Anything).Return(false)
	a.On("IsLoggedIn", mock.Anything).Return(false)
	jsonValue, _ = json.Marshal("")
	req, _ = http.NewRequest("GET", "/tag/test", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	//get not found tag
	req, _ = http.NewRequest("GET", "/tag/blah", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusNotFound, w.Code)
	}

	//get all tags
	req, _ = http.NewRequest("GET", "/tags", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	//get all posts
	req, _ = http.NewRequest("GET", "/posts", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	//get home
	req, _ = http.NewRequest("GET", "/", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	//no route
	req, _ = http.NewRequest("GET", "/dfadfasdf", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusNotFound, w.Code)
	}

	//html post
	req, _ = http.NewRequest("GET", "/posts/"+strconv.Itoa(post.CreatedAt.Year())+"/"+strconv.Itoa(int(post.CreatedAt.Month()))+"/"+strconv.Itoa(post.CreatedAt.Day())+"/"+post.Slug, bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), testPost.Title) {
		t.Errorf("Expected to see a post with title: " + testPost.Title + " but didn't")
	}

	//html post not found
	req, _ = http.NewRequest("GET", "/posts/2020/12/12/slug", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusNotFound, w.Code)
	}
}
