package blog

import "time"

// PostRevision stores a snapshot of a post's content before each edit,
// allowing users to view previous versions and roll back to them.
type PostRevision struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	PostID     uint      `json:"post_id" gorm:"index;not null"`
	Title      string    `json:"title"`
	Slug       string    `json:"slug"`
	Content    string    `gorm:"type:text" json:"content"`
	Draft      bool      `json:"draft"`
	PostTypeID uint      `json:"post_type_id"`
}
