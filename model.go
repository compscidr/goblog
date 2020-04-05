package main

import (
  "time"
  "github.com/jinzhu/gorm"
)

type Post struct {
  gorm.Model
  Title       string
  Slug        string
  Posted      time.Time
  Modified    time.Time
}

type Tag struct {
  gorm.Model
  Name        string
}
