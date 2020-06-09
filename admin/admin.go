package admin

import (
	"fmt"
	"goblog/auth"
	"goblog/blog"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// Admin handles admin requests
type Admin struct {
	db   *gorm.DB
	auth auth.IAuth
}

//New constructs an Admin API
func New(db *gorm.DB, auth auth.IAuth) Admin {
	api := Admin{db, auth}
	return api
}

//debug function, shows the user table
func (a Admin) displayUserTable() {
	var users []auth.BlogUser
	a.db.Find(&users)
	log.Println(users)
}

// formatRequest generates ascii representation of a request
func formatRequest(r *http.Request) string {
	// Create return string
	var request []string
	// Add the request string
	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	// Add the host
	request = append(request, fmt.Sprintf("Host: %v", r.Host))
	// Loop through headers
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}

	// If this is a POST, add post data
	if r.Method == "POST" {
		r.ParseForm()
		request = append(request, "\n")
		request = append(request, r.Form.Encode())
	}

	// Return the request as a string
	return strings.Join(request, "\n")
}

//////JSON API///////

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
		c.JSON(http.StatusBadRequest, "Malformed request")
		return
	}

	if requestPost.Title == "" || requestPost.Content == "" {
		c.JSON(http.StatusBadRequest, "Missing Title or Content")
		return
	}

	//todo: make tags work - need to get the relations working
	requestPost.Slug = url.QueryEscape(strings.Replace(requestPost.Title, " ", "-", -1))
	log.Print("CREATING POST: ", requestPost)
	a.db.Create(&requestPost)

	//todo - improve this in case of collision on title
	var post blog.Post
	a.db.First(&post, "title = ?", requestPost.Title)

	log.Println("POST CREATED: ", post)
	c.JSON(http.StatusCreated, post)
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
	c.BindJSON(&requestPost)

	if requestPost.Title == "" || requestPost.Content == "" || requestPost.ID < 0 {
		c.JSON(http.StatusBadRequest, "Missing ID, Title or Content")
		return
	}
	requestPost.Slug = url.QueryEscape(strings.Replace(requestPost.Title, " ", "-", -1))

	var existingPost blog.Post
	err := a.db.Where("id = ?", requestPost.ID).First(&existingPost).Error

	if err != nil {
		c.JSON(http.StatusBadRequest, "Existing post with ID "+fmt.Sprint(requestPost.ID)+" not found")
		return
	}

	existingPost.Title = requestPost.Title
	existingPost.Content = requestPost.Content
	existingPost.Slug = requestPost.Slug
	a.db.Model(&existingPost).Where("id = ?", requestPost.ID).Updates(&existingPost)

	log.Println("POST UPDATED: ", existingPost)
	c.JSON(http.StatusAccepted, existingPost)
}

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
	c.BindJSON(&requestPost)

	if requestPost.ID < 0 {
		c.JSON(http.StatusBadRequest, "Missing ID, Title or Content")
		return
	}

	a.db.Where("id = ?", requestPost.ID).Delete(&blog.Post{})

	c.JSON(http.StatusOK, "")
}

// AdminHandler handles admin requests
//func (a Admin) AdminHandler(w http.ResponseWriter, r *http.Request) {
func (a Admin) AdminHandler(c *gin.Context) {
	token := c.Request.Header.Get("Authorization")

	//check to see if user is logged in (todo add expiry)
	//can't do this until we publish a version with the auth module in it
	var existingUser auth.BlogUser
	err := a.db.Where("access_token = ?", token).First(&existingUser).Error
	if err != nil {
		c.JSON(http.StatusUnauthorized, "Not Authorized: "+token)
		return
	}

	log.Println("AUTHORZIED: ", token)
	c.JSON(http.StatusOK, token)
}

//////HTML API///////

//Admin is the admin dashboard of the website
func (a Admin) Admin(c *gin.Context) {
	c.HTML(http.StatusOK, "admin.html", gin.H{
		"logged_in": a.auth.IsLoggedIn(c),
		"is_admin":  a.auth.IsAdmin(c),
	})
}
