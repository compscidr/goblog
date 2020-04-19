package main

import (
  "crypto/rand"
	"encoding/base64"
  "fmt"
  "io"
  "html/template"
  "log"
  "os"
  "errors"
  "strconv"
  "strings"
  "time"
  "net/http"
  "net/url"
  "github.com/gin-gonic/gin"
  "github.com/google/go-github/github"
  "github.com/gin-gonic/contrib/sessions"
  "github.com/jinzhu/gorm"
  _ "github.com/jinzhu/gorm/dialects/sqlite"
  "golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
)

type App struct {
  DB *gorm.DB
  R *gin.Engine
  OAuthConf *oauth2.Config
}

// https://gist.github.com/dtan4/a3b5027dd3c7d5c5ed3119ea97fb7235
func generateSecretKey() (string, error) {
	b := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, b)

	if err != nil {
		fmt.Println("error:", err)
		return "", err
	}

	return strings.TrimRight(base64.StdEncoding.EncodeToString(b), "="), nil
}

func (a *App) Initialize(dbDriver string, dbURI string) {

  a.OAuthConf = &oauth2.Config{
		//ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
    //ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
    ClientID:     "9a4892a7a4a8c4646225",
		ClientSecret: "cfe1fa8128e81caf81b21645c0784231751b3627",
		Scopes:       []string{"user"},
		Endpoint:     githuboauth.Endpoint,
	}

  //secretKey := os.Getenv("SECRET_KEY_BASE")
  secretKey := ""

	if secretKey == "" {
		sk, err := generateSecretKey()

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		secretKey = sk
	}

  //https://gorm.io/docs/
  db, err := gorm.Open(dbDriver, dbURI)
  if err != nil {
    panic("failed to connect database")
  }
  a.DB = db

  // Migrate the schema
  a.DB.AutoMigrate(&Post{})
  a.DB.AutoMigrate(&Tag{})

  a.R = gin.New()

  store := sessions.NewCookieStore([]byte(secretKey))
	a.R.Use(sessions.Sessions("www.jasonernst.com", store))

  // load helper functions for templates
  a.R.SetFuncMap(template.FuncMap{
		"get_post_url": get_post_url,
	})

  //load templates
  a.R.LoadHTMLGlob("templates/*")

  //see more at: https://github.com/gin-gonic/gin
  //setup the routes
  a.R.GET("/", a.default_route)
  a.R.GET("/posts", a.list_posts)
  a.R.GET("/tags", a.list_tags)
  a.R.GET("/posts/:yyyy/:mm/:dd/:slug", a.specific_post)
  a.R.GET("/tags/:slug", a.specific_tag)
  a.R.POST("/post", a.create_post)
  a.R.POST("/tag", a.create_tag)
  a.R.PUT("/post/:yyyy/:mm/:dd/:slug", a.update_post)
  a.R.PUT("/tag/:slug", a.update_tag)
  a.R.DELETE("/posts/:yyyy/:mm/:dd/:slug", a.delete_post)
  a.R.DELETE("/tags/:slug", a.delete_tag)

  a.R.GET("/signin", func(c *gin.Context) {
		url := a.OAuthConf.AuthCodeURL("www.jasonernst.com", oauth2.AccessTypeOnline)
		c.Redirect(http.StatusMovedPermanently, url)
	})

  a.R.GET("/signout", func(c *gin.Context) {
    session := sessions.Default(c)
		session.Delete("token")
    session.Save()
    c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
	})

  a.R.GET("/callback", func(c *gin.Context) {
		code := c.Query("code")

		token, err := a.OAuthConf.Exchange(oauth2.NoContext, code)

		if err != nil {
			c.String(http.StatusBadRequest, "Error: %s", err)
			return
		}

		session := sessions.Default(c)
		fmt.Println(token.AccessToken)
		session.Set("token", token.AccessToken)
		session.Save()

		c.Redirect(http.StatusMovedPermanently, "/")
	})
}

func (a *App) Listen(httpPort int) {
  //starts actually listening for http requests on this port
	a.R.Run(":8000")
}

// This is essentially the index page for the blog. Need to decide what this
// should show in terms of functionality. Should it show a list of posts with
// their content? Should it just give the clickable titles? etc.
func (a *App) default_route(c *gin.Context) {

  user, err := a.is_admin(c)

  if err == nil {
    c.HTML(
      http.StatusOK,
      "index.html",
      gin.H {
        "title": "Home Page",
        "logged_in": true,
				"login":     user.Login,
				"name":      user.Name,
      },
    )
  } else {
    c.HTML(
      http.StatusOK,
      "index.html",
      gin.H {
        "logged_in": false,
        "title": "Home Page",
        "error": err,
      },
    )
  }
}

func get_post_url(post Post) string {
  year, month, day := post.CreatedAt.Date()
  return fmt.Sprintf("/posts/%d/%02d/%02d/%s", year, month, day, post.Slug)
}

func (a *App) is_admin(c *gin.Context) (*github.User, error) {
  session := sessions.Default(c)
  token := session.Get("token")

  if token == nil {
    return nil, errors.New("Token is nil")
  } else {
    oauthClient := a.OAuthConf.Client(oauth2.NoContext, &oauth2.Token{AccessToken: token.(string)})
    client := github.NewClient(oauthClient)
    user, _, err := client.Users.Get(oauth2.NoContext, "")
    if err != nil {
			return nil, err
		} else if (*user.Login == "compscidr") {
      return user, nil
    } else {
      return nil, errors.New("Not an admin user")
    }
  }
}

