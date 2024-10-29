package main

//this implements https://jsonapi.org/format/ as best as possible

import (
	"fmt"
	scholar "github.com/compscidr/scholar"
	"github.com/joho/godotenv"
	"goblog/admin"
	"goblog/auth"
	"goblog/blog"
	"goblog/tools"
	"goblog/wizard"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Version of the code generated from git describe
var Version = "development"

type goblog struct {
	_wizard            *wizard.Wizard
	_blog              *blog.Blog
	_auth              *auth.Auth
	_admin             *admin.Admin
	sessionKey         string
	router             *gin.Engine
	handlersRegistered bool
}

func envFilePresent() bool {
	_, err := os.Stat(".env")
	if err != nil {
		return false
	}
	return true
}

func isAuthConfigured() bool {
	envFile, err := godotenv.Read(".env")
	if err != nil {
		log.Println("Couldn't read the .env file: " + err.Error())
		return false
	}
	if envFile["client_id"] == "" || envFile["client_secret"] == "" {
		return false
	}
	return true
}

func attemptConnectDb() *gorm.DB {
	envFile, err := godotenv.Read(".env")
	if err != nil {
		log.Println("Couldn't read the .env file: " + err.Error())
		return nil
	}
	database := envFile["database"]
	if database != "mysql" && database != "sqlite" {
		log.Println("Database type: " + database + " is not valid. Expecting `mysql` or `sqlite`")
		return nil
	}

	if database == "sqlite" {
		db_file := envFile["sqlite_db"]
		db, err := gorm.Open(sqlite.Open(db_file), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		})
		if err != nil {
			log.Println("Error opening sqlite db: " + err.Error())
			return nil
		}
		log.Println("opened sqlite db")
		return db
	} else {
		host := envFile["MYSQL_HOST"]
		port := envFile["MYSQL_PORT"]
		user := envFile["MYSQL_USER"]
		pass := envFile["MYSQL_PASSWORD"]
		dbname := envFile["MYSQL_DATABASE"]
		dsn := user + ":" + pass + "@tcp(" + host + ":" + port + ")/" + dbname + "?charset=utf8mb4&parseTime=True&loc=Local"
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Println("Error opening mysql db: " + err.Error())
			return nil
		}
		log.Println("connected to mysql db")
		return db
	}
}

// depending on if the env file is present or not, we will show the wizard or the main site
func (g goblog) rootHandler(c *gin.Context) {
	if !envFilePresent() {
		log.Println("Root handler: No .env file found")
		c.HTML(http.StatusOK, "wizard_db.html", gin.H{
			"version": Version,
			"title":   "GoBlog Install Wizard",
		})
		return
	} else {
		log.Println("Root handler:  Found .env file")
		envFile, err := godotenv.Read(".env")
		if err != nil {
			c.HTML(http.StatusOK, "wizard_db.html", gin.H{
				"version": Version,
				"title":   "GoBlog Install Wizard",
				"errors":  "Couldn't read the .env file: " + err.Error(),
			})
			return
		}
		fmt.Println(envFile)
		// detect if the database hasn't be configured yet
		if (envFile["database"] != "mysql") && (envFile["database"] != "sqlite") {
			log.Println("Root handler:  Database is not configured, redirecting to db wizard")
			c.HTML(http.StatusOK, "wizard_db.html", gin.H{
				"version": Version,
				"title":   "GoBlog Install Wizard",
			})
			return
		}
		log.Println("Root handler: Database is configured")
		if g._wizard.IsDbNil() {
			log.Println("Root handler: Database is nil - need to connect")
			db := attemptConnectDb()
			if db == nil {
				log.Println("Root handler: Couldn't connect to the database, showing db wizard")
				// show the wizard and get them to re-enter the db info with an error message
				c.HTML(http.StatusOK, "wizard_db.html", gin.H{
					"version": Version,
					"title":   "GoBlog Install Wizard",
					"errors":  "Couldn't connect to the database",
				})
				return
			}
			err := tools.Migrate(db)
			if err != nil {
				log.Println("Root handler: Couldn't migrate the database: " + err.Error())
				// show the wizard with an error message saying the db isn't compatible and let them file a gh ticket
				c.HTML(http.StatusOK, "wizard_db.html", gin.H{
					"version": Version,
					"title":   "GoBlog Install Wizard",
					"errors":  "Failed to Migrate the database: " + err.Error(),
				})
				return
			}
			log.Println("Root handler: Migrated the database, updating the db in the blog, auth, admin, and wizard")
			g._blog.UpdateDb(db)
			g._auth.UpdateDb(db)
			g._admin.UpdateDb(db)
			g._wizard.UpdateDb(db)

			if !isAuthConfigured() {
				g._wizard.Landing(c)
				return
			}
			g.addRoutes()
			g._blog.Home(c)
		} else {
			if !isAuthConfigured() {
				g._wizard.Landing(c)
				return
			}
			g._blog.Home(c)
			return
		}
	}
}

