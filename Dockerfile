# Build stage: the Go toolchain is only needed to compile. cgo is on
# (SQLite and Go plugins need it), so the runtime base below must be the
# same Debian release, for a matching glibc.
FROM golang:1.26.1-trixie AS build

ADD . /go/src/github.com/compscidr/goblog/

ARG VERSION
RUN cd /go/src/github.com/compscidr/goblog/ && go build -ldflags="-X 'main.Version=$VERSION'" -v .

FROM debian:trixie-slim

# ca-certificates: outgoing HTTPS (OAuth, the plugin directory, SMTP over TLS).
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Same path as before, so existing bind mounts still line up, and
# `docker exec <container> ./goblog reset-admin-password` finds .env.
WORKDIR /go/src/github.com/compscidr/goblog

# The binary plus what it reads from the working directory at runtime.
COPY --from=build /go/src/github.com/compscidr/goblog/goblog ./
COPY www ./www
COPY themes ./themes
COPY templates ./templates

ENTRYPOINT cd /go/src/github.com/compscidr/goblog && ./goblog

# Document that the service listens on port 7000.
EXPOSE 7000
