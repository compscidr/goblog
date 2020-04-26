package main

//this implements https://jsonapi.org/format/ as best as possible

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/rs/cors"
)

//user provides code from the initial call to github
//we use the code to obtain a bearer token which is passed back to the user
//and used by us to obtain the user email. We then map the email to the bearer
//token. When the user logs out, we destroy the bearer token in the database
//so when they try to use it as an API access token it isn't found and they
//have to login again. The bearer token is used for all authorized API calls.
func loginHandler(w http.ResponseWriter, r *http.Request) {
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

	//use the Github app credentials + the code we received from javascript
	//client side to make the access token (bearer) request
	formData := url.Values{
		"client_id":     {"9a4892a7a4a8c4646225"},
		"client_secret": {"cfe1fa8128e81caf81b21645c0784231751b3627"},
		"code":          {parsedCode},
	}
	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(formData.Encode()))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Accept", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		panic(err)
	}

	bodyString := ""
	if resp.StatusCode == http.StatusOK {
		//log.Println(resp)
		defer resp.Body.Close()
		bodyBytes, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			panic(err)
		}
		bodyString = string(bodyBytes)
		fmt.Println("post:\n", bodyString)
	}

	//everything went well, send back the bearer token + user info for the website
	//to use if it wants
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(bodyString); err != nil {
		panic(err)
	}
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/login", loginHandler)

	// todo: restrict cors properly to same domain: https://github.com/rs/cors
	// this lets us get a request from localhost:8000 without the web browser
	// bitching about it
	cors := cors.Default().Handler(mux)
	http.ListenAndServe(":7000", cors)
}
