#### go实现的apns 提供https、redis协议方式的发送IOS Push服务
    单个连接的流控
    连接池的实现
    结合feedback过滤当前现有不合法的apnstoken提高送达率
    提供Https方式的发送ios push
    more.....
    
#### mock压测报告 
	4 * 2CPU /8GBRAM/50个apns链接
	http ab压测：
	ab c=10 n=1000000  8609.94 ops 
	ab c=50 n=1000000  12146.87 ops
	ab c=100 n=1000000 12952.80 op
============
#### 安装
    go get github.com/blackbeans/go-apns
    go install github/com/blackbeans/go-apns

quick start
============

#### 加载证书初始化apns
    cert, err := tls.LoadX509KeyPair(CERT_PATH, KEY_PATH)
    if nil != err {
    t.Logf("READ CERT FAIL|%s", err.Error())
		  t.Fail()
		  return
		}

	responseChan := make(chan entry.Response, 10)
	feedbackChan := make(chan entry.Feedback, 1000)

	body := "hello apns"
	payload := entry.NewSimplePayLoad("ms.caf", 1, body)
	client := NewDefaultApnsClient(cert, responseChan, PUSH_APPLE, feedbackChan, FEED_BACK_APPLE)
	
	
#####开始发送
	//发送enchanced push
	client.SendEnhancedNotification(1, math.MaxUint32, apnsToken, *payload)
	//调用读取feedback
	err := client.FetchFeedback()
	
##### 读取feedback和response结果

	//从channel 中读取数据
	select {
		case resp := <-responseChan:
			t.Logf("RESPONSE===============%t", resp)
			//如果有返回错误则说明发送失败的
		case fb := <-feedbackChan:
			i := 0
			for i < 100 {
				t.Logf("FEEDBACK===============%s", fb)
				i++
			}
	}
	
	

Http方式发送IOS PUSH
===================
####Server端使用：
    1. sh build.sh
    2. ./go-apns -startMode=1 -bindAddr=${IP:PORT} -certPath=${HOME}/cert.pem -keyPath=${HOME}/key.pem  -env=1 -pprof=:18070 -serverMode=moa -configPath=./conf/go_apns_moa.toml

    参数解释：
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

####MOA Client端发起调用
    go get git.wemomo.com/bibi/go-moa-client/client
    go get git.wemomo.com/bibi/go-moa/proxy
    go install  git.wemomo.com/bibi/go-moa-client/client
    go install  git.wemomo.com/bibi/go-moa/proxy

服务的URI为：
    /service/bibi/apns-service
客户端代码：
    
    package main
    import (
        "git.wemomo.com/bibi/go-moa-client/client"
        "git.wemomo.com/bibi/go-moa/proxy"
    )
    
    //apns发送的参数
    type ApnsParams struct {
        ExpSeconds int                    `json:"expiredSeconds"`
        Token      string                 `json:"token"`
        Sound      string                 `json:"sound"`
        Badge      int                    `json:"badge"`
        Body       string                 `json:"body"`
        ExtArgs    map[string]interface{} `json:"extArgs"`
    }
    
    type ApnsService struct {
        SendNotification func(pushType byte, params ApnsParams) (bool, error)
    }
    
    func main() {
        consumer := client.NewMoaConsumer("go_moa_client.toml",
            []proxy.Service{proxy.Service{
                ServiceUri: " /service/bibi/apns-service",
                Interface:  &ApnsService{}},
            })
    
        h := consumer.GetService("/service/bibi/apns-service").(*ApnsService)
        succ, err := h.SendNotification(1, ApnsParams{})
    
    }

注：go_moa_client.toml [文件参考](http://github.com/blackbeans/go-moa-client/blob/master/conf/moa_client.toml)




 


####HTTP Client端发起调用
    发送PUSH的POST协议：
    请求REQ：
        http://localhost:7070/apns/push
        pushType:= req.PostFormValue("pt") //notification 的类型
    
        pushType:
            NOTIFY_SIMPLE_FORMAT   = "0" //simple notification
            NOTIFY_ENHANCED_FORMAT = "1" //enhanced notification 
    
        token := req.PostFormValue("token") 
        sound := req.PostFormValue("sound")
        badgeV := req.PostFormValue("badge")
        body := req.PostFormValue("body")
        //是个大的Json数据即可
        extArgs := req.PostFormValue("extArgs")
        RESP：
        //---------定义返回状态码
        RESP_STATUS_SUCC                            = 200 //成功
        RESP_STATUS_ERROR                           = 500 //服务器端错误
        RESP_STATUS_INVALID_PROTO                   = 201 //不允许使用GET 请求发送数据
        RESP_STATUS_PUSH_ARGUMENTS_INVALID          = 400 //请求参数错误
        RESP_STATUS_INVALID_NOTIFY_FORMAT           = 501   //错误的NotificationFormat类型
        RESP_STATUS_PAYLOAD_BODY_DECODE_ERROR       = 505 //payload 的body   存在反序列化失败的问题
        RESP_STATUS_PAYLOAD_BODY_DEEP_ITERATOR      = 505 //payload 的body   不允许多层嵌套
        RESP_STATUS_SEND_OVER_TRY_ERROR             = 506 //推送到IOS PUSH     重试3次后失败
        RESP_STATUS_FETCH_FEEDBACK_OVER_LIMIT_ERROR = 507   //获取feedback的数量超过最大限制
        获取Feedback协议：
        GET ：
        REQ： http://localhost:7070/apns/feedback?limit=50
        RESP：返回指定数量的feedback 
            feedback: 
                time uint32
                devicetoken string
        NOTE :
            limit服务端最大每次可拉取 100条。








