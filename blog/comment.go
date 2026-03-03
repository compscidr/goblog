package blog

import "time"

// Comment represents a user comment on a blog post
type Comment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	PostID    uint      `json:"post_id" gorm:"index;not null"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Content   string    `sql:"type:text;" json:"content"`
	IPAddress string    `json:"-"`
}
