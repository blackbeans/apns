package apns

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/blackbeans/log4go"
)

const (

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
)

type Config struct {
	Env             string
	Sound           string
	ExpiredSec      int
	StorageCapacity int    //用于存储临时发送的数据量
	CertPathPrefix  string // 证书地址
}

type ApnsOption struct {
	Cert            tls.Certificate
	PushAddr        string
	FeedbackAddr    string
	ExpiredTime     uint32 //默认十分钟过期
	Sound           string //声音文件名
	StorageCapacity int    //用于存储临时发送的数据量
}

func NewApnsOption(option Config) ApnsOption {
	pushaddr := ADDR_ONLINE
	feedbackAddr := ADDR_FEEDBACK
	certpath := fmt.Sprintf("%s/online_cert.pem", option.CertPathPrefix)
	keypath := fmt.Sprintf("%s/online_key.pem", option.CertPathPrefix)
	if option.Env == "dev" {
		//启动sandbox
		pushaddr = ADDR_SANDBOX
		feedbackAddr = ADDR_FEEDBACK_SANDBOX
		certpath = fmt.Sprintf("%s/dev_cert.pem", option.CertPathPrefix)
		keypath = fmt.Sprintf("%s/dev_key.pem", option.CertPathPrefix)
	}
	log.Info("keyPath:%s\ncertPath:%s\n", keypath, certpath)
	cert := loadCert(certpath, keypath)

	return ApnsOption{
		Cert:            cert,
		PushAddr:        pushaddr,
		FeedbackAddr:    feedbackAddr,
		ExpiredTime:     uint32(option.ExpiredSec),
		Sound:           option.Sound,
		StorageCapacity: option.StorageCapacity}
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
