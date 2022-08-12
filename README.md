# goblog
![Go](https://github.com/compscidr/goblog/workflows/Go/badge.svg)
[![codecov](https://codecov.io/gh/compscidr/goblog/branch/master/graph/badge.svg)](https://codecov.io/gh/compscidr/goblog)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Simple blogging platform built with golang. Currently running on my website: https://www.jasonernst.com

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
I'll likely create an initial install onboarding which makes the first logged
in user the admin.

Other logged in users are able to post, update and delete comments (todo)

Every function in the API should be covered by units and integration tests.

What works:
- Install Wizard for onboarding
- List, Create, Update, Delete
- Local sqlite3 db in a file
- Markdown rendering, code highlighting for content
- Posts, Tags, Error Pages, File Uploads (images, pdfs, etc)
- Github action which builds and deploys a tagged dockerhub release when a versioned release is cut
- Version string in template header from `git describe`
- Meta, Title, etc which changes with posts for SEO
- Post date editing so old posts can be imported

Todo:
- draft posts
- default hero images or something so posts don't look so bare
- cron job to backup posts
- user comments
- mysql [WiP], postgres, other dbs

## Other tools used:
- Gin: https://github.com/gin-gonic/gin. Used for multiplexing / routing the
http requests.
- Gorm: https://github.com/jinzhu/gorm. Used for object relational mapping.

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
