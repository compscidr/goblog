package blog

import (
	"github.com/jinzhu/gorm"
)

// Post defines blog posts
type Post struct {
	gorm.Model
	Title   string `json:"title"`
	Slug    string `json:"slug"`
	Content string `sql:"type:text;" json:"content"`
}
