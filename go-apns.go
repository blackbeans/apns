package main

import (
	"flag"
	log "github.com/blackbeans/log4go"
	"go-apns/apns"
	"go-apns/entry"
	"go-apns/server"
	h "go-apns/server/http"
	"go-apns/server/moa"
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
	bindAddr := flag.String("bindAddr", "", "-bindAddr=:17070")
	certPath := flag.String("certPath", "./cert.pem", "-certPath=xxxxxx/cert.pem or -certPath=http://")
	keyPath := flag.String("keyPath", "./key.pem", "-keyPath=xxxxxx/key.pem or -keyPath=http://")
	runMode := flag.Int("runMode", 0, "-runMode=1(online) ,0(sandbox)")
	storeCap := flag.Int("storeCap", 1000, "-storeCap=100000  //重发链条长度")
	logxml := flag.String("log", "log.xml", "-log=log.xml //log配置文件")
	pprofPort := flag.String("pprof", ":9090", "-pprof=:9090 //端口")
	configPath := flag.String("configPath", "", "-configPath=~/cluster_moa.toml //moa启动的配置文件")
	flag.Parse()

	go func() {
		if len(*pprofPort) > 0 {
			addr, _ := net.ResolveTCPAddr("tcp4", *bindAddr)
			log.Error(http.ListenAndServe(addr.IP.String()+*pprofPort, nil))
		}
	}()

	//设置启动项
	option := server.NewOption(*startMode, *bindAddr, *certPath, *keyPath, *runMode, *storeCap)
	feedbackChan := make(chan *entry.Feedback, 1000)
	var apnsClient *apns.ApnsClient
	if option.StartMode == server.STARTMODE_MOCK {
		//初始化mock apns
		apnsClient = apns.NewMockApnsClient(option.Cert,
			option.PushAddr, chan<- *entry.Feedback(feedbackChan),
			option.FeedbackAddr, entry.NewCycleLink(3, option.StorageCapacity))
		log.Info("MOCK APNS HTTPSERVER IS STARTING ....")
	} else {
		//初始化apns
		apnsClient = apns.NewDefaultApnsClient(option.Cert,
			option.PushAddr, chan<- *entry.Feedback(feedbackChan),
			option.FeedbackAddr, entry.NewCycleLink(3, option.StorageCapacity))
		log.InfoLog("push_handler", "ONLINE APNS HTTPSERVER IS STARTING ....")
	}
	var apnsserver *h.ApnsHttpServer
	if nil != bindAddr && len(*bindAddr) > 0 {
		//启动http形式的
		apnsserver = h.NewApnsHttpServer(*logxml, option, feedbackChan, apnsClient)
	}
	var app *moa.Bootstrap
	if nil != configPath && len(*configPath) > 0 {
		app = moa.NewBootstrap(*configPath, option, feedbackChan, apnsClient)
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Kill)
	//kill掉的server
	<-ch
	if nil != apnsserver {
		apnsserver.Shutdown()
	}
	if nil != app {
		app.Destory()
	}

	log.Info("APNS SERVER IS STOPPED!")

}
