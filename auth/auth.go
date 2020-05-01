package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

//Auth API
type Auth struct {
}

//AccessTokenResponse comes from Github OAuth API when the user has successfully
//authenticated - note the github api provides more fields but we can just leave
//them out and it will parse just fine - cool!
type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

//GithubUser specifies the fields we will use to map a github identity to users
//in the blog. The only really important one is the admin, otherwise they're
//just used for comments at the moment.
type GithubUser struct {
	ID          string `gorm:"primary_key"`
	Login       string `json:"login"`
	AvatarURL   string `json:"avatar_url"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	AccessToken string `json:"access_token"`
}

//use the Github app credentials + the code we received from javascript
//client side to make the access token (bearer) request
func (a Auth) requestAccessToken(parsedCode string) (*AccessTokenResponse, error) {
	data := &AccessTokenResponse{}

	//todo: move these out of the code and into environment variables
	formData := url.Values{
		"client_id":     {"9a4892a7a4a8c4646225"},
		"client_secret": {"cfe1fa8128e81caf81b21645c0784231751b3627"},
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
func (a Auth) requestUser(accessToken string) (*GithubUser, error) {
	data := &GithubUser{}
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
	fmt.Println("post:\n", bodyString) //todo: remove - just for debugging

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(bodyString)
	}

	json.Unmarshal(bodyBytes, &data)
	data.AccessToken = accessToken
	return data, nil
}

//LoginHandler handles login requests
//user provides code from the initial call to github
//we use the code to obtain a bearer token which is passed back to the user
//and used by us to obtain the user email. We then map the email to the bearer
//token. When the user logs out, we destroy the bearer token in the database
//so when they try to use it as an API access token it isn't found and they
//have to login again. The bearer token is used for all authorized API calls.
func (a Auth) LoginHandler(w http.ResponseWriter, r *http.Request) {
	parsedCode := ""
	if r.Method == "GET" {
		code, ok := r.URL.Query()["code"]
		if !ok || len(code[0]) < 1 {
			log.Println("API parameters 'code' is missing")
			http.Error(w, "API parameters 'code' is missing", http.StatusBadRequest)
			return
		}
		parsedCode = code[0]
	} else if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Error parsing POST request: "+err.Error(), http.StatusBadRequest)
			return
		}
		parsedCode = r.FormValue("code")
	} else {
		log.Println("http method not supported: " + r.Method)
		http.Error(w, "http method not supported: "+r.Method, http.StatusBadRequest)
		return
	}

	data, err := a.requestAccessToken(parsedCode)
	if err != nil {
		http.Error(w, "Error requesting token access: "+err.Error(), http.StatusUnauthorized)
		return
	}

	user, err := a.requestUser(data.AccessToken)

	js, err := json.Marshal(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//everything went well, send back the bearer token + user info for the website
	//to use if it wants
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	w.Write(js)
}
