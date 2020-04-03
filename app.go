package main

import (
  "strconv"
  "time"
  "net/http"
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

  //list all posts
  r.GET("/posts", a.list_posts)
  //specific post
  r.GET("/posts/:yyyy/:mm/:dd/:slug", a.specific_post)
  //list all tags
  r.GET("/tags", a.list_tags)
  //specifc tag
  r.GET("/tags/:tag", a.specific_tag)

  r.POST("/post", a.create_post)
  // r.POST("/tag", create_tag)
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
