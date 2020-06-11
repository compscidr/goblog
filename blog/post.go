package blog

import (
	"bytes"

	"github.com/jinzhu/gorm"
)

// Post defines blog posts
type Post struct {
	gorm.Model
	Title   string `json:"title"`
	Slug    string `json:"slug"`
	Content string `sql:"type:text;" json:"content"`
	Tags    []Tag  `gorm:"many2many:post_tags" json:"tags"`
}

// Tag is used to collect Posts with similar topics
type Tag struct {
	Name  string `gorm:"PRIMARY_KEY" json:"name"`
	Posts []Post `gorm:"many2many:post_tags"`
}

//PreviewContent gets a shortened version of the content for showing a preview
//https://stackoverflow.com/questions/23466497/how-to-truncate-a-string-in-a-golang-template
func (c Post) PreviewContent(length int) string {
	// This cast is O(N)
	runes := bytes.Runes([]byte(c.Content))
	if len(runes) > length {
		return string(runes[:length])
	}
	return string(runes)
}

//Permalink returns the link to the post relative to root
func (c Post) Permalink() string {
	return c.CreatedAt.Format("/posts/2006/01/02/") + c.Slug
}
