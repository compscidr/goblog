package main

//this implements https://jsonapi.org/format/ as best as possible

import (
	"goblog/admin"
	"goblog/auth"
	"goblog/blog"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/static"
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
	store := cookie.NewStore([]byte("changelater"))
	router.Use(sessions.Sessions("www.jasonernst.com", store))

	auth := auth.New(db)
	admin := admin.New(db, &auth)
	blog := blog.New(db, &auth)

	// todo: restrict cors properly to same domain: https://github.com/rs/cors
	// this lets us get a request from localhost:8000 without the web browser
	// bitching about it
	router.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost", "http://localhost:8000"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE"},
		AllowCredentials: true,
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		// Enable Debugging for testing, consider disabling in production
		Debug: true,
	}))

	//just for testing, remove soon
	router.GET("/api/v1/admin", admin.AdminHandler)

	//all of this is the json api
	router.POST("/api/login", auth.LoginPostHandler)
	router.POST("/api/v1/posts", admin.CreatePost)
	router.PATCH("/api/v1/posts", admin.UpdatePost)
	router.DELETE("/api/v1/posts", admin.DeletePost)
	router.GET("/api/v1/posts/:yyyy/:mm/:dd/:slug", blog.GetPost)
	router.GET("/api/v1/posts", blog.ListPosts)

	//all of this serves html full pages, but re-uses much of the logic of
	//the json API. The json API is tested more easily. Also javascript can
	//served in the html can be used to create and update posts by directly
	//working with the json API.

	//todo - make the template folder configurable by command line arg
	//so that people can pass in their own template folder instead of the default
	router.LoadHTMLGlob("templates/*")

	//if we use true here - it will override the home route and just show files
	router.Use(static.Serve("/", static.LocalFile(".", false)))
	router.GET("/", blog.Home)
	router.GET("/posts/:yyyy/:mm/:dd/:slug", blog.Post)
	router.GET("/login", blog.Login)
	router.GET("/logout", blog.Logout)

	//todo all people to register a template mapping to a "page type"
	router.GET("/posts", blog.Posts)
	router.GET("/presentations", blog.Speaking)
	router.GET("/projects", blog.Projects)
	router.GET("/about", blog.About)

	router.GET("/admin", admin.Admin)

	router.Run("0.0.0.0:7000")

	defer db.Close()
}
