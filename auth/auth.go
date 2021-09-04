package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

//IAuth interface for auth so that it can be mocked easier
type IAuth interface {
	IsAdmin(c *gin.Context) bool
	IsLoggedIn(c *gin.Context) bool
}

//Auth API
type Auth struct {
	db      *gorm.DB
	version string
}

//New constructs an Auth API
func New(db *gorm.DB, version string) Auth {
	api := Auth{db, version}
	return api
}

//AccessTokenResponse comes from Github OAuth API when the user has successfully
//authenticated - note the github api provides more fields but we can just leave
//them out and it will parse just fine - cool!
type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

//use the Github app credentials + the code we received from javascript
//client side to make the access token (bearer) request
func (a Auth) requestAccessToken(parsedCode string) (*AccessTokenResponse, error) {
	err := godotenv.Load(".env")
	if err != nil {
		return nil, errors.New("Error loading .env file: " + err.Error())
	}
	clientID := os.Getenv("client_id")
	clientSecret := os.Getenv("client_secret")

	data := &AccessTokenResponse{}

	//todo: move these out of the code and into environment variables
	formData := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {parsedCode},
	}
	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	bodyString := string(bodyBytes)
	fmt.Println("post:\n", bodyString) //todo: remove - just for debugging

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(bodyString)
	}

	json.Unmarshal(bodyBytes, &data)
	return data, nil
}

// formatRequest generates ascii representation of a request
func (a Auth) formatRequest(r *http.Request) string {
	// Create return string
	var request []string // Add the request string
	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)                             // Add the host
	request = append(request, fmt.Sprintf("Host: %v", r.Host)) // Loop through headers
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}

	// If this is a POST, add post data
	if r.Method == "POST" {
		r.ParseForm()
		request = append(request, "\n")
		request = append(request, r.Form.Encode())
	} // Return the request as a string
	return strings.Join(request, "\n")
}

//todo: make these request functions generalized
func (a Auth) requestUser(accessToken string) (*BlogUser, error) {
	data := &BlogUser{}
	//get the user info from Github
	req, err := http.NewRequest("GET", "https://api.github.com/user", strings.NewReader(""))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "token "+accessToken)
	fmt.Println("REQ" + a.formatRequest(req))

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	bodyString := string(bodyBytes)
	fmt.Println("github response:\n", bodyString) //todo: remove - just for debugging

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(bodyString)
	}

	json.Unmarshal(bodyBytes, &data)
	data.AccessToken = accessToken

	fmt.Println("Parsed user: ", data)

	return data, nil
}

//LoginPostHandler should be called with the code provided by github. After
//receiving the code, this will reach out to github to retrieve and auth token
//which is stored in the db along with the user information from github.
//this can then be used for authoization when the api user supplies the same
//auth token later on for API access. Only one auth token per user can be used
//at once. Logout should remove the auth token from the table.
func (a Auth) LoginPostHandler(c *gin.Context) {
	parsedCode := c.PostForm("code")
	data, err := a.requestAccessToken(parsedCode)
	if err != nil {
		c.JSON(http.StatusUnauthorized, "Error requesting token access: "+err.Error())
		return
	}

	user, err := a.requestUser(data.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}

	//check if user exists, if not add them, if they do update access token
	var existingUser BlogUser
	err = a.db.Where("github_id = ?", user.GithubID).First(&existingUser).Error
	if err != nil {
		a.db.Create(&user)
	} else {
		a.db.Model(&user).Where("github_id = ?", user.GithubID).Updates(&user)
		existingUser = *user
	}

	//save the access token in the session
	session := sessions.Default(c)
	session.Set("token", data.AccessToken)
	session.Save()

	c.JSON(http.StatusOK, existingUser)
}

//DisplayUserTable is a debug function, shows the user table
func (a Auth) DisplayUserTable() {
	var users []BlogUser
	a.db.Find(&users)
	log.Println(users)
}

//IsAdmin returns true if the user logged in is the admin user
//First tries for a session token, and if that fails falls back on an auth token
func (a Auth) IsAdmin(c *gin.Context) bool {
	session := sessions.Default(c)
	token := session.Get("token")
	if token == nil {
		token = c.Request.Header.Get("Authorization")
	}

	//debug
	a.DisplayUserTable()

	//todo un-hardcode the admin lol.
	var existingUser BlogUser
	err := a.db.Where("access_token = ? AND email = ?", token, "ernstjason1@gmail.com").First(&existingUser).Error
	if err != nil {
		return false
	}
	log.Println("TOKEN:", token)
	log.Println("ADMIN USER:", existingUser)
	return true
}

//IsLoggedIn Returns true if the user is logged in, false otherwise
func (a Auth) IsLoggedIn(c *gin.Context) bool {
	session := sessions.Default(c)
	token := session.Get("token")
	if token == nil {
		return false
	}
	var existingUser BlogUser
	err := a.db.Where("access_token = ?", token).First(&existingUser).Error
	if err != nil {
		return false
	}
	return true
}
