package main

//this implements https://jsonapi.org/format/ as best as possible

import (
	"net/http"

  "goblog/auth"
	"goblog/admin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite" // this is the db driver

	"github.com/rs/cors"
)

func main() {

	//https://gorm.io/docs/
	//todo - convert this to a non-local db when not running locally
	db, err := gorm.Open("sqlite3", "test.db")
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&auth.BlogUser{})

	mux := http.NewServeMux()

	auth := auth.New(db)
	admin := admin.New(db)
	mux.HandleFunc("/api/login", auth.LoginHandler)
	mux.HandleFunc("/api/v1/admin", admin.AdminHandler)

	// todo: restrict cors properly to same domain: https://github.com/rs/cors
	// this lets us get a request from localhost:8000 without the web browser
	// bitching about it
	cors := cors.Default().Handler(mux)
	http.ListenAndServe(":7000", cors)

	defer db.Close()
}
