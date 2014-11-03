#### go实现的apns 提供https、redis协议方式的发送IOS Push服务
    单个连接的流控
    连接池的实现
    结合feedback过滤当前现有不合法的apnstoken提高送达率
    提供Https方式的发送ios push
    more.....

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
    Server端使用：

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

    测试启动：
    go run demo.go  -certPath=/Users/blackbeans/pushcert.pem -keyPath=/Users/blackbeans/key.pem -bindAddr=:17070 -runMode=1


Client端发起调用
发送PUSH协议：
POST：
    REQ：
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
    RESP_STATUS_INVALID_NOTIFY_FORMAT           = 501 //错误的NotificationFormat类型
    RESP_STATUS_PAYLOAD_BODY_DECODE_ERROR       = 505 //payload 的body 存在反序列化失败的问题
    RESP_STATUS_PAYLOAD_BODY_DEEP_ITERATOR      = 505 //payload 的body 不允许多层嵌套
    RESP_STATUS_SEND_OVER_TRY_ERROR             = 506 //推送到IOS PUSH 重试3次后失败
    RESP_STATUS_FETCH_FEEDBACK_OVER_LIMIT_ERROR = 507 //获取feedback的数量超过最大限制
    获取Feedback协议：
    GET ：
    REQ： http://localhost:7070/apns/feedback?limit=50
    RESP：返回指定数量的feedback 
        feedback: 
            time uint32
            devicetoken string
    NOTE :
        limit服务端最大每次可拉取 100条。








