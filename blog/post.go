package blog

import (
	"bytes"
	"time"
)

// Post defines blog posts
type Post struct {
	ID 		uint   `gorm:"primary_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
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
func (p Post) PreviewContent(length int) string {
	// This cast is O(N)
	runes := bytes.Runes([]byte(p.Content))
	if len(runes) > length {
		return string(runes[:length])
	}
	return string(runes)
}

//Permalink returns the link to the post relative to root
func (p Post) Permalink() string {
	return p.CreatedAt.Format("/posts/2006/01/02/") + p.Slug
}

func (p Post) Adminlink() string {
	return p.CreatedAt.Format("/admin/posts/2006/01/02/") + p.Slug
}

func (t Tag) Permalink() string {
	return "/tag/" + t.Name
}
