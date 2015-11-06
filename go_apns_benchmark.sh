#!/bin/bash

#ab -c $1 -n $2 -p ./push.txt  -T application/x-www-form-urlencoded  https://bibi.wemomo.com/apns/push
ab -c $1 -n $2 -p ./push.txt  -T application/x-www-form-urlencoded  http://localhost:7070/apns/push
