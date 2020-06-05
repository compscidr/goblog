# goblog
![Go](https://github.com/compscidr/goblog/workflows/Go/badge.svg)

Simple blogging platform built with golang.

It is split into two parts:
- A JSON REST API located at /api/v1/
- An HTML API located at /

The HTML API is optional - any frontend could be used instead. I toyed with a
single page Javascript app for a bit, but I like to have all of the content
generated on the server side so that SEO works better.

Users can log into the blog using Github authentication code which is then
translated by the blog API into an authorization token, which is then stored
in a session cookie.

Creating, modifying and deleting posts and administering comments may only be
done with the admin user. Presently the user is hardcoded by github email, but
I'll likely create an initial install onboarding which makes the first logged in
user the admin.

Other logged in users are able to post, update and delete comments.

Every function in the API should be covered by units and integration tests.

What works:
- List
- Create
- Update
- Delete
- Local sqlite3 db in a file
- Markdown rendering for content

Todo:
- Increase test coverage (and coverage reporting on the readme would be nice)
- Error pages
- Tags
- User Comments
- Install onboarding
- mysql, postgres, other dbs

## Other tools used:
- Gin: https://github.com/gin-gonic/gin. Used for multiplexing / routing the
http requests.
- Gorm: https://github.com/jinzhu/gorm. Used for object relational mapping.

## Other things to consider, take inspiration from
- Go project structure: https://github.com/golang-standards/project-layout

- Buffalo: https://github.com/gobuffalo/buffalo

- Go-Web-Api: https://rshipp.com/go-web-api/
- Go-Web-Api Integration Testing: https://rshipp.com/go-api-integration-testing/

- Gin-Web-Api: https://semaphoreci.com/community/tutorials/building-go-web-applications-and-microservices-using-gin
- GIn-Web-Api Test-Driven Dev: https://semaphoreci.com/community/tutorials/test-driven-development-of-go-web-applications-with-gin

- Golang session auth: https://www.sohamkamani.com/blog/2018/03/25/golang-session-authentication/
- Auth example: https://gist.github.com/dtan4/a3b5027dd3c7d5c5ed3119ea97fb7235

## Building and running:
```
go build
goblog
```

## Testing
```
go test goblog/...
```

## Coverage:
More details here: https://blog.golang.org/cover
```
go test goblog/... -coverprofile=coverage.out
go tool cover -func=coverage.out
go tool cover -html=coverage.out
```
