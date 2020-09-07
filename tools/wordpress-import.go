package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite" // this is the db driver
	"goblog/auth"
	"goblog/blog"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// https://tutorialedge.net/golang/golang-mysql-tutorial/
type Post struct {
	ID      int       `json:"ID"`
	Author  int       `json:"post_author"`
	Date    time.Time `json:"post_date"`
	DateGmt time.Time `json:"post_date_gmt"`
	Content string    `json:"post_content"`
	Title   string    `json:"post_title"`
	Category int      `json:"post_category"`
	Excerpt string `json:"post_excerpt"`
	Status string `json:"post_status"`
	CommentStatus string `json:"comment_status"`
	PingStatus string `json:"ping_status"`
	PostPassword string `json:"post_password"`
	Slug string `json:"post_name"`
	ToPing string `json:"to_ping"`
	Pinged string `json:"pinged"`
	Modified time.Time `json:"post_modified"`
	ModifiedGMT time.Time `json:"post_modified_gmt"`
	ContentFiltered string `json:"post_content_filtered"`
	Parent int `json:"post_parent"`
	GUID string `json:"guid"`
	MenuOrder int `json:"menu_order"`
	Type string `json:"post_type"`
	MimeType string `json:"post_mime_type"`
	CommentCount int `json:"comment_count"`
}

type TermRelation struct {
	ObjectID int `json:"object_id"`
	TermTaxonomyID int `json:"term_taxonomy_id"`
	TermOrder int `json:"term_order"`
}

type Term struct {
	TermID int `json:"term_id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	TermGroup int `json:"term_group"`
}

/**
 * Gets a list of tag IDs for a given post ID
 */
func getTags(id int, db * sql.DB) []blog.Tag {
	var tags []blog.Tag
	results, err := db.Query("SELECT * FROM `jason_term_relationships` WHERE object_id = '" + strconv.Itoa(id) + "'")
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	for results.Next() {
		var termRelation TermRelation
		err = results.Scan(&termRelation.ObjectID, &termRelation.TermTaxonomyID, &termRelation.TermOrder)
		term := getTagName(termRelation.TermTaxonomyID, db)
		if term != nil {
			var tag blog.Tag
			tag.Name = term.Name
			tags = append(tags, tag)
		}
	}
	results.Close()
	//fmt.Println(tags)
	return tags
}

func getTagName(id int, db * sql.DB) *Term {
	results, err := db.Query("SELECT * FROM `jason_terms` WHERE term_id = '" + strconv.Itoa(id) + "'")
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	for results.Next() {
		var term Term
		err = results.Scan(&term.TermID, &term.Name, &term.Slug, &term.TermGroup)
		results.Close()
		//fmt.Println(term.Name)
		return &term
	}
	return nil
}

func safeSlug(slug string) string {
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "/", "")
	return url.QueryEscape(slug)
}

func main() {
	sdb, serr := gorm.Open("sqlite3", "test.db")
	if serr != nil {
		panic("failed to connect database")
	}
	sdb.AutoMigrate(&auth.BlogUser{})
	sdb.AutoMigrate(&blog.Post{})
	sdb.AutoMigrate(&blog.Tag{})

	// https://pliutau.com/working-with-db-time-in-go/
	db, err := sql.Open("mysql", "root:password@tcp(127.0.0.1:3306)/wordpress?parseTime=true")

	if err != nil {
		panic(err.Error())
	}

	results, err := db.Query("select * from jason_posts where post_status = 'publish' and post_type = 'post' order by post_date DESC")
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	for results.Next() {
		var post Post
		err = results.Scan(&post.ID, &post.Author, &post.Date, &post.DateGmt, &post.Content, &post.Title,
			&post.Category, &post.Excerpt, &post.Status, &post.CommentStatus, &post.PingStatus, &post.PostPassword,
			&post.Slug, &post.ToPing, &post.Pinged, &post.Modified, &post.ModifiedGMT, &post.ContentFiltered,
			&post.Parent, &post.GUID, &post.MenuOrder, &post.Type, &post.MimeType, &post.CommentCount)
		if err != nil {
			panic(err.Error()) // proper error handling instead of panic in your app
		}

		var newPost blog.Post
		newPost.Slug = safeSlug(post.Slug)
		newPost.CreatedAt = post.Date
		newPost.UpdatedAt = post.Modified
		newPost.Title = post.Title
		newPost.Content = post.Content
		newPost.Tags = getTags(post.ID, db)

		// fix: all code blocks:
		newPost.Content = strings.ReplaceAll(newPost.Content, "<code>", "```")
		newPost.Content = strings.ReplaceAll(newPost.Content, "</code>", "```")
		newPost.Content = strings.ReplaceAll(newPost.Content, "<p>", "")
		newPost.Content = strings.ReplaceAll(newPost.Content, "</p>", "")
		newPost.Content = strings.ReplaceAll(newPost.Content, "<div class=\"clearer\">&nbsp;</div>", "")
		newPost.Content = strings.ReplaceAll(newPost.Content, "<div class=\"image\" style=\"float:right;\">", "")
		newPost.Content = strings.ReplaceAll(newPost.Content, "<div class=\"snippet\">", "")
		newPost.Content = strings.ReplaceAll(newPost.Content, "<div class=\"snippet-shell\">", "")
		newPost.Content = strings.ReplaceAll(newPost.Content, "<!--more-->", "")
		newPost.Content = strings.ReplaceAll(newPost.Content, "</div>", "")
		newPost.Content = strings.ReplaceAll(newPost.Content, "http://jasonernst.com/wp-content", "")
		newPost.Content = strings.ReplaceAll(newPost.Content, "http://www.jasonernst.com/wp-content", "")

		sdb.Create(&newPost)
		fmt.Print(".")

		//log.Printf(newPost)
		//fmt.Println(newPost)
	}
	defer results.Close()
	defer sdb.Close()
	defer db.Close()
}
