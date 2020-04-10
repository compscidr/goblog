package main

import (
  "github.com/jinzhu/gorm"
)

type Post struct {
  gorm.Model
  Title       string
  Slug        string
  Content     string `sql:"type:text;"`
}

type Tag struct {
  gorm.Model
  Name        string
  Slug        string
}
