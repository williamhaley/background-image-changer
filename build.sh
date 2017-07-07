#!/usr/bin/env bash

export GOPATH=`pwd`
export GOBIN=$GOPATH/bin
export GOOS=windows
# export GOARCH=arm64
export GOARCH=386

go clean
go get -d
go install
go build -o $GOBIN/background-image-changer.exe background-image-changer.go

