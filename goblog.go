package main

import (
  "time"
  "github.com/gin-gonic/gin"
  "github.com/jinzhu/gorm"
  _ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Post struct {
  gorm.Model
  Title       string
  Slug        string
  Posted      *time.Time
  Modified    *time.Time
}

type Tag struct {
  gorm.Model
  Name        string
}

func main() {

  //https://gorm.io/docs/
  db, err := gorm.Open("sqlite3", "test.db")
  if err != nil {
    panic("failed to connect database")
  }
  defer db.Close()

  // Migrate the schema
  db.AutoMigrate(&Post{})
  db.AutoMigrate(&Tag{})

  r := gin.New()

  //see more at: https://github.com/gin-gonic/gin
  r.GET("/", default_route)

  //list all posts
  r.GET("/posts", list_posts)
  //specific post
  r.GET("/posts/:yyyy/:mm/:dd/:slug", specific_post)
  //list all tags
  r.GET("/tags", list_tags)
  //specifc tag
  r.GET("/tags/:tag", specific_tag)

  r.POST("/post", create_post)
  r.POST("/tag", create_tag)

  r.DELETE("/posts/:yyyy/:mm/:dd/:slug", delete_post)
  r.DELETE("/tags/:tag", delete_tag)

	r.Run(":8000") // listen and serve on 0.0.0.0:8080
}

func default_route(c *gin.Context) {
  c.String(200, "INDEX")
}

func list_posts(c *gin.Context) {
  c.String(200, "POSTS")
}

func specific_post(c *gin.Context) {
  year := c.Param("yyyy")
  month := c.Param("mm")
  day := c.Param("dd")
  slug := c.Param("slug")
  c.String(200, "SPECIFIC POST: " + year + "/" + month + "/" + day + "/" + slug)
}

func list_tags(c *gin.Context) {
  c.String(200, "TAGS")
}

func specific_tag(c *gin.Context) {
  tag := c.Param("tag")
  c.String(200, "SPECIFIC TAG: " + tag)
}

func create_post(c *gin.Context) {
  c.String(200, "CREATE POST")
}

func create_tag(c *gin.Context) {
  c.String(200, "CREATE TAG")
}

func delete_post(c *gin.Context) {
  c.String(200, "DELETE POST")
}

func delete_tag(c *gin.Context) {
  c.String(200, "DELETE TAG")
}