// This lists all of the posts in the blog.
// todo: 1) figure out whether to template or not 2) pagination
// 3) if we want to be able to query posts by date, we need to update the router
// for example, all posts from 2020, or all posts from 2020/01
func (a *App) list_posts(c *gin.Context) {
  var posts []Post
  a.DB.Find(&posts)

  //if we want to return json instead
  //c.JSON(http.StatusOK, posts)

  c.HTML(
    http.StatusOK,
    "posts.html",
    gin.H {
      "payload": posts,
    },
  )
}

// Gets a specific post by the year, month, date and slug. Right now it returns
// this as json. need to decide if this should be the case or if it should
// return rendered html
func (a *App) specific_post(c *gin.Context) {
  var post Post
  year, err := strconv.Atoi(c.Param("yyyy"))
  if err != nil {
    c.JSON(http.StatusBadRequest, "Year must be an integer")
    return
  }
  month, err := strconv.Atoi(c.Param("mm"))
  if err != nil {
    c.JSON(http.StatusBadRequest, "Month must be an integer")
    return
  }
  day, err := strconv.Atoi(c.Param("dd"))
  if err != nil {
    c.JSON(http.StatusBadRequest, "Day must be an integer")
    return
  }
  slug := c.Param("slug")
  if err := a.DB.Where("created_at > ? AND slug = ?", time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), slug).First(&post).Error; err != nil {
    c.JSON(http.StatusNotFound, "Post Not Found")
    return
  }

  //c.JSON(http.StatusOK, post)
  c.HTML(
    http.StatusOK,
    "post.html",
    gin.H {
      "payload": post,
    },
  )
}

//
func (a *App) list_tags(c *gin.Context) {
  var tags []Tag
  a.DB.Find(&tags)
  c.JSON(http.StatusOK, tags)
}

func (a *App) specific_tag(c *gin.Context) {
  var tag Tag
  name := c.Param("tag")
  if err := a.DB.Where("name = ?", name).First(&tag).Error; err != nil {
    c.JSON(http.StatusNotFound, "Tag Not Found")
    return
  }
  c.JSON(http.StatusOK, tag)
}

func (a *App) create_post(c *gin.Context) {
  title := c.PostForm("Title")
  content := c.PostForm("Content")
  slug := url.QueryEscape(strings.Replace(title, " ", "-", -1))
  log.Print("CREATING POST WITH TITLE: " + title + " SLUG: " + slug)
  a.DB.Create(&Post{
    Title: title,
    Slug: slug,
    Content: content,
  })

  // Read from DB.
  var post Post
  a.DB.First(&post, "title = ?", title)

  c.JSON(http.StatusOK, post)
}

func (a *App) update_post(c *gin.Context) {
  id := c.PostForm("ID")
  title := c.PostForm("Title")
  slug := url.QueryEscape(strings.Replace(title, " ", "-", -1))
  log.Print("UPDATING POST ID: " + id  + " WITH TITLE: " + title + " AND SLUG: ")

  post := &Post {
    Title: title,
    Slug: slug,
  }

  a.DB.Model(&post).Where("id = ?", id).Updates(&post)
  c.JSON(http.StatusOK, post)
}

func (a *App) create_tag(c *gin.Context) {
  name := c.PostForm("Name")
  log.Print("CREATING TAG WITH NAME: '" + name + "'")
  a.DB.Create(&Tag{Name: name})

  // Read from DB.
  var tag Tag
  a.DB.First(&tag, "name = ?", name)

  c.JSON(http.StatusOK, tag)
}

func (a *App) update_tag(c *gin.Context) {
  id := c.PostForm("ID")
  name := c.PostForm("Name")
  log.Print("RENAMING TAG ID: " + id + " TO '" + name + "'")

  tag := &Tag {
    Name: name,
  }

  a.DB.Model(&tag).Where("id = ?", id).Updates(&tag)
  c.JSON(http.StatusOK, tag)
}

func (a *App) delete_post(c *gin.Context) {
  var post Post
  year, err := strconv.Atoi(c.Param("yyyy"))
  if err != nil {
    c.JSON(http.StatusBadRequest, "Year must be an integer")
    return
  }
  month, err := strconv.Atoi(c.Param("mm"))
  if err != nil {
    c.JSON(http.StatusBadRequest, "Month must be an integer")
    return
  }
  day, err := strconv.Atoi(c.Param("dd"))
  if err != nil {
    c.JSON(http.StatusBadRequest, "Day must be an integer")
    return
  }
  slug := c.Param("slug")
  if err := a.DB.Where("created_at > ? AND slug = ?", time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), slug).First(&post).Error; err != nil {
    c.JSON(http.StatusNotFound, "Post Not Found")
    return
  }
  a.DB.Delete(&post)

  c.JSON(http.StatusOK, post)
}

func (a *App) delete_tag(c *gin.Context) {
  var tag Tag
  slug := c.Param("slug")
  if err := a.DB.Where("slug = ?", slug).First(&tag).Error; err != nil {
    c.JSON(http.StatusNotFound, "Tag Not Found")
    return
  }
  a.DB.Delete(&tag)
  c.JSON(http.StatusOK, tag)
}
