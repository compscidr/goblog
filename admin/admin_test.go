package admin_test

import (
	"bytes"
	"encoding/json"
	"goblog/admin"
	"goblog/auth"
	"goblog/blog"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestCreatePost(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"))
	db.AutoMigrate(&auth.BlogUser{})
	db.AutoMigrate(&blog.Post{})
	a := &Auth{}
	b := blog.New(db, a, "test")
	ad := admin.New(db, a, b, "test")

	router := gin.Default()
	store := cookie.NewStore([]byte("changelater"))
	router.Use(sessions.Sessions("www.jasonernst.com", store))
	router.POST("/api/v1/posts", ad.CreatePost)
	router.GET("/api/v1/posts", b.ListPosts)
	router.PATCH("/api/v1/posts", ad.UpdatePost)
	router.DELETE("/api/v1/posts", ad.DeletePost)
	router.POST("/api/v1/upload", ad.UploadFile)

	router.GET("/admin", ad.Admin)
	router.GET("/admin/dashboard", ad.AdminDashboard)
	router.GET("/admin/posts", ad.AdminPosts)
	router.GET("/admin/newpost", ad.AdminNewPost)
	router.GET("/admin/settings", ad.AdminSettings)

	//improper content-type
	testPost := blog.Post{
		Title:   "Test title",
		Content: "This is some test content",
	}
	jsonValue, _ := json.Marshal(testPost)
	req, _ := http.NewRequest("POST", "/api/v1/posts", bytes.NewBuffer(jsonValue))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusUnsupportedMediaType, w.Code)
	}

	//is not admin
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	req.Header.Add("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusUnauthorized, w.Code)
	}

	//is admin
	a.On("IsAdmin", mock.Anything).Return(true).Once()
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusCreated, w.Code)
	}

	//missing title
	testPost = blog.Post{
		Content: "This is some test content",
	}
	jsonValue, _ = json.Marshal(testPost)
	req, _ = http.NewRequest("POST", "/api/v1/posts", bytes.NewBuffer(jsonValue))
	req.Header.Add("Content-Type", "application/json")
	a.On("IsAdmin", mock.Anything).Return(true).Once()
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusBadRequest, w.Code)
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

	//update post
	testPost = blog.Post{
		Title:   "Test title updated",
		Content: "This is some test content updated",
	}
	testPost.ID = post.ID
	jsonValue, _ = json.Marshal(testPost)
	a.On("IsAdmin", mock.Anything).Return(true).Once()
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PATCH", "/api/v1/posts", bytes.NewBuffer(jsonValue))
	req.Header.Add("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusAccepted, w.Code)
	}

	//update post, bad type
	jsonValue, _ = json.Marshal(testPost)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PATCH", "/api/v1/posts", bytes.NewBuffer(jsonValue))
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusUnsupportedMediaType, w.Code)
	}

	//update post, not admin
	jsonValue, _ = json.Marshal(testPost)
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PATCH", "/api/v1/posts", bytes.NewBuffer(jsonValue))
	req.Header.Add("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusUnauthorized, w.Code)
	}

	//missing Title
	testPost = blog.Post{
		Title:   "",
		Content: "This is some test content updated",
	}
	testPost.ID = post.ID
	jsonValue, _ = json.Marshal(testPost)
	a.On("IsAdmin", mock.Anything).Return(true).Once()
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PATCH", "/api/v1/posts", bytes.NewBuffer(jsonValue))
	req.Header.Add("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusBadRequest, w.Code)
	}

	//missing id
	testPost = blog.Post{
		Title:   "Test",
		Content: "This is some test content updated",
	}
	testPost.ID = 99999
	jsonValue, _ = json.Marshal(testPost)
	a.On("IsAdmin", mock.Anything).Return(true).Once()
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PATCH", "/api/v1/posts", bytes.NewBuffer(jsonValue))
	req.Header.Add("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusBadRequest, w.Code)
	}

	//delete post: incorrect content type
	jsonValue, _ = json.Marshal(testPost)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/v1/posts", bytes.NewBuffer(jsonValue))
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusUnsupportedMediaType, w.Code)
	}

	//delete post: not admin
	jsonValue, _ = json.Marshal(testPost)
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/v1/posts", bytes.NewBuffer(jsonValue))
	req.Header.Add("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusUnauthorized, w.Code)
	}

	//good Delete
	jsonValue, _ = json.Marshal(testPost)
	a.On("IsAdmin", mock.Anything).Return(true).Once()
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/v1/posts", bytes.NewBuffer(jsonValue))
	req.Header.Add("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}

	//upload file test
	//https://www.programmersought.com/article/6833575288/
	path := "../README.md"
	file, err := os.Open(path)
	if err != nil {
		t.Error(err)
	}

	defer file.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		writer.Close()
		t.Error(err)
	}
	io.Copy(part, file)
	writer.Close()

	req = httptest.NewRequest("POST", "/api/v1/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()
	a.On("IsAdmin", mock.Anything).Return(true).Once()
	admin.WWWFolder = "../www/"
	admin.UploadsFolder = "uploads/"
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		body, _ := ioutil.ReadAll(w.Body)
		t.Fatalf("Expected to get status %d but instead got %d\n%s", http.StatusOK, w.Code, body)
	}
	err = os.Remove("../uploads/README.md")

	//file upload, upload folder doesn't exist
	//admin.UploadsFolder = "dfadf/"
	//w = httptest.NewRecorder()
	//a.On("IsAdmin", mock.Anything).Return(true).Once()
	//router.ServeHTTP(w, req)
	//if w.Code != http.StatusBadRequest {
	//	body, _ := ioutil.ReadAll(w.Body)
	//	t.Fatalf("Expected to get status %d but instead got %d\n%s", http.StatusBadRequest, w.Code, body)
	//}

	//file upload, not admin
	a.On("IsAdmin", mock.Anything).Return(false).Once()
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		body, _ := ioutil.ReadAll(w.Body)
		t.Fatalf("Expected to get status %d but instead got %d\n%s", http.StatusUnauthorized, w.Code, body)
	}

	//file upload, missing file
	a.On("IsAdmin", mock.Anything).Return(true).Once()
	w = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/v1/upload", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		body, _ := ioutil.ReadAll(w.Body)
		t.Fatalf("Expected to get status %d but instead got %d\n%s", http.StatusBadRequest, w.Code, body)
	}

	//get admin
	router.LoadHTMLGlob("../templates/*")
	a.On("IsAdmin", mock.Anything).Return(true).Once()
	a.On("IsLoggedIn", mock.Anything).Return(true).Once()
	req, _ = http.NewRequest("GET", "/admin", bytes.NewBuffer(jsonValue))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected to get status %d but instead got %d\n", http.StatusOK, w.Code)
	}
}
