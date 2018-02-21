FROM golang:1.7-alpine

# Go and Glide require GIT
RUN apk update && apk add --no-cache git

# Install Glide so we can get dependenciesß
RUN go get github.com/Masterminds/glide
RUN go build github.com/Masterminds/glide