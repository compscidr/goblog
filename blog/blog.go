package blog

import (
	"log"
	"net/http"
	"strconv"
	"time"

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
