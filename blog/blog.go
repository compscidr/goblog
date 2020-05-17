package blog

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// Blog API handles non-admin functions of the blog like listing posts, tags
// comments, etc.
type Blog struct {
	db *gorm.DB
}

//New constructs an Admin API
func New(db *gorm.DB) Blog {
	api := Blog{db}
	return api
}

//////JSON API///////

//ListPosts lists all blog posts
func (b Blog) ListPosts(c *gin.Context) {
	var posts []Post
	b.db.Find(&posts)

	c.JSON(http.StatusOK, posts)
}

//GetPost returns a post with yyyy/mm/dd/slug
func (b Blog) GetPost(c *gin.Context) {
	var post Post
	year, err := strconv.Atoi(c.Param("yyyy"))
	if err != nil {
		c.JSON(http.StatusBadRequest, "Year must be an integer")
		return
	}
	month, err := strconv.Atoi(c.Param("mm"))
	if err != nil {
		c.JSON(http.StatusBadRequest, "Month must be an integer")
		return
	}
	day, err := strconv.Atoi(c.Param("dd"))
	if err != nil {
		c.JSON(http.StatusBadRequest, "Day must be an integer")
		return
	}
	slug := c.Param("slug")

	log.Println("Looking for post: ", year, "/", month, "/", day, "/", slug)

	if err := b.db.Where("created_at > ? AND slug = ?", time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), slug).First(&post).Error; err != nil {
		c.JSON(http.StatusNotFound, "Post Not Found")
		return
	}

	c.JSON(http.StatusOK, post)
}

//Returns true if the user is logged in, false otherwise
func isLoggedIn(c *gin.Context) bool {
	session := sessions.Default(c)
	token := session.Get("token")
	if token == nil {
		return false
	}
	return true
}

//////HTML API///////

//Home returns html of the home page using the template
//if people want to have different stuff show on the home page they probably
//need to modify this function
func (b Blog) Home(c *gin.Context) {
	c.HTML(http.StatusOK, "home.html", gin.H{
		"logged_in": isLoggedIn(c),
	})
}

//Posts is the index page for blog posts
func (b Blog) Posts(c *gin.Context) {
	c.HTML(http.StatusOK, "posts.html", gin.H{
		"logged_in": isLoggedIn(c),
	})
}

//Speaking is the index page for presentations
func (b Blog) Speaking(c *gin.Context) {
	c.HTML(http.StatusOK, "presentations.html", gin.H{
		"logged_in": isLoggedIn(c),
	})
}

//Projects is the index page for projects / code
func (b Blog) Projects(c *gin.Context) {
	c.HTML(http.StatusOK, "projects.html", gin.H{
		"logged_in": isLoggedIn(c),
	})
}

//About is the about page
func (b Blog) About(c *gin.Context) {
	c.HTML(http.StatusOK, "about.html", gin.H{
		"logged_in": isLoggedIn(c),
	})
}

//Login to the blog
func (b Blog) Login(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"logged_in": isLoggedIn(c),
	})
}

//Logout of the blog
func (b Blog) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("token")
	session.Save()
	c.Redirect(http.StatusTemporaryRedirect, "/")
}
