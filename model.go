package main

import (
  "github.com/jinzhu/gorm"
)

type Post struct {
  gorm.Model
  Title       string
  Slug        string
}

type Tag struct {
  gorm.Model
  Name        string
}
