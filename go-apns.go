package main

import (
	"encoding/json"
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
	env := flag.Int("env", 0, "-env=1(online) ,0(sandbox)")
	storeCap := flag.Int("storeCap", 1000, "-storeCap=100000  //重发链条长度")
	logxml := flag.String("log", "./conf/log.xml", "-log=./conf/log.xml //log配置文件")
	pprofPort := flag.String("pprof", ":9090", "-pprof=:9090 //端口")
	configPath := flag.String("configPath", "", "-configPath=conf/go_apns_moa.toml //moa启动的配置文件")
	serverMode := flag.String("serverMode", "http", "-serverMode=http/moa //http或者moa方式启动")
	flag.Parse()

	//设置启动项
	option := server.NewOption(*startMode, *bindAddr, *certPath, *keyPath, *env, *storeCap)
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

	//------------启动pprof
	go func() {
		if len(*pprofPort) > 0 {
			addr, _ := net.ResolveTCPAddr("tcp4", *bindAddr)
			http.HandleFunc("/apns/stat", func(out http.ResponseWriter, req *http.Request) {
				//获取状态
				status := apnsClient.Monitor()
				jsonData, _ := json.Marshal(status)
				out.Header().Set("content-type", "text/json")
				out.Write(jsonData)
			})
			log.Error(http.ListenAndServe(addr.IP.String()+*pprofPort, nil))
		}
	}()

	var apnsserver *h.ApnsHttpServer
	var app *moa.Bootstrap
	if nil != serverMode && "http" == *serverMode {
		//启动http形式的
		apnsserver = h.NewApnsHttpServer(*logxml, option, feedbackChan, apnsClient)
	} else if nil != serverMode && "moa" == *serverMode {
		app = moa.NewBootstrap(*configPath, option, feedbackChan, apnsClient)
	} else {
		panic("UnSupport ServerMode [" + *serverMode + "]")
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
