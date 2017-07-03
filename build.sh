#!/usr/bin/env bash

export GOPATH=`pwd`
export GOBIN=$GOPATH/bin
export GOOS=windows
export GOARCH=amd64

go clean
go get -d
go install
go build -o $GOBIN/wallpaper-changer.exe wallpaper-changer.go

