package main

//this implements https://jsonapi.org/format/ as best as possible

import (
	"goblog/admin"
	"goblog/auth"
	"goblog/blog"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite" // this is the db driver

	cors "github.com/rs/cors/wrapper/gin"
)

func main() {

	//https://gorm.io/docs/
	//todo - convert this to a non-local db when not running locally
	db, err := gorm.Open("sqlite3", "test.db")
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&auth.BlogUser{})
	db.AutoMigrate(&blog.Post{})

	//mux := http.NewServeMux()
	router := gin.Default()

	auth := auth.New(db)
	admin := admin.New(db)
	blog := blog.New(db)

	// todo: restrict cors properly to same domain: https://github.com/rs/cors
	// this lets us get a request from localhost:8000 without the web browser
	// bitching about it
	router.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost", "http://localhost:8000"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowCredentials: true,
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		// Enable Debugging for testing, consider disabling in production
		Debug: true,
	}))

	//just for testing, remove soon
	router.GET("/api/v1/admin", admin.AdminHandler)

	router.POST("/api/login", auth.LoginPostHandler)
	router.POST("/api/v1/posts", admin.CreatePost)
	router.PATCH("/api/v1/posts", admin.UpdatePost)

	router.GET("/api/v1/posts/:yyyy/:mm/:dd/:slug", blog.GetPost)
	router.GET("/api/v1/posts", blog.ListPosts)

	router.Run(":7000")

	defer db.Close()
}
