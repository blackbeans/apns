package example

import (
	"flag"
	"go-apns/server"
	"os"
	"os/signal"
)

func main() {
	bindAddr := flag.String("bindAddr", ":17070", "-bindAddr=:17070")
	certPath := flag.String("certPath", "", "-certPath=/User/xxx")
	keyPath := flag.String("keyPath", "", "-keyPath=/User/xxx")
	runMode := flag.Int("runMode", 0, "-runMode=1(online) ,0(sandbox)")
	flag.Parse()

	//设置启动项
	option := server.NewOption(*bindAddr, *certPath, *keyPath, *runMode)
	apnsserver := server.NewApnsHttpServer(option)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Kill)
	//kill掉的server
	<-ch
	apnsserver.Shutdown()

}
