#!/bin/bash

#https://blog.codeship.com/building-minimal-docker-containers-for-go-applications/

VERSION=11

CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o hookyd ../cmd/hookyd/main.go
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o hooky ../cmd/hooky/main.go

docker build -t sebest/hooky:$VERSION .
docker build -t sebest/hooky:latest .
docker push sebest/hooky:$VERSION
docker push sebest/hooky:latest
