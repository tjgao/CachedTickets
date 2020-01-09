#!/bin/sh

go get github.com/gorilla/mux
go get github.com/gorilla/websocket
go get github.com/sirupsen/logrus
go get github.com/lib/pq

GOOS=linux GOARCH=amd64 go build -o ticket_cache

# GOOS=windows GOARCH=amd64 go build -o 

# GOOS=darwin GOARCH=amd64 go build -o 

