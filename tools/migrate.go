package tools

import (
	"goblog/auth"
	"goblog/blog"
	"gorm.io/gorm"
	"log"
)

func Migrate(db *gorm.DB) {
	err := db.AutoMigrate(&auth.BlogUser{}, &blog.Post{}, &blog.Tag{}, &auth.AdminUser{})
	if err != nil {
		log.Println("Error migrating tables: " + err.Error())
		return
	}
}
