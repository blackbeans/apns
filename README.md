
go-apns is apple apns libary providing redis and http protocol to use 

####  feature:
    connection pool 
    [go-moa](https://github.com/blackbeans/go-apns) interface
    http protocol interface
    invalid token filted for reducing the rate of connection broken
    message resend while recieved fail information
    
#### mock benchmark：
	4 * 2CPU /8GBRAM/50 apns connections
	http ab benchmark：
	ab c=10 n=1000000  8609.94 ops 
	ab c=50 n=1000000  12146.87 ops
	ab c=100 n=1000000 12952.80 op
============
#### install
    sh build.sh

quick start
============

#### load and initial apns server
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
	
	
##### send
	// send enchanced push
	client.SendEnhancedNotification(1, math.MaxUint32, apnsToken, *payload)
	//read feedback
	err := client.FetchFeedback()
	
##### read feedback and response 

	//recieve response from channel 
	select {
		case resp := <-responseChan:
			t.Logf("RESPONSE===============%t", resp)
		case fb := <-feedbackChan:
			i := 0
			for i < 100 {
				t.Logf("FEEDBACK===============%s", fb)
				i++
			}
	}
	
	

Use MOA Protocol  
===================
####Server：
    1. sh build.sh
    2. ./go-apns -startMode=1 -bindAddr=${IP:PORT} -certPath=${HOME}/cert.pem -keyPath=${HOME}/key.pem  -env=1 -pprof=:18070 -serverMode=moa -configPath=./conf/go_apns_moa.toml

    参数解释：
        startMode := flag.Int("startMode", 1, " 0 mock ,1 online")
        bindAddr := flag.String("bindAddr", "", "-bindAddr=:17070")
        certPath := flag.String("certPath", "./cert.pem", "-certPath=xxxxxx/cert.pem or -certPath=http://")
        keyPath := flag.String("keyPath", "./key.pem", "-keyPath=xxxxxx/key.pem or -keyPath=http://")
        env := flag.Int("env", 0, "-env=1(online) ,0(sandbox)")
        storeCap := flag.Int("storeCap", 1000, "-storeCap=100000  //resender queue length")
        logxml := flag.String("log", "./conf/log.xml", "-log=./conf/log.xml //log config")
        pprofPort := flag.String("pprof", ":9090", "-pprof=:9090 //端口")
        configPath := flag.String("configPath", "", "-configPath=conf/go_apns_moa.toml //moa config file")
        serverMode := flag.String("serverMode", "http", "-serverMode=http/moa // using http/moa server ")
        tokenStorage := flag.String("tokenStorage", "",
        "redis://addrs=localhost:6379,localhost:6379&expiredSec=86400 //invalid token storage")
        flag.Parse()

####MOA Client

    go get github.com/blackbeans/go-moa-client/client
    go get github.com/blackbeans/go-moa/proxy
    go install  github.com/blackbeans/go-moa-client/client
    go install  github.com/blackbeans/go-moa/proxy

service uri ：
    /service/apns-service
 
client ：
    
    package main
    import (
        "github.com/blackbeans/go-moa-client/client"
        "github.com/blackbeans/go-moa/proxy"
    )
    
    //apns params
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

note：go_moa_client.toml [ref](http://github.com/blackbeans/go-moa-client/blob/master/conf/moa_client.toml)




 


#### Use HTTP Protocol
    send push by post 
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
        //extral Json data
        extArgs := req.PostFormValue("extArgs")
        RESP：
        //---------status code
        RESP_STATUS_SUCC                            = 200 //succ
        RESP_STATUS_ERROR                           = 500 //fail
        RESP_STATUS_INVALID_PROTO                   = 201 //GET not supported
        RESP_STATUS_PUSH_ARGUMENTS_INVALID          = 400 //invalid request params 
        RESP_STATUS_INVALID_NOTIFY_FORMAT           = 501 //invalid NotificationFormat type
        RESP_STATUS_PAYLOAD_BODY_DECODE_ERROR       = 505 //payload deserilaized fail 
        RESP_STATUS_PAYLOAD_BODY_DEEP_ITERATOR      = 505 //payload body is complex
        RESP_STATUS_SEND_OVER_TRY_ERROR             = 506 //send fail over try
        RESP_STATUS_FETCH_FEEDBACK_OVER_LIMIT_ERROR = 507 //over limit
        Feedback：
        GET ：
        REQ： http://localhost:7070/apns/feedback?limit=50
        RESP：feedback 
            feedback: 
                time uint32
                devicetoken string
        NOTE :
            limit is less than 100


#### Donate

![image](github.com/blackbeans/kiteq/blob/master/doc/qcode.png)







