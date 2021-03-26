package blog

import (
	"errors"
	"goblog/auth"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/joho/godotenv"

	"github.com/ikeikeikeike/go-sitemap-generator/stm"
)

// Blog API handles non-admin functions of the blog like listing posts, tags
// comments, etc.
type Blog struct {
	db      *gorm.DB
	auth    auth.IAuth
	Version string
}

//New constructs an Admin API
func New(db *gorm.DB, auth auth.IAuth, version string) Blog {
	api := Blog{db, auth, version}
	return api
}

//Generic Functions (not JSON or HTML)
func (b Blog) GetPosts() []Post {
	var posts []Post
	b.db.Preload("Tags").Order("created_at desc").Find(&posts)
	return posts
}

func (b Blog) getTags() []Tag {
	var tags []Tag
	b.db.Preload("Posts").Order("name asc").Find(&tags)
	return tags
}

func (b Blog) GetPostObject(c *gin.Context) (*Post, error) {
	var post Post
	year, err := strconv.Atoi(c.Param("yyyy"))
	if err != nil {
		return nil, errors.New("year must be an integer")
	}
	month, err := strconv.Atoi(c.Param("mm"))
	if err != nil {
		return nil, errors.New("month must be an integer")
	}
	day, err := strconv.Atoi(c.Param("dd"))
	if err != nil {
		return nil, errors.New("day must be an integer")
	}
	slug := c.Param("slug")
	slug = url.QueryEscape(slug)

	log.Println("Looking for post: ", year, "/", month, "/", day, "/", slug)

	if err := b.db.Preload("Tags").Where("created_at > ? AND slug LIKE ?", time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), slug).First(&post).Error; err != nil {
		return nil, errors.New("No post at " + strconv.Itoa(year) + "/" + strconv.Itoa(month) + "/" + strconv.Itoa(day) + "/" + slug)
	}

	//b.db.Model(&post).Related(&post.Tags, "Tags")
	log.Println("Found: ", post.Title, " TAGS: ", post.Tags)
	return &post, nil
}

func (b Blog) getPostByParams(year int, month int, day int, slug string) (*Post, error) {
	log.Println("trying: " + strconv.Itoa(year) + "/" + strconv.Itoa(month) + "/" + strconv.Itoa(day) + "/" + slug)
	var post Post
	slug = url.QueryEscape(slug)
	if err := b.db.Preload("Tags").Where("created_at > ? AND slug LIKE ?", time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), slug).First(&post).Error; err != nil {
		log.Println("NOT FOUND")
		return nil, errors.New("No post at " + strconv.Itoa(year) + "/" + strconv.Itoa(month) + "/" + strconv.Itoa(day) + "/" + slug)
	}
	log.Println("Found: ", post.Title, " TAGS: ", post.Tags)
	return &post, nil
}

func (b Blog) getPostsByTag(c *gin.Context) ([]Post, error) {
	var posts []Post
	var tag Tag
	name := c.Param("name")
	if err := b.db.Where("name = ?", name).First(&tag).Error; err != nil {
		return nil, errors.New("No tag named " + name)
	}

	b.db.Model(&tag).Order("created_at desc").Related(&posts, "Posts")
	log.Print("POSTS: ", posts)
	return posts, nil
}

//////JSON API///////

//ListPosts lists all blog posts
func (b Blog) ListPosts(c *gin.Context) {
	c.JSON(http.StatusOK, b.GetPosts())
}

//GetPost returns a post with yyyy/mm/dd/slug
func (b Blog) GetPost(c *gin.Context) {
	post, err := b.GetPostObject(c)
	if err != nil {
		log.Println("Bad request in GetPost: " + err.Error())
		c.JSON(http.StatusBadRequest, err)
	}
	if post == nil {
		c.JSON(http.StatusNotFound, "Post Not Found")
	}
	c.JSON(http.StatusOK, post)
}

//////HTML API///////

//NoRoute returns a custom 404 page
func (b Blog) NoRoute(c *gin.Context) {

	tokens := strings.Split(c.Request.URL.String(), "/")
	// for some reason, first token is empty
	if len(tokens) >= 5 {
		year, _ := strconv.Atoi(tokens[1])
		month, _ := strconv.Atoi(tokens[2])
		day, _ := strconv.Atoi(tokens[3])
		post, err := b.getPostByParams(year, month, day, tokens[4])
		if err == nil && post != nil {
			if b.auth.IsAdmin(c) {
				c.HTML(http.StatusOK, "post-admin.html", gin.H{
					"logged_in": b.auth.IsLoggedIn(c),
					"is_admin":  b.auth.IsAdmin(c),
					"post":      post,
					"version":   b.Version,
				})
			} else {
				c.HTML(http.StatusOK, "post.html", gin.H{
					"logged_in": b.auth.IsLoggedIn(c),
					"is_admin":  b.auth.IsAdmin(c),
					"post":      post,
					"version":   b.Version,
				})
			}
			return
		}
	} else {
		log.Println("TOKEN LEN: " + strconv.Itoa(len(tokens)))
		for _, s := range tokens {
			log.Println(s)
		}
	}

	c.HTML(http.StatusNotFound, "error.html", gin.H{
		"error":       "404: Page Not Found",
		"description": "The page at '" + c.Request.URL.String() + "' was not found",
		"version":     b.Version,
	})
}

//Home returns html of the home page using the template
//if people want to have different stuff show on the home page they probably
//need to modify this function
func (b Blog) Home(c *gin.Context) {
	c.HTML(http.StatusOK, "home.html", gin.H{
		"logged_in": b.auth.IsLoggedIn(c),
		"is_admin":  b.auth.IsAdmin(c),
		"version":   b.Version,
		"title":     "Software Engineer",
	})
}

