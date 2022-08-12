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
	//err = db.AutoMigrate(&auth.BlogUser{})
	//if err != nil {
	//	log.Println("Error migrating the BlogUser struct: " + err.Error())
	//	return
	//}
	//err = db.AutoMigrate(&blog.Post{})
	//if err != nil {
	//	log.Println("Error migrating the Post struct: " + err.Error())
	//	return
	//}
	//err = db.AutoMigrate(&blog.Tag{})
	//if err != nil {
	//	log.Println("Error migrating the Tag struct: " + err.Error())
	//	return
	//}
	//err = db.AutoMigrate(&auth.AdminUser{})
	//if err != nil {
	//	log.Println("Error migrating the AdminUser struct: " + err.Error())
	//	return
	//}
}
