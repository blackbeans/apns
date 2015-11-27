#!/bin/bash

go get -u github.com/blackbeans/log4go

go build go-apns/entry
go install go-apns/entry
go build go-apns/apns
go install  go-apns/apns
go build go-apns/server
go install go-apns/server

echo "------------ compoments  installing is finished!-------------"

PROJ=`pwd | awk -F'/' '{print $(NF)}'`
#VERSION=$1
#go build  -o ./$PROJ-$VERSION $PROJ.go
go build  -o ./$PROJ $PROJ.go

tar -zcvf go-apns.tar.gz $PROJ log.xml
