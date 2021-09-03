package main

//this implements https://jsonapi.org/format/ as best as possible

import (
	"github.com/joho/godotenv"
	"goblog/admin"
	"goblog/auth"
	"goblog/blog"
	"log"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

//Version of the code generated from git describe
var Version = "development"

func main() {
	log.Println("Starting blog version: ", Version)

	err := godotenv.Load(".env")
	if err != nil {
		//fall back to local config
		err = godotenv.Load("local.env")
		if err != nil {
			log.Println("Error loading .env file: " + err.Error())
			return
		}
	}

	database := os.Getenv("database")
	if database != "mysql" && database != "sqlite" {
		log.Println("Database type: " + database + " is not valid. Expecting `mysql` or `sqlite`")
		return
	}

	var db *gorm.DB
	if database == "sqlite" {
		db, err = gorm.Open(sqlite.Open("test.db"))
	} else {
		user := os.Getenv("MYSQL_USER")
		pass := os.Getenv("MYSQL_PASSWORD")
		dbname := os.Getenv("MYSQL_DATABASE")
		dsn := user + ":" + pass + "@tcp(db:3306)/" + dbname + "?charset=utf8mb4&parseTime=True&loc=Local"
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	}
	if err != nil {
		panic("failed to connect database")
	}
	err = db.AutoMigrate(&auth.BlogUser{})
	err = db.AutoMigrate(&blog.Post{})
	err = db.AutoMigrate(&blog.Tag{})

	if err != nil {
		log.Println("Error migrating db")
		return
	}

	router := gin.Default()
	router.Use(CORS())
	store := cookie.NewStore([]byte("changelater"))
	router.Use(sessions.Sessions("www.jasonernst.com", store))

	_auth := auth.New(db, Version)
	_blog := blog.New(db, &_auth, Version)
	_admin := admin.New(db, &_auth, _blog, Version)

	// todo: restrict cors properly to same domain: https://github.com/rs/cors
	// this lets us get a request from localhost:8000 without the web browser
	// bitching about it
	//router.Use(cors.New(cors.Config{
	//	AllowOrigins:     []string{"http://localhost", "http://localhost:8000"},
	//	AllowMethods:     []string{"GET", "POST", "PATCH", "OPTIONS", "DELETE"},
	//	ExposeHeaders: 	  []string{"Content-Length"},
	//	AllowCredentials: true,
	//	AllowAllOrigins:  false,
	//	AllowOriginFunc:  func(origin string) bool { return true },
	//}))

	//all of this is the json api
	router.MaxMultipartMemory = 50 << 20
	router.POST("/api/login", _auth.LoginPostHandler)
	router.POST("/api/v1/posts", _admin.CreatePost)
	router.POST("/api/v1/upload", _admin.UploadFile)
	router.PATCH("/api/v1/posts", _admin.UpdatePost)
	router.DELETE("/api/v1/posts", _admin.DeletePost)
	router.GET("/api/v1/posts/:yyyy/:mm/:dd/:slug", _blog.GetPost)
	router.GET("/api/v1/posts", _blog.ListPosts)

	//all of this serves html full pages, but re-uses much of the logic of
	//the json API. The json API is tested more easily. Also javascript can
	//served in the html can be used to create and update posts by directly
	//working with the json API.

	//todo - make the template folder configurable by command line arg
	//so that people can pass in their own template folder instead of the default
	//https://github.com/gin-gonic/gin/issues/464
	router.LoadHTMLGlob("templates/*.html")

	//if we use true here - it will override the home route and just show files
	router.Use(static.Serve("/", static.LocalFile(".", false)))
	router.GET("/", _blog.Home)
	router.GET("/index.php", _blog.Home)
	router.GET("/posts/:yyyy/:mm/:dd/:slug", _blog.Post)
	router.GET("/admin/posts/:yyyy/:mm/:dd/:slug", _admin.Post)
	router.GET("/tag/:name", _blog.Tag)
	router.GET("/login", _blog.Login)
	router.GET("/logout", _blog.Logout)

	//todo: register a template mapping to a "page type"
	router.GET("/posts", _blog.Posts)
	router.GET("/tags", _blog.Tags)
	router.GET("/presentations", _blog.Speaking)
	router.GET("/research", _blog.Research)
	router.GET("/projects", _blog.Projects)
	router.GET("/about", _blog.About)
	router.GET("/sitemap.xml", _blog.Sitemap)

	router.GET("/admin", _admin.Admin)

	router.NoRoute(_blog.NoRoute)

	router.Run("0.0.0.0:7000")
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Next()
	}
}
