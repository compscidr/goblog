package main

import (
  "strings"
  "testing"
  "net/url"
  "net/http"
  "net/http/httptest"
  "github.com/stretchr/testify/assert"
  "github.com/bitly/go-simplejson"
)

// https://rshipp.com/go-api-integration-testing/
//https://semaphoreci.com/community/tutorials/test-driven-development-of-go-web-applications-with-gin
func setup() *App {
  // Initialize an in-memory database for full integration testing.
  app := &App{}
  app.Initialize("sqlite3", ":memory:")
  return app
}

func teardown(app *App) {
	// Closing the connection discards the in-memory database.
	app.DB.Close()
}

func TestCreatePost(t *testing.T) {
	app := setup()

  testTitle := "This is a test title"
  testContent := "This is some test content"
	data := url.Values{
    "Title": {testTitle},
    "Content": {testContent},
  }
  form := strings.NewReader(data.Encode())

  // Set up a new request.
	req, err := http.NewRequest("POST", "/post", form)
	if err != nil {
		t.Fatal(err)
	}
  // Our API expects a form body, so set the content-type header to make sure it's treated as one.
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
  w := httptest.NewRecorder()

  app.R.ServeHTTP(w, req)

  assert.Equal(t, 200, w.Code)

  //https://medium.com/@xoen/go-testing-technique-testing-json-http-requests-76d9ce0e11f
  reqJson, err := simplejson.NewFromReader(w.Body)
  if err != nil {
    t.Errorf("Error while reading request JSON: %s", err)
  }
  assert.Equal(t, testTitle, reqJson.GetPath("Title").MustString())
  assert.Equal(t, testContent, reqJson.GetPath("Content").MustString())

	teardown(app)
}

func TestCreateTag(t *testing.T) {
	app := setup()

  testTag := "This is a test tag"
	data := url.Values{
    "Name": {testTag},
  }
  form := strings.NewReader(data.Encode())

  // Set up a new request.
	req, err := http.NewRequest("POST", "/tag", form)
	if err != nil {
		t.Fatal(err)
	}
  // Our API expects a form body, so set the content-type header to make sure it's treated as one.
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
  w := httptest.NewRecorder()

  app.R.ServeHTTP(w, req)

  assert.Equal(t, 200, w.Code)

  //https://medium.com/@xoen/go-testing-technique-testing-json-http-requests-76d9ce0e11f
  reqJson, err := simplejson.NewFromReader(w.Body)
  if err != nil {
    t.Errorf("Error while reading request JSON: %s", err)
  }
  assert.Equal(t, testTag, reqJson.GetPath("Name").MustString())

	teardown(app)
}
