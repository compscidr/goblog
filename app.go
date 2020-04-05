package main

import (
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
}

func (a *App) Listen(httpPort int) {
  r := gin.New()

  //see more at: https://github.com/gin-gonic/gin
  r.GET("/", a.default_route)

  //list all posts and tags
  r.GET("/posts", a.list_posts)
  r.GET("/tags", a.list_tags)

  //retrieve specific post and tags
  r.GET("/posts/:yyyy/:mm/:dd/:slug", a.specific_post)
  r.GET("/tags/:tag", a.specific_tag)

  //create a post or tag
  r.POST("/post", a.create_post)
  r.POST("/tag", a.create_tag)
  //
  // r.DELETE("/posts/:yyyy/:mm/:dd/:slug", delete_post)
  // r.DELETE("/tags/:tag", delete_tag)

	r.Run(":8000") // listen and serve on 0.0.0.0:8080
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
  if err := a.DB.Where("Posted > ? AND Slug = ?", time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), slug).First(&post).Error; err != nil {
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
  title := c.Param("title")
  slug := url.QueryEscape(title)
  a.DB.Create(&Post{
    Title: title,
    Slug: slug,
    Posted: time.Now(),
    Modified: time.Now(),
  })
}

func (a *App) create_tag(c *gin.Context) {
  name := c.Param("tag")
  a.DB.Create(&Tag{Name: name})

  // Read from DB.
  var tag Tag
  a.DB.First(&tag, "name = ?", name)

  c.JSON(http.StatusOK, tag)
}