func (g goblog) loginHandler(c *gin.Context) {
	if !envFilePresent() {
		log.Println("Root handler: No .env file found")
		c.HTML(http.StatusOK, "wizard_db.html", gin.H{
			"version": Version,
			"title":   "GoBlog Install Wizard",
		})
		return
	}
	if g._wizard.IsDbNil() {
		log.Println("Wizard db is nil in loginHandler")
		c.HTML(http.StatusOK, "wizard_db.html", gin.H{
			"version": Version,
			"title":   "GoBlog Install Wizard",
			"errors":  "Database is not configured",
		})
		return
	}
	if !isAuthConfigured() {
		err := g._wizard.LoginCode(c)
		if err != nil {
			c.HTML(http.StatusOK, "wizard_auth.html", gin.H{
				"version": Version,
				"title":   "GoBlog Install Wizard",
				"errors":  "Couldn't get the login code: " + err.Error(),
			})
			return
		} else {
			g.addRoutes()
			g._blog.Home(c)
		}
	} else {
		g._blog.Login(c)
	}
}

func main() {
	log.Println("Starting blog version: ", Version)
	var sessionKey string
	var db *gorm.DB = nil
	if !envFilePresent() {
		log.Println("No .env file found, creating one with a new session key")
		sessionKey = uuid.New().String()
		f, err := os.Create(".env")
		if err != nil {
			log.Println("Couldn't create the .env file: " + err.Error())
			return
		}
		_, err = f.WriteString("SESSION_KEY=" + sessionKey + "\n")
		err = f.Close()
		if err != nil {
			log.Println("Couldn't close the .env file: " + err.Error())
			return
		}
	} else {
		log.Println("Found .env file")
		envFile, err := godotenv.Read(".env")
		if err != nil {
			log.Println("Couldn't read the .env file: " + err.Error())
			return
		}
		sessionKey = envFile["SESSION_KEY"]
		if (sessionKey == "") || (len(sessionKey) != 36) {
			log.Println("No session key found or it's invalid, creating a new one")
			sessionKey = uuid.New().String()
			f, err := os.OpenFile(".env", os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				log.Println("Couldn't open the .env file: " + err.Error())
				return
			}
			_, err = f.WriteString("SESSION_KEY=" + sessionKey + "\n")
			err = f.Close()
			if err != nil {
				log.Println("Couldn't close the .env file: " + err.Error())
				return
			}
			log.Println("New Session key: ", sessionKey)
		} else {
			log.Println("Found Session key: ", sessionKey)
		}

		// database is configured, lets try to connect now
		if (envFile["database"] == "mysql") || (envFile["database"] == "sqlite") {
			log.Println("Database is configured, trying to connect")
			db = attemptConnectDb()
			if db == nil {
				log.Println("Couldn't connect to the database")
				return
			}
			err := tools.Migrate(db)
			if err != nil {
				log.Println("Couldn't migrate the database: " + err.Error())
				return
			}
		} else {
			log.Println("Database is not configured, should get routed to wizard")
		}
	}

	_auth := auth.New(db, Version)
	_sch := scholar.New("profiles.json", "articles.json")
	_blog := blog.New(db, &_auth, Version, _sch)
	_admin := admin.New(db, &_auth, &_blog, Version)
	_wizard := wizard.New(db, Version)

	// setup the minimal router at the start to support both the wizard and the main server once the wizard is done
	router := gin.Default()

	goblog := goblog{
		_wizard:    &_wizard,
		_blog:      &_blog,
		_auth:      &_auth,
		_admin:     &_admin,
		sessionKey: sessionKey,
		router:     router,
	}

	router.Use(CORS())
	store := cookie.NewStore([]byte(sessionKey))
	hostname, err := os.Hostname()
	router.Use(sessions.Sessions(hostname, store))
	log.Println("Session key: ", sessionKey)
	log.Println("Hostname: ", hostname)
	//todo - make the template folder configurable by command line arg
	//so that people can pass in their own template folder instead of the default
	//https://github.com/gin-gonic/gin/issues/464
	router.LoadHTMLGlob("templates/*.html")
	router.GET("/", goblog.rootHandler)
	router.GET("/login", goblog.loginHandler)
	router.GET("/wizard", goblog._wizard.SaveToken)
	router.POST("/wizard_settings", goblog._wizard.Settings)
	router.POST("/wizard_db", updateDB)
	router.POST("/test_db", testDB)
	//if we use true here - it will override the home route and just show files
	router.Use(static.Serve("/", static.LocalFile("www", false)))
	if err != nil {
		log.Println("Couldn't get the hostname")
		return
	}

	if db != nil {
		goblog.addRoutes()
	}

	err = router.Run(":7000")
	if err != nil {
		log.Println("Error running goblog server: " + err.Error())
	}
}

