#!/bin/bash
go get  github.com/blackbeans/log4go
go get 	gopkg.in/redis.v3
go get  github.com/go-errors/errors
go get  github.com/blackbeans/go-moa/core
go get  github.com/blackbeans/go-moa/proxy

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

tar -zcvf go-apns.tar.gz $PROJ conf/*
