package admin

import (
	"fmt"
	"goblog/auth"
	"goblog/blog"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

//UploadsFolder configures where the file uploads should be stored. This is
//mostly used for testing
var UploadsFolder = "uploads/"

// Admin handles admin requests
type Admin struct {
	db      *gorm.DB
	auth    auth.IAuth
	b 		blog.Blog
	version string
}

//New constructs an Admin API
func New(db *gorm.DB, auth auth.IAuth, b blog.Blog, version string) Admin {
	api := Admin{db, auth, b, version}
	return api
}

//////JSON API///////
func safeSlug(slug string) string {
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "/", "")
	slug = strings.ReplaceAll(slug, ".", "-")
	return url.QueryEscape(slug)
}

//CreatePost adds a post if the user has permission
func (a Admin) CreatePost(c *gin.Context) {
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
	a.db.Create(&requestPost)

	log.Println("POST CREATED: ", requestPost)
	c.JSON(http.StatusCreated, requestPost)
}

//UploadFile is the endpoint for storing files on the server
//https://github.com/gin-gonic/examples/blob/master/upload-file/single/main.go
func (a Admin) UploadFile(c *gin.Context) {
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
	if err := c.SaveUploadedFile(file, filename); err != nil {
		log.Println(fmt.Sprintf("Save Upload File Error erorr: %s", err.Error()))
		c.JSON(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()))
		return
	}

	log.Println("Saved file okay: " + filename)

	c.JSON(http.StatusOK, map[string]interface{}{"filename": "/" + filename})
}

//UpdatePost modifies an existing post
//Requires the ID of the post, title and content to not be empty
func (a Admin) UpdatePost(c *gin.Context) {
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
	requestPost.Slug = safeSlug(requestPost.Title)

	var existingPost blog.Post
	err := a.db.Where("id = ?", requestPost.ID).First(&existingPost).Error

	if err != nil {
		c.JSON(http.StatusBadRequest, "Existing post with ID "+fmt.Sprint(requestPost.ID)+" not found")
		return
	}

	//clear old associations
	a.db.Model(&existingPost).Association("Tags").Clear()

	existingPost.Title = requestPost.Title
	existingPost.Content = requestPost.Content
	existingPost.Slug = requestPost.Slug
	existingPost.Tags = requestPost.Tags
	existingPost.CreatedAt = requestPost.CreatedAt
	a.db.Model(&existingPost).Where("id = ?", requestPost.ID).Updates(&existingPost)

	log.Println("POST UPDATED: ", existingPost)
	c.JSON(http.StatusAccepted, existingPost)
}

//DeletePost deletes a post from the database
func (a Admin) DeletePost(c *gin.Context) {
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

	a.db.Where("id = ?", requestPost.ID).Delete(&blog.Post{})

	c.JSON(http.StatusOK, "")
}

//////HTML API///////

//Admin is the admin dashboard of the website
func (a Admin) Admin(c *gin.Context) {
	c.HTML(http.StatusOK, "admin.html", gin.H{
		"posts": a.b.GetPosts(),
		"logged_in": a.auth.IsLoggedIn(c),
		"is_admin":  a.auth.IsAdmin(c),
		"version": a.version,
	})
}

func (a Admin) Post(c *gin.Context) {
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
		})
	} else {
		c.HTML(http.StatusOK, "post-admin.html", gin.H{
			"logged_in": a.auth.IsAdmin(c),
			"is_admin":  a.auth.IsLoggedIn(c),
			"post":      post,
			"version":   a.b.Version,
		})
	}
}