func (g goblog) addRoutes() {
	if g.handlersRegistered {
		log.Println("Handlers already registered")
		return
	}
	g.handlersRegistered = true
	log.Println("Adding main blog routes")
	//all of this is the json api
	g.router.MaxMultipartMemory = 50 << 20
	g.router.POST("/api/login", g._auth.LoginPostHandler)
	g.router.POST("/api/v1/posts", g._admin.CreatePost)
	g.router.POST("/api/v1/upload", g._admin.UploadFile)
	g.router.PATCH("/api/v1/posts", g._admin.UpdatePost)
	g.router.PATCH("/api/v1/publish/:id", g._admin.PublishPost)
	g.router.PATCH("/api/v1/draft/:id", g._admin.DraftPost)
	g.router.DELETE("/api/v1/posts", g._admin.DeletePost)
	g.router.GET("/api/v1/posts/:yyyy/:mm/:dd/:slug", g._blog.GetPost)
	g.router.GET("/api/v1/posts", g._blog.ListPosts)
	g.router.GET("/api/v1/setting/:slug", g._admin.GetSetting)
	g.router.GET("/api/v1/settings", g._admin.GetSettings)
	g.router.POST("/api/v1/setting", g._admin.AddSetting)
	g.router.PATCH("/api/v1/settings", g._admin.UpdateSettings)

	//all of this serves html full pages, but re-uses much of the logic of
	//the json API. The json API is tested more easily. Also javascript can
	//served in the html can be used to create and update posts by directly
	//working with the json API.
	g.router.GET("/index.php", g._blog.Home)
	g.router.GET("/posts/:yyyy/:mm/:dd/:slug", g._blog.Post)
	// lets posts work with our without the word posts in front
	g.router.GET("/:yyyy/:mm/:dd/:slug", g._blog.Post)
	g.router.GET("/admin/posts/:yyyy/:mm/:dd/:slug", g._admin.Post)
	g.router.GET("/tag/:name", g._blog.Tag)
	g.router.GET("/logout", g._blog.Logout)

	//todo: register a template mapping to a "page type"
	g.router.GET("/posts", g._blog.Posts)
	g.router.GET("/blog", g._blog.Posts)
	g.router.GET("/tags", g._blog.Tags)
	g.router.GET("/presentations", g._blog.Speaking)
	g.router.GET("/research", g._blog.Research)
	g.router.GET("/projects", g._blog.Projects)
	g.router.GET("/about", g._blog.About)
	g.router.GET("/sitemap.xml", g._blog.Sitemap)
	g.router.GET("/archives", g._blog.Archives)
	// lets old WordPress stuff stored at wp-content/uploads work
	g.router.Use(static.Serve("/wp-content", static.LocalFile("www", false)))

	g.router.GET("/admin", g._admin.Admin)
	g.router.GET("/admin/dashboard", g._admin.AdminDashboard)
	g.router.GET("/admin/posts", g._admin.AdminPosts)
	g.router.GET("/admin/newpost", g._admin.AdminNewPost)
	g.router.GET("/admin/settings", g._admin.AdminSettings)

	g.router.NoRoute(g._blog.NoRoute)
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

// parse the form which should have passed the db info
// and then create the env file with it
func updateDB(c *gin.Context) {
	err := c.Request.ParseForm()
	if err != nil {
		log.Println("Couldn't parse the form: " + err.Error())
		return
	}
	f, err := os.Create(".env")
	if err != nil {
		c.HTML(http.StatusOK, "wizard_db.html", gin.H{
			"version": Version,
			"title":   "GoBlog Install Wizard",
			"errors":  "Couldn't create the .env file: " + err.Error(),
		})
		return
	}
	defer f.Close()
	db_type := c.PostForm("dbtype")
	if (db_type != "mysql") && (db_type != "sqlite") {
		c.HTML(http.StatusOK, "wizard_db.html", gin.H{
			"version": Version,
			"title":   "GoBlog Install Wizard",
			"errors":  "Invalid database type",
		})
		return
	}
	if db_type == "mysql" {
		host := c.PostForm("mysql_host")
		user := c.PostForm("mysql_user")
		pass := c.PostForm("mysql_pass")
		dbname := c.PostForm("mysql_dbname")
		_, err = f.WriteString("database=mysql\n")
		if err != nil {
			c.HTML(http.StatusOK, "wizard_db.html", gin.H{
				"version": Version,
				"title":   "GoBlog Install Wizard",
				"errors":  "Couldn't write to the .env file: " + err.Error(),
			})
			return
		}
		_, err = f.WriteString("MYSQL_HOST=" + host + "\n")
		if err != nil {
			c.HTML(http.StatusOK, "wizard_db.html", gin.H{
				"version": Version,
				"title":   "GoBlog Install Wizard",
				"errors":  "Couldn't write to the .env file: " + err.Error(),
			})
			return
		}
		_, err = f.WriteString("MYSQL_USER=" + user + "\n")
		if err != nil {
			c.HTML(http.StatusOK, "wizard_db.html", gin.H{
				"version": Version,
				"title":   "GoBlog Install Wizard",
				"errors":  "Couldn't write to the .env file: " + err.Error(),
			})
			return
		}
		_, err = f.WriteString("MYSQL_PASSWORD=" + pass + "\n")
		if err != nil {
			c.HTML(http.StatusOK, "wizard_db.html", gin.H{
				"version": Version,
				"title":   "GoBlog Install Wizard",
				"errors":  "Couldn't write to the .env file: " + err.Error(),
			})
			return
		}
		_, err = f.WriteString("MYSQL_DATABASE=" + dbname + "\n")
		if err != nil {
			c.HTML(http.StatusOK, "wizard_db.html", gin.H{
				"version": Version,
				"title":   "GoBlog Install Wizard",
				"errors":  "Couldn't write to the .env file: " + err.Error(),
			})
			return
		}
	} else {
		db_file := c.PostForm("sqlite_file")
		_, err = f.WriteString("database=sqlite\n")
		if err != nil {
			c.HTML(http.StatusOK, "wizard_db.html", gin.H{
				"version": Version,
				"title":   "GoBlog Install Wizard",
				"errors":  "Couldn't write to the .env file: " + err.Error(),
			})
			return
		}
		_, err = f.WriteString("sqlite_db=" + db_file + "\n")
		if err != nil {
			c.HTML(http.StatusOK, "wizard_db.html", gin.H{
				"version": Version,
				"title":   "GoBlog Install Wizard",
				"errors":  "Couldn't write to the .env file: " + err.Error(),
			})
			return
		}
	}

	// if we make it this far, success, redirect to /
	c.Redirect(http.StatusSeeOther, "/")
}

func testDB(c *gin.Context) {
	err := c.Request.ParseForm()
	if err != nil {
		log.Println("Couldn't parse the form: " + err.Error())
		return
	}
	for key, value := range c.Request.PostForm {
		log.Println("key: ", key, " value: ", value)
	}

	db_type := c.PostForm("dbtype")
	if (db_type != "mysql") && (db_type != "sqlite") {
		log.Println("Invalid database type: " + db_type)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid database type",
		})
		return
	}
	if db_type == "mysql" {
		host := c.PostForm("mysql_host")
		port := c.PostForm("mysql_port")
		user := c.PostForm("mysql_user")
		pass := c.PostForm("mysql_pass")
		dbname := c.PostForm("mysql_dbname")
		dsn := user + ":" + pass + "@tcp(" + host + ":" + port + ")/" + dbname + "?charset=utf8mb4&parseTime=True&loc=Local"
		_, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Println("Couldn't connect to the database: " + err.Error())
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Couldn't connect to the database: " + err.Error(),
			})
			return
		} else {
			log.Println("Connected to the database")
			c.JSON(http.StatusOK, gin.H{
				"success": "Connected to the database",
			})
			return
		}
	} else {
		db_file := c.PostForm("sqlite_db")
		_, err := gorm.Open(sqlite.Open(db_file), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		})
		if err != nil {
			log.Println("Couldn't connect to the database: " + err.Error())
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Couldn't connect to the database: " + err.Error(),
			})
			return
		} else {
			log.Println("Connected to the database")
			c.JSON(http.StatusOK, gin.H{
				"success": "Connected to the database",
			})
			return
		}
	}
}
