#!/bin/bash
go build go-apns/entry
go install go-apns/entry
go build go-apns/apns
go install  go-apns/apns
go build go-apns/server
go install go-apns/server

echo "------------ compoments  installing is finished!-------------"

PROJ=`pwd | awk -F'/' '{print $(NF)}'`
VERSION=$1
go build  -o ./$PROJ-$VERSION $PROJ.go
