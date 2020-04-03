package main

import (
  "github.com/gin-gonic/gin"
)

func main() {
  //structure of this taken from: https://rshipp.com/go-web-api/
  a := &App{}
  a.Initialize("sqlite3", "test.db")
  a.Listen(8000)
  defer a.DB.Close()
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
