package server

import (
	"crypto/tls"
	"encoding/json"
	"log"
)

const (
	//--------------启动模式
	RUNMODE_SANDBOX = 0 //启动沙河模式
	RUNMODE_ONLINE  = 1 //启动线上的模式

	//苹果发送Push
	ADDR_SANDBOX  = "gateway.sandbox.push.apple.com:2195"
	ADDR_ONLINE   = "gateway.push.apple.com:2195"
	ADDR_FEEDBACK = "feedback.push.apple.com:2196"

	//---------定义返回状态码
	RESP_STATUS_SUCC                            = 200 //成功
	RESP_STATUS_ERROR                           = 500 //服务器端错误
	RESP_STATUS_INVALID_PROTO                   = 401 //不允许使用GET 请求发送数据
	RESP_STATUS_PAYLOAD_BODY_DECODE_ERROR       = 505 //payload 的body 存在反序列化失败的问题
	RESP_STATUS_PAYLOAD_BODY_DEEP_ITERATOR      = 505 //payload 的body 不允许多层嵌套
	RESP_STATUS_SEND_OVER_TRY_ERROR             = 506 //payload 的body 不允许多层嵌套
	RESP_STATUS_FETCH_FEEDBACK_OVER_LIMIT_ERROR = 507 //payload 的body 不允许多层嵌套
)

type response struct {
	Status int         `json:"status,omitempty"`
	Error  error       `json:"error,omitempty"`
	Body   interface{} `json:"body,omitempty"` //只有在response的时候才会有
}

func (self *response) Marshal() []byte {
	data, err := json.Marshal(self)
	if nil != err {
		//就是数据哦有问题了
		return nil
	} else {
		return data
	}
}

type Option struct {
	bindAddr     string
	cert         tls.Certificate
	pushAddr     string
	feedbackAddr string
	expiredTime  uint32
}

func NewOption(bindaddr string, certpath string, keypath string, runmode int) Option {
	pushaddr := ADDR_SANDBOX
	if runmode == 1 {
		//启动sandbox
		pushaddr = ADDR_ONLINE
	}

	//加载证书
	cert, err := tls.LoadX509KeyPair(certpath, keypath)
	if nil != err {
		log.Printf("LOAD CERT FAIL|%s\n", err.Error())
		panic(err)
	}

	return Option{bindAddr: bindaddr, cert: cert, pushAddr: pushaddr, feedbackAddr: ADDR_FEEDBACK}
}
