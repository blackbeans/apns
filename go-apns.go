package main

import (
	"flag"
	log "github.com/blackbeans/log4go"
	"go-apns/server"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
)

func main() {

	runtime.GOMAXPROCS(8)
	startMode := flag.Int("startMode", 1, " 0 为mock ,1 为正式")
	bindAddr := flag.String("bindAddr", ":17070", "-bindAddr=:17070")
	certPath := flag.String("certPath", "./cert.pem", "-certPath=xxxxxx/cert.pem or -certPath=http://")
	keyPath := flag.String("keyPath", "./key.pem", "-keyPath=xxxxxx/key.pem or -keyPath=http://")
	runMode := flag.Int("runMode", 0, "-runMode=1(online) ,0(sandbox)")
	storeCap := flag.Int("storeCap", 1000, "-storeCap=100000  //重发链条长度")
	logxml := flag.String("log", "log.xml", "-log=log.xml //log配置文件")
	pprofPort := flag.String("pprof", ":9090", "pprof=:9090 //端口")
	flag.Parse()

	go func() {
		if len(*pprofPort) > 0 {
			addr, _ := net.ResolveTCPAddr("tcp4", *bindAddr)
			log.Error(http.ListenAndServe(addr.IP.String()+*pprofPort, nil))
		}
	}()

	//加载log4go的配置
	log.LoadConfiguration(*logxml)

	//设置启动项
	option := server.NewOption(*startMode, *bindAddr, *certPath, *keyPath, *runMode, *storeCap)
	apnsserver := server.NewApnsHttpServer(option)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Kill)
	//kill掉的server
	<-ch
	apnsserver.Shutdown()

	log.Info("APNS SERVER IS STOPPED!")

}
