package main

import (
	"flag"
	"go-apns/server"
	"log"
	"os"
	"os/signal"
)

func main() {
	bindAddr := flag.String("bindAddr", ":17070", "-bindAddr=:17070")
	certPath := flag.String("certPath", "./cert.pem", "-certPath=xxxxxx/cert.pem or -certPath=http://")
	keyPath := flag.String("keyPath", "./key.pem", "-keyPath=xxxxxx/key.pem or -keyPath=http://")
	runMode := flag.Int("runMode", 0, "-runMode=1(online) ,0(sandbox)")
	storeCap := flag.Int("storeCap", 0, "-storeCap=100000  //重发链条长度")
	flag.Parse()

	//设置启动项
	option := server.NewOption(*bindAddr, *certPath, *keyPath, *runMode, *storeCap)
	apnsserver := server.NewApnsHttpServer(option)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Kill)
	//kill掉的server
	<-ch
	apnsserver.Shutdown()
	log.Println("APNS SERVER IS STOPPED!")

}
