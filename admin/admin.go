package admin

import (
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
	db *gorm.DB
}

//New constructs an Admin API
func New(db *gorm.DB) Admin {
	api := Admin{db}
	return api
}

//debug function, shows the user table
func (a Admin) displayUserTable() {
	var users []auth.BlogUser
	a.db.Find(&users)
	log.Println(users)
}

//CreatePost adds a post if the user has permission
func (a Admin) CreatePost(c *gin.Context) {
	token := c.Request.Header.Get("Authorization")
	log.Println("CREATE POST, AUTH: ", token)
	a.displayUserTable()

	//check to see if user is logged in (todo add expiry)
	//can't do this until we publish a version with the auth module in it
	var existingUser auth.BlogUser
	err := a.db.Where("access_token = ?", token).First(&existingUser).Error
	if err != nil {
		c.JSON(http.StatusForbidden, "Not Authorized: "+token)
		return
	}

	var requestPost blog.Post
	c.BindJSON(&requestPost)

	//todo: make tags work - need to get the relations working
	requestPost.Slug = url.QueryEscape(strings.Replace(requestPost.Title, " ", "-", -1))
	log.Print("CREATING POST: ", requestPost)
	a.db.Create(&requestPost)

	//todo - improve this in case of collision on title
	var post blog.Post
	a.db.First(&post, "title = ?", requestPost.Title)

	log.Println("CREATE POST AUTHORIZED: ", token, "\n", post)
	c.JSON(http.StatusCreated, post)
}

func (a Admin) UpdatePost(c *gin.Context) {
	token := c.Request.Header.Get("Authorization")

	//check to see if user is logged in (todo add expiry)
	//can't do this until we publish a version with the auth module in it
	var existingUser auth.BlogUser
	err := a.db.Where("access_token = ?", token).First(&existingUser).Error
	if err != nil {
		c.JSON(http.StatusForbidden, "Not Authorized: "+token)
		return
	}

	log.Println("UPDATE POST AUTHORZIED: ", token)
	c.JSON(http.StatusOK, token)
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
		c.JSON(http.StatusForbidden, "Not Authorized: "+token)
		return
	}

	log.Println("AUTHORZIED: ", token)
	c.JSON(http.StatusOK, token)
}
