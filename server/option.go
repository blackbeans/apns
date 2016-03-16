package server

import (
	"crypto/tls"

	log "github.com/blackbeans/log4go"
	"io/ioutil"
	"net/http"
	"strings"
)

const (

	//--------------启动模式
	STARTMODE_MOCK   = 0 //启动mock模式 用于压测
	STARTMODE_ONLINE = 1 //启动线上的模式

	//--------------启动模式
	RUNMODE_SANDBOX = 0 //启动沙河模式
	RUNMODE_ONLINE  = 1 //启动线上的模式

	NOTIFY_SIMPLE_FORMAT   = "0"
	NOTIFY_ENHANCED_FORMAT = "1"

	//苹果发送Push
	ADDR_SANDBOX          = "gateway.sandbox.push.apple.com:2195"
	ADDR_ONLINE           = "gateway.push.apple.com:2195"
	ADDR_FEEDBACK         = "feedback.push.apple.com:2196"
	ADDR_FEEDBACK_SANDBOX = "feedback.sandbox.push.apple.com:2196"

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
)

type Option struct {
	StartMode       int
	BindAddr        string
	Cert            tls.Certificate
	PushAddr        string
	FeedbackAddr    string
	ExpiredTime     uint32 //默认十分钟过期
	StorageCapacity int    //用于存储临时发送的数据量
}

func NewOption(startMode int, bindaddr string, certpath string, keypath string, runmode int, storageCapacity int) Option {
	pushaddr := ADDR_SANDBOX
	feedbackAddr := ADDR_FEEDBACK_SANDBOX
	if runmode == 1 {
		//启动sandbox
		pushaddr = ADDR_ONLINE
		feedbackAddr = ADDR_FEEDBACK

	}
	cert := loadCert(certpath, keypath)

	return Option{StartMode: startMode, BindAddr: bindaddr,
		Cert: cert, PushAddr: pushaddr, FeedbackAddr: feedbackAddr,
		ExpiredTime: 24 * 60 * 60, StorageCapacity: storageCapacity}
}
func loadCert(certpath string, keypath string) tls.Certificate {

	var cert tls.Certificate
	var err error
	//判断当前文件协议是从http方式读取么
	if strings.HasPrefix(keypath, "http://") || strings.HasPrefix(keypath, "https://") {

		log.Info("keyPath:%s\ncertPath:%s\n", keypath, certpath)
		resp, kerr := http.Get(keypath)
		if nil != kerr {
			log.Exitf("loading key from [%s] is fail! -> %s", keypath, kerr)
		}
		key, kerr := ioutil.ReadAll(resp.Body)
		if nil != kerr {
			log.Exitf("reading key from [%s] is fail! -> %s", keypath, kerr)
		}
		defer resp.Body.Close()

		resp, cerr := http.Get(certpath)
		if nil != cerr {
			log.Exitf("loading cert from [%s] is fail! -> %s", certpath, cerr)
		}
		certb, cerr := ioutil.ReadAll(resp.Body)
		if nil != cerr {
			log.Exitf("reading cert from [%s] is fail! -> %s", certpath, cerr)
		}

		defer resp.Body.Close()

		cert, err = tls.X509KeyPair(certb, key)

	} else {
		//直接读取文件的
		//加载证书
		cert, err = tls.LoadX509KeyPair(certpath, keypath)

	}

	if nil != err {
		log.Error("LOAD CERT FAIL|%s", err.Error())
		panic(err)
	}

	return cert

}
