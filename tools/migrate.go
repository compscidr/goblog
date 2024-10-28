package tools

import (
	"goblog/auth"
	"goblog/blog"
	"gorm.io/gorm"
	"log"
)

func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(&auth.BlogUser{}, &blog.Post{}, &blog.Tag{}, &auth.AdminUser{}, &blog.Setting{})
	if err != nil {
		log.Println("Error migrating tables: " + err.Error())
		return err
	}
	return nil
}
