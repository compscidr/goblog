package main

import (
  "log"
  "strconv"
  "time"
  "net/http"
  "net/url"
  "github.com/gin-gonic/gin"
  "github.com/jinzhu/gorm"
  _ "github.com/jinzhu/gorm/dialects/sqlite"
)

type App struct {
  DB *gorm.DB
  R *gin.Engine
}

func (a *App) Initialize(dbDriver string, dbURI string) {

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

  //see more at: https://github.com/gin-gonic/gin
  a.R.GET("/", a.default_route)

  //list all posts and tags
  a.R.GET("/posts", a.list_posts)
  a.R.GET("/tags", a.list_tags)

  //retrieve specific post and tags
  a.R.GET("/posts/:yyyy/:mm/:dd/:slug", a.specific_post)
  a.R.GET("/tags/:tag", a.specific_tag)

  //create a post or tag
  a.R.POST("/post", a.create_post)
  a.R.POST("/tag", a.create_tag)
  //
  // r.DELETE("/posts/:yyyy/:mm/:dd/:slug", delete_post)
  // r.DELETE("/tags/:tag", delete_tag)
}

func (a *App) Listen(httpPort int) {
	a.R.Run(":8000") // listen and serve on 0.0.0.0:8080
}

func (a *App) default_route(c *gin.Context) {
  c.String(200, "INDEX")
}

func (a *App) list_posts(c *gin.Context) {
  var posts []Post
  a.DB.Find(&posts)
  c.JSON(http.StatusOK, posts)
}

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
  if err := a.DB.Where("CreatedAt > ? AND Slug = ?", time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), slug).First(&post).Error; err != nil {
    c.JSON(http.StatusNotFound, "Post Not Found")
    return
  }
  c.JSON(http.StatusOK, post)
}

func (a *App) list_tags(c *gin.Context) {
  var tags []Tag
  a.DB.Find(&tags)
  c.JSON(http.StatusOK, tags)
}

func (a *App) specific_tag(c *gin.Context) {
  var tag Tag
  name := c.Param("tag")
  if err := a.DB.Where("Name = ?", name).First(&tag).Error; err != nil {
    c.JSON(http.StatusNotFound, "Tag Not Found")
    return
  }
  c.JSON(http.StatusOK, tag)
}

func (a *App) create_post(c *gin.Context) {
  title := c.PostForm("title")
  slug := url.QueryEscape(title)
  log.Print("CREATING POST WITH TITLE: " + title + " SLUG: " + slug)
  a.DB.Create(&Post{
    Title: title,
    Slug: slug,
  })

  // Read from DB.
  var post Post
  a.DB.First(&post, "title = ?", title)

  c.JSON(http.StatusOK, post)
}

func (a *App) create_tag(c *gin.Context) {
  name := c.PostForm("name")
  log.Print("CREATING TAG WITH NAME: '" + name + "'")
  a.DB.Create(&Tag{Name: name})

  // Read from DB.
  var tag Tag
  a.DB.First(&tag, "name = ?", name)

  c.JSON(http.StatusOK, tag)
}
