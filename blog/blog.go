package blog

import (
	"net/http"

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
