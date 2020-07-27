package main_test

import (
	"goblog/admin"
	"goblog/auth"
	"goblog/blog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite" // this is the db driver
	"github.com/stretchr/testify/assert"
)

//This test is a sort of integration test because it requires the auth package
//the admin package and the blog package to all work together
func TestCreatePost(t *testing.T) {
	db, err := gorm.Open("sqlite3", ":memory:")
	assert.Nil(t, err)
	db.AutoMigrate(&auth.BlogUser{})
	db.AutoMigrate(&blog.Post{})

	auth := auth.New(db, "test")
	admin := admin.New(db, &auth, "test")
	//b := blog.New(db, &auth)

	router := gin.Default()
	store := cookie.NewStore([]byte("changelater"))
	router.Use(sessions.Sessions("www.jasonernst.com", store))
	router.POST("/api/v1/posts", admin.CreatePost)

	//not authorized
	post := url.Values{
		"Title":   {"Test"},
		"Content": {"Test Content"},
	}
	form := strings.NewReader(post.Encode())
	req, err := http.NewRequest("POST", "/api/v1/posts", form)
	assert.Nil(t, err)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
}
