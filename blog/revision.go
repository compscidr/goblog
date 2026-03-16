package blog

import "time"

type PostRevision struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	PostID     uint      `json:"post_id" gorm:"index;not null"`
	Title      string    `json:"title"`
	Slug       string    `json:"slug"`
	Content    string    `sql:"type:text;" json:"content"`
	Draft      bool      `json:"draft"`
	PostTypeID uint      `json:"post_type_id"`
}
