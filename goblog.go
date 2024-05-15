package main

//this implements https://jsonapi.org/format/ as best as possible

import (
	"github.com/joho/godotenv"
	"goblog/admin"
	"goblog/auth"
	"goblog/blog"
	"goblog/tools"
	"goblog/wizard"
	"log"
	"os"
	"strconv"
	"syscall"

	"github.com/fvbock/endless"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Version of the code generated from git describe
var Version = "development"

func setup_wizard() {
	_wizard := wizard.New(Version)
	router := gin.Default()
	router.Use(CORS())
	store := cookie.NewStore([]byte("changelater"))
	router.Use(sessions.Sessions("www.jasonernst.com", store))
	router.LoadHTMLGlob("templates/*.html")
	router.GET("/", _wizard.Landing)
	router.GET("/wizard", _wizard.SaveToken)
	router.Use(static.Serve("/", static.LocalFile("www", false)))
	router.GET("/login", _wizard.LoginCode)
	// router.Run("0.0.0.0:7000")
	server := endless.NewServer(":7000", router)
	server.BeforeBegin = func(add string) {
		log.Printf("Actual pid is %d", syscall.Getpid())
		pid := syscall.Getpid()
		f, err := os.Create("/tmp/goblog.pid")
		if err != nil {
			log.Println("Unable to create /tmp/goblog.pid")
			return
		}
		_, err = f.WriteString(strconv.Itoa(pid))
		if err != nil {
			log.Println("Unable to write to /tmp/goblog.pid")
			return
		}
		err = f.Close()
		if err != nil {
			log.Println("Unable to close /tmp/goblog.pid")
			return
		}
	}
	server.ListenAndServe()
}

func main() {
	log.Println("Starting blog version: ", Version)

	err := godotenv.Load(".env")
	if err != nil {
		setup_wizard()
		err := godotenv.Load(".env")
		if err != nil {
			log.Println("Failed to read the .env file after the wizard, can't proceed")
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
		db_file := os.Getenv("sqlite_db")
		db, err = gorm.Open(sqlite.Open(db_file), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		})
		if err != nil {
			panic("failed to open sqlite db: " + db_file)
		}
		log.Println("opened sqlite db")
	} else {
		user := os.Getenv("MYSQL_USER")
		pass := os.Getenv("MYSQL_PASSWORD")
		dbname := os.Getenv("MYSQL_DATABASE")
		dsn := user + ":" + pass + "@tcp(db:3306)/" + dbname + "?charset=utf8mb4&parseTime=True&loc=Local"
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			panic("failed to connect to mysql db")
		}
		log.Println("connected to mysql db")
	}

	tools.Migrate(db)

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
	router.PATCH("/api/v1/publish/:id", _admin.PublishPost)
	router.PATCH("/api/v1/draft/:id", _admin.DraftPost)
	router.DELETE("/api/v1/posts", _admin.DeletePost)
	router.GET("/api/v1/posts/:yyyy/:mm/:dd/:slug", _blog.GetPost)
	router.GET("/api/v1/posts", _blog.ListPosts)
	router.GET("/api/v1/setting/:slug", _admin.GetSetting)
	router.POST("/api/v1/setting", _admin.UpdateSetting)
	router.PATCH("/api/v1/setting", _admin.UpdateSetting)

	//all of this serves html full pages, but re-uses much of the logic of
	//the json API. The json API is tested more easily. Also javascript can
	//served in the html can be used to create and update posts by directly
	//working with the json API.

	//todo - make the template folder configurable by command line arg
	//so that people can pass in their own template folder instead of the default
	//https://github.com/gin-gonic/gin/issues/464
	router.LoadHTMLGlob("templates/*.html")

	//if we use true here - it will override the home route and just show files
	router.Use(static.Serve("/", static.LocalFile("www", false)))
	router.GET("/", _blog.Home)
	router.GET("/index.php", _blog.Home)
	router.GET("/posts/:yyyy/:mm/:dd/:slug", _blog.Post)
	// lets posts work with our without the word posts in front
	router.GET("/:yyyy/:mm/:dd/:slug", _blog.Post)
	router.GET("/admin/posts/:yyyy/:mm/:dd/:slug", _admin.Post)
	router.GET("/tag/:name", _blog.Tag)
	router.GET("/login", _blog.Login)
	router.GET("/logout", _blog.Logout)

	//todo: register a template mapping to a "page type"
	router.GET("/posts", _blog.Posts)
	router.GET("/blog", _blog.Posts)
	router.GET("/tags", _blog.Tags)
	router.GET("/presentations", _blog.Speaking)
	router.GET("/research", _blog.Research)
	router.GET("/projects", _blog.Projects)
	router.GET("/about", _blog.About)
	router.GET("/sitemap.xml", _blog.Sitemap)
	router.GET("/archives", _blog.Archives)
	// lets old WordPress stuff stored at wp-content/uploads work
	router.Use(static.Serve("/wp-content", static.LocalFile("www", false)))

	router.GET("/admin", _admin.Admin)
	router.GET("/admin/dashboard", _admin.AdminDashboard)
	router.GET("/admin/posts", _admin.AdminPosts)
	router.GET("/admin/newpost", _admin.AdminNewPost)
	router.GET("/admin/settings", _admin.AdminSettings)

	router.NoRoute(_blog.NoRoute)

	err = endless.ListenAndServe(":7000", router)
	if err != nil {
		log.Println("Error running goblog server: " + err.Error())
	}
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
