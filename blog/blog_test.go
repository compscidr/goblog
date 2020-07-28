package blog_test

import (
	"bytes"
	"encoding/json"
	"goblog/admin"
	"goblog/auth"
	"goblog/blog"
	"net/http"
	"net/http/httptest"
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
	a := &Auth{}
	admin := admin.New(db, a, "test")
	b := blog.New(db, a, "test")

	router := gin.Default()
	store := cookie.NewStore([]byte("changelater"))
	router.Use(sessions.Sessions("www.jasonernst.com", store))
	router.POST("/api/v1/posts", admin.CreatePost)
	router.GET("/api/v1/posts", b.ListPosts)

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
	testPost := blog.Post{
		Title:   "Test title",
		Content: "This is some test content",
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

}