//Posts is the index page for blog posts
func (b Blog) Posts(c *gin.Context) {
	c.HTML(http.StatusOK, "posts.html", gin.H{
		"logged_in": b.auth.IsLoggedIn(c),
		"is_admin":  b.auth.IsAdmin(c),
		"posts":     b.GetPosts(),
		"version":   b.Version,
		"title":     "Posts",
	})
}

//Post is the page for all individual posts
func (b Blog) Post(c *gin.Context) {
	post, err := b.GetPostObject(c)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error":       "Post Not Found",
			"description": err.Error(),
			"version":     b.Version,
			"title":       "Post Not Found",
		})
	} else {
		c.HTML(http.StatusOK, "post.html", gin.H{
			"logged_in": b.auth.IsLoggedIn(c),
			"is_admin":  b.auth.IsAdmin(c),
			"post":      post,
			"version":   b.Version,
		})
		//if b.auth.IsAdmin(c) {
		//	c.HTML(http.StatusOK, "post-admin.html", gin.H{
		//		"logged_in": b.auth.IsLoggedIn(c),
		//		"is_admin":  b.auth.IsAdmin(c),
		//		"post":      post,
		//		"version":   b.version,
		//	})
		//} else {
		//	c.HTML(http.StatusOK, "post.html", gin.H{
		//		"logged_in": b.auth.IsLoggedIn(c),
		//		"is_admin":  b.auth.IsAdmin(c),
		//		"post":      post,
		//		"version":   b.version,
		//	})
		//}
	}
}

//Tag lists all posts with a given tag
func (b Blog) Tag(c *gin.Context) {
	tag := c.Param("name")
	posts, err := b.getPostsByTag(c)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error":       "Tag '" + tag + "' Not Found",
			"description": err.Error(),
			"version":     b.Version,
			"title":       "Tag '" + tag + "' Not Found",
		})
	} else {
		c.HTML(http.StatusOK, "tag.html", gin.H{
			"logged_in": b.auth.IsLoggedIn(c),
			"is_admin":  b.auth.IsAdmin(c),
			"posts":     posts,
			"tag":       tag,
			"version":   b.Version,
			"title":     "Posts with Tag '" + tag + "'",
		})
	}
}

//Tags is the index page for all Tags
func (b Blog) Tags(c *gin.Context) {
	c.HTML(http.StatusOK, "tags.html", gin.H{
		"version": b.Version,
		"title":   "Tags",
		"tags": b.getTags(),
	})
}

//Speaking is the index page for presentations
func (b Blog) Speaking(c *gin.Context) {
	c.HTML(http.StatusOK, "presentations.html", gin.H{
		"logged_in": b.auth.IsLoggedIn(c),
		"is_admin":  b.auth.IsAdmin(c),
		"version":   b.Version,
		"title":     "Speaking",
	})
}

//Projects is the index page for projects / code
func (b Blog) Projects(c *gin.Context) {
	c.HTML(http.StatusOK, "projects.html", gin.H{
		"logged_in": b.auth.IsLoggedIn(c),
		"is_admin":  b.auth.IsAdmin(c),
		"version":   b.Version,
		"title":     "Projects",
	})
}

//About is the about page
func (b Blog) About(c *gin.Context) {
	c.HTML(http.StatusOK, "about.html", gin.H{
		"logged_in": b.auth.IsLoggedIn(c),
		"is_admin":  b.auth.IsAdmin(c),
		"version":   b.Version,
		"title":     "About Jason",
	})
}

func (b Blog) Sitemap(c *gin.Context) {
	sm := stm.NewSitemap(1)
	sm.SetDefaultHost("https://www.jasonernst.com")
	sm.Create()
	sm.Add(stm.URL{{"loc", "/"}, {"changefreq", "weekly"}, {"priority", 1.0}})
	sm.Add(stm.URL{{"loc", "/posts"}, {"changefreq", "weekly"}, {"priority", 0.8}})
	sm.Add(stm.URL{{"loc", "/about"}, {"changefreq", "yearly"}, {"priority", 0.2}})

	posts := b.GetPosts()
	for _, post := range posts {
		sm.Add(stm.URL{{"loc", post.Permalink()}, {"changefreq", "yearly"}, {"priority", 0.55}})
	}
	tags := b.getTags()
	for _, tag := range tags {
		if len(tag.Posts) > 0 {
			sm.Add(stm.URL{{"loc", tag.Permalink()}, {"changefreq", "weekly"}, {"priority", 0.55}})
		}
	}

	c.Data(http.StatusOK, "text/xml", sm.XMLContent())
}

//Login to the blog
func (b Blog) Login(c *gin.Context) {
	err := godotenv.Load(".env")
	if err != nil {
		//fall back to local config
		err = godotenv.Load("local.env")
		if err != nil {
			//todo: handle better - perhaps return error to browser
			c.HTML(http.StatusInternalServerError, "Error loading .env file: "+err.Error(), gin.H{
				"logged_in": b.auth.IsLoggedIn(c),
				"is_admin":  b.auth.IsAdmin(c),
				"version":   b.Version,
				"title":     "Login Configuration Error",
			})
			return
		}
	}

	clientID := os.Getenv("client_id")
	c.HTML(http.StatusOK, "login.html", gin.H{
		"logged_in": b.auth.IsLoggedIn(c),
		"is_admin":  b.auth.IsAdmin(c),
		"client_id": clientID,
		"version":   b.Version,
		"title":     "Login",
	})
}

//Logout of the blog
func (b Blog) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("token")
	session.Save()
	c.Redirect(http.StatusTemporaryRedirect, "/")
}
