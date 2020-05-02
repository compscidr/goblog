# goblog
![Go](https://github.com/compscidr/goblog/workflows/Go/badge.svg)

Simple blogging platform built with golang.

The basic idea is that there is some type of front-end (likely in a separate
repository) which makes the UI that someone actually uses in the web browser.

This repository will simply have an api located at the website/api/v<version.
The api is split into several parts, the first requires no user authentication.
It allows viewing and searching of public content like posts, comments, etc.

The second part of the api is user auth for things like posting comments. It
does not allow for making, editing or deleting posts.

For the two authenticated sections of the api, the login will use github oAuth.
The token provided by github oauth will be used for API access to the other
parts.

https://www.sohamkamani.com/blog/2018/03/25/golang-session-authentication/
https://gist.github.com/dtan4/a3b5027dd3c7d5c5ed3119ea97fb7235

Every function in the api should be covered by units and integration tests.

What works:
- List
- Create
- Delete
- Local sqlite3 db in a file

Todo:
- Update
- Templates
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

## Building and running:
```
go build ./cmd/goblog.go
goblog
```

## Testing
```
go test
```

## Coverage:
More details here: https://blog.golang.org/cover
```
go test -coverprofile=coverage.out
go tool cover -func=coverage.out
go tool cover -html=coverage.out
```
