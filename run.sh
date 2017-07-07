#!/usr/bin/env bash

export GOPATH=`pwd`
export GOBIN=$GOPATH/bin

go get -d
go install
go run wallpaper-changer.go

