package core

import (
	"fmt"
	"github.com/blackbeans/go-moa/lb"
	"github.com/blackbeans/go-moa/log4moa"
	"github.com/blackbeans/go-moa/protocol"
	"github.com/blackbeans/go-moa/proxy"
	log "github.com/blackbeans/log4go"
	"github.com/blackbeans/turbo"
	"github.com/blackbeans/turbo/client"
	"github.com/blackbeans/turbo/codec"
	"github.com/blackbeans/turbo/packet"
	"github.com/blackbeans/turbo/server"
	"net"
	"net/http"
	_ "net/http/pprof"
)

type ServiceBundle func() []proxy.Service

type Application struct {
	remoting      *server.RemotingServer
	invokeHandler *proxy.InvocationHandler
	options       *MOAOption
	configCenter  *lb.ConfigCenter
	moaStat       *log4moa.MoaStat
}

func NewApplcation(configPath string, bundle ServiceBundle) *Application {
	return NewApplicationWithAlarm(configPath, bundle,
		func(serviceUri, hostname string, moaInfo log4moa.MoaInfo) {
			//do nothing
		})
}

//with alarm
func NewApplicationWithAlarm(configPath string, bundle ServiceBundle,
	monitor func(serviceUri, host string, moainfo log4moa.MoaInfo)) *Application {
	services := bundle()

	options, err := LoadConfiruation(configPath)
	if nil != err {
		panic(err)
	}

	//修正serviceUri的后缀
	for i, s := range services {
		s.ServiceUri = (s.ServiceUri + options.serviceUriSuffix)
		services[i] = s
	}

	name := options.name + "/" + options.hostport
	rc := turbo.NewRemotingConfig(name,
		options.maxDispatcherSize,
		options.readBufferSize,
		options.readBufferSize,
		options.writeChannelSize,
		options.readChannelSize,
		options.idleDuration,
		50*10000)

	//需要开发对应的codec
	cf := func() codec.ICodec {
		return protocol.RedisGetCodec{32 * 1024}
	}

	//创建注册服务
	configCenter := lb.NewConfigCenter(options.registryType,
		options.registryHosts, options.hostport, services)

	app := &Application{}
	app.options = options
	app.configCenter = configCenter
	//moastat
	moaStat := log4moa.NewMoaStat(options.hostport, services[0].ServiceUri, monitor, func() turbo.NetworkStat {
		return app.remoting.NetworkStat()

	})
	app.moaStat = moaStat
	app.invokeHandler = proxy.NewInvocationHandler(services, moaStat)

	//启动remoting
	remoting := server.NewRemotionServerWithCodec(options.hostport, rc, cf,
		func(remoteClient *client.RemotingClient, p *packet.Packet) {
			packetDispatcher(app, remoteClient, p)
		})
	app.remoting = remoting
	remoting.ListenAndServer()
	moaStat.StartLog()

	//------------启动pprof
	go func() {
		hp, _ := net.ResolveTCPAddr("tcp4", options.hostport)
		pprof := fmt.Sprintf("%s:%d", hp.IP, (hp.Port + 1000))
		log.ErrorLog("moa-server", http.ListenAndServe(pprof, nil))

	}()

	//注册服务
	configCenter.RegisteAllServices()
	log.InfoLog("moa-server", "Application|Start|SUCC|%s|%s", name, options.hostport)
	return app
}

func (self Application) DestroyApplication() {

	//取消注册服务
	self.configCenter.Destroy()
	//关闭remoting
	self.remoting.Shutdown()
}

//需要开发对应的分包
func packetDispatcher(self *Application, remoteClient *client.RemotingClient, p *packet.Packet) {

	defer func() {
		if err := recover(); nil != err {
			log.ErrorLog("moa-server", "Application|packetDispatcher|FAIL|%s", err)
		}
	}()

	//如果是get命令
	if p.Header.CmdType == protocol.GET {
		//这里面根据解析包的内容得到调用不同的service获得结果
		req, err := protocol.Wrap2MoaRawRequest(p.Data)
		if nil != err {
			log.ErrorLog("moa-server", "Application|packetDispatcher|Wrap2MoaRequest|FAIL|%s|%s", err, string(p.Data))
		} else {
			req.Source = remoteClient.RemoteAddr()
			req.Channel = remoteClient.AttachChannel
			req.Timeout = self.options.processTimeout
			result := self.invokeHandler.Invoke(req)
			resp, err := protocol.Wrap2ResponsePacket(p, result)
			if nil != err {
				log.ErrorLog("moa-server", "Application|packetDispatcher|Wrap2ResponsePacket|FAIL|%s|%s", err, result)
			} else {
				remoteClient.Write(*resp)
				//log.DebugLog("moa-server", "Application|packetDispatcher|SUCC|%s", *resp)
			}
		}

	} else if p.Header.CmdType == protocol.PING {
		//PING 协议 不做人事事情
		resp, _ := protocol.Wrap2ResponsePacket(p, "PONG")
		remoteClient.Write(*resp)
	} else if p.Header.CmdType == protocol.INFO {
		//INFO 协议，返回服务端信息
		stat := make(map[string]interface{}, 2)
		stat["network"] = self.remoting.NetworkStat()
		stat["moa"] = self.moaStat.GetMoaInfo()
		resp, _ := protocol.Wrap2ResponsePacket(p, stat)
		remoteClient.Write(*resp)
	}

}
