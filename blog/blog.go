package blog

import (
	"errors"
	"goblog/auth"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/joho/godotenv"
)

// Blog API handles non-admin functions of the blog like listing posts, tags
// comments, etc.
type Blog struct {
	db   *gorm.DB
	auth *auth.Auth
}

//New constructs an Admin API
func New(db *gorm.DB, auth *auth.Auth) Blog {
	api := Blog{db, auth}
	return api
}

//Generic Functions (not JSON or HTML)
func (b Blog) getPosts() []Post {
	var posts []Post
	b.db.Order("created_at desc").Find(&posts)
	return posts
}

func (b Blog) getPost(c *gin.Context) (*Post, error) {
	var post Post
	year, err := strconv.Atoi(c.Param("yyyy"))
	if err != nil {
		return nil, errors.New("Year must be an integer")
	}
	month, err := strconv.Atoi(c.Param("mm"))
	if err != nil {
		return nil, errors.New("Month must be an integer")
	}
	day, err := strconv.Atoi(c.Param("dd"))
	if err != nil {
		return nil, errors.New("Day must be an integer")
	}
	slug := c.Param("slug")

	log.Println("Looking for post: ", year, "/", month, "/", day, "/", slug)

	if err := b.db.Where("created_at > ? AND slug = ?", time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), slug).First(&post).Error; err != nil {
		return nil, errors.New("No post at " + strconv.Itoa(year) + "/" + strconv.Itoa(month) + "/" + strconv.Itoa(day) + "/" + slug)
	}

	b.db.Model(&post).Related(&post.Tags, "Tags")
	log.Println("Found: ", post.Title, " TAGS: ", post.Tags)
	return &post, nil
}

func (b Blog) getPostsByTag(c *gin.Context) ([]Post, error) {
	var posts []Post
	var tag Tag
	name := c.Param("name")
	if err := b.db.Where("name = ?", name).First(&tag).Error; err != nil {
		return nil, errors.New("No tag named " + name)
	}

	b.db.Model(&tag).Related(&posts, "Posts")
	log.Print("POSTS: ", posts)
	return posts, nil
}

//////JSON API///////

//ListPosts lists all blog posts
func (b Blog) ListPosts(c *gin.Context) {
	c.JSON(http.StatusOK, b.getPosts())
}

//GetPost returns a post with yyyy/mm/dd/slug
func (b Blog) GetPost(c *gin.Context) {
	post, err := b.getPost(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, err)
	}
	if post == nil {
		c.JSON(http.StatusNotFound, "Post Not Found")
	}
	c.JSON(http.StatusOK, post)
}

//////HTML API///////

//Home returns html of the home page using the template
//if people want to have different stuff show on the home page they probably
//need to modify this function
func (b Blog) Home(c *gin.Context) {
	c.HTML(http.StatusOK, "home.html", gin.H{
		"logged_in": b.auth.IsLoggedIn(c),
		"is_admin":  b.auth.IsAdmin(c),
	})
}

//Posts is the index page for blog posts
func (b Blog) Posts(c *gin.Context) {
	c.HTML(http.StatusOK, "posts.html", gin.H{
		"logged_in": b.auth.IsLoggedIn(c),
		"is_admin":  b.auth.IsAdmin(c),
		"posts":     b.getPosts(),
	})
}

//Post is the page for all individual posts
func (b Blog) Post(c *gin.Context) {
	post, err := b.getPost(c)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error":       "Post Not Found",
			"description": err.Error(),
		})
	} else {
		if b.auth.IsAdmin(c) {
			c.HTML(http.StatusOK, "post-admin.html", gin.H{
				"logged_in": b.auth.IsLoggedIn(c),
				"is_admin":  b.auth.IsAdmin(c),
				"post":      post,
			})
		} else {
			c.HTML(http.StatusOK, "post.html", gin.H{
				"logged_in": b.auth.IsLoggedIn(c),
				"is_admin":  b.auth.IsAdmin(c),
				"post":      post,
			})
		}
	}
}

//Tag lists all posts with a given tag
func (b Blog) Tag(c *gin.Context) {
	tag := c.Param("name")
	posts, err := b.getPostsByTag(c)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error":       "Tag Not Found",
			"description": err.Error(),
		})
	} else {
		c.HTML(http.StatusOK, "tag.html", gin.H{
			"logged_in": b.auth.IsLoggedIn(c),
			"is_admin":  b.auth.IsAdmin(c),
			"posts":     posts,
			"tag":       tag,
		})
	}
}

func (b Blog) Tags(c *gin.Context) {
	c.HTML(http.StatusOK, "tags.html", gin.H{})
}

//Speaking is the index page for presentations
func (b Blog) Speaking(c *gin.Context) {
	c.HTML(http.StatusOK, "presentations.html", gin.H{
		"logged_in": b.auth.IsLoggedIn(c),
		"is_admin":  b.auth.IsAdmin(c),
	})
}

//Projects is the index page for projects / code
func (b Blog) Projects(c *gin.Context) {
	c.HTML(http.StatusOK, "projects.html", gin.H{
		"logged_in": b.auth.IsLoggedIn(c),
		"is_admin":  b.auth.IsAdmin(c),
	})
}

//About is the about page
func (b Blog) About(c *gin.Context) {
	c.HTML(http.StatusOK, "about.html", gin.H{
		"logged_in": b.auth.IsLoggedIn(c),
		"is_admin":  b.auth.IsAdmin(c),
	})
}

//Login to the blog
func (b Blog) Login(c *gin.Context) {
	err := godotenv.Load(".env")
	if err != nil {
		//fall back to local config
		err = godotenv.Load("local.env")
		if err != nil {
			//todo: handle better - perhaps return error to browser
			c.HTML(http.StatusInternalServerError, "Error loading .env file: "+err.Error(), gin.H{
				"logged_in": b.auth.IsLoggedIn(c),
				"is_admin":  b.auth.IsAdmin(c),
			})
			return
		}
	}

	clientID := os.Getenv("client_id")
	c.HTML(http.StatusOK, "login.html", gin.H{
		"logged_in": b.auth.IsLoggedIn(c),
		"is_admin":  b.auth.IsAdmin(c),
		"client_id": clientID,
	})
}

//Logout of the blog
func (b Blog) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("token")
	session.Save()
	c.Redirect(http.StatusTemporaryRedirect, "/")
}
