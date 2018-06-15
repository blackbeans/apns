package apns

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
	"golang.org/x/net/http2"
	"github.com/blackbeans/log4go"
)

const (
	//开发环境
	URL_DEV = "api.development.push.apple.com:443"
	//正式环境
	URL_PRODUCTION = "api.push.apple.com:443"
)

type Notification struct {
	Topic       string
	ApnsID      string
	CollapseID  string
	Priority    int
	Expiration  time.Time
	DeviceToken string
	Payload     PayLoad
	Response    Response
}

//alert
type Alert struct {
	Body         string        `json:"body,omitempty"`
	ActionLocKey string        `json:"action-loc-key,omitempty"`
	LocKey       string        `json:"loc-key,omitempty"`
	LocArgs      []interface{} `json:"loc-args,omitempty"`
}

type Aps struct {
	Alert string `json:"alert,omitempty"`
	Badge int    `json:"badge,omitempty"` //显示气泡数
	Sound string `json:"sound"`           //控制push弹出的声音
}

type PayLoad struct {
	Aps Aps `json:"aps"`
}

//响应结果
type Response struct {
	Status int    `json:"status"`
	Reason string `json:"reason"`
}

//apns的链接
type ApnsConn struct {
	ctx             context.Context
	cancel          context.CancelFunc
	cert            *tls.Config //ssl证书
	hostport        string
	worktime        time.Time
	keepalivePeriod time.Duration

	c     *http2.ClientConn
	conn  net.Conn
	alive bool //是否存活
}

//NewApnsConn ...
func NewApnsConn(
	ctx context.Context,
	certificates tls.Certificate,
	hostport string,
	keepalivePeriod time.Duration) (*ApnsConn, error) {

	tlsConfig := &tls.Config{}
	tlsConfig.Certificates = []tls.Certificate{certificates}
	tlsConfig.InsecureSkipVerify = true
	if len(certificates.Certificate) > 0 {
		tlsConfig.BuildNameToCertificate()
	}

	ctx, cancel := context.WithCancel(ctx)
	conn := &ApnsConn{
		ctx:             ctx,
		cancel:          cancel,
		cert:            tlsConfig,
		hostport:        hostport,
		keepalivePeriod: keepalivePeriod}
	err := conn.Open()
	go conn.keepalive()
	return conn, err
}

//keepalive
func(self *ApnsConn) keepalive(){

		ticker := time.NewTicker(5 * time.Second)
		for self.alive {
			select {
			case <-ticker.C:
				//send ping if connection is  still alive and connection is idle for half of keepalivePeriod
				if nil != self.c && self.alive &&
					time.Since(self.worktime) > self.keepalivePeriod {
					err := self.c.Ping(self.ctx)
					if nil != err {
						log4go.WarnLog("service","CheckAlive|%s|Ping|FAIL|...",self.hostport)
						self.close0()
						//重新连接
						self.Open()
					} else {
						log4go.DebugLog("service","CheckAlive|%s|Ping|SUCC|...",self.hostport)
						break
					}
				}
			case <-self.ctx.Done():
				ticker.Stop()
				if nil != self.c {
					self.Destroy()
				}
			}
		}
}

//打开apns的链接
func (self *ApnsConn) Open() error {

	dialer := &net.Dialer{
		Timeout:   self.keepalivePeriod * 2,
		KeepAlive: self.keepalivePeriod}

	DialTLS := func(network, addr string, cfg *tls.Config) (net.Conn, error) {
		conn, err := tls.DialWithDialer(dialer,
			network, self.hostport, self.cert)
		if nil != err {
			return nil, err
		}
		return conn, err
	}

	conn, err := DialTLS("tcp", self.hostport, self.cert)
	if nil != err {
		return err
	}

	transport := &http2.Transport{
		TLSClientConfig: self.cert}
	h2c, err := transport.NewClientConn(conn.(*tls.Conn))
	if nil != err {
		return err
	}

	//open http2
	self.c = h2c
	self.conn = conn
	self.alive = true
	log.Printf("Reconnect Apns|SUCC|...")

	return nil
}

//发送消息
func (self *ApnsConn) SendMessage(notification *Notification) error {
	data, err := json.Marshal(notification.Payload)
	if nil != err {
		return errors.New("Invalid Payload !")
	}

	domain, _, _ := net.SplitHostPort(self.hostport)
	url := fmt.Sprintf("https://%s/3/device/%v", domain, notification.DeviceToken)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if nil != err {
		log.Printf("CreateReq|FAIL|%v|%s|%s", err, url, string(data))
		return err
	}
	setHeaders(req, notification)
	response, err := self.c.RoundTrip(req)
	if nil != err {
		log.Printf("FireReq|FAIL|%v|%s|%s", err, url, string(data))
		return err
	}
	defer response.Body.Close()

	//reset worktime time
	self.worktime = time.Now()

	decoder := json.NewDecoder(response.Body)
	resp := &Response{}
	resp.Status = response.StatusCode
	if err = decoder.Decode(&resp); nil != err && err != io.EOF {
		log.Printf("UnmarshaldResponse|FAIL|%v|%s|%s", err, url, string(data))
		return err
	}
	notification.Response = *resp
	return nil
}

//config header
func setHeaders(r *http.Request, n *Notification) {
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	if n.Topic != "" {
		r.Header.Set("apns-topic", n.Topic)
	}
	if n.ApnsID != "" {
		r.Header.Set("apns-id", n.ApnsID)
	}
	if n.CollapseID != "" {
		r.Header.Set("apns-collapse-id", n.CollapseID)
	}
	if n.Priority > 0 {
		r.Header.Set("apns-priority", fmt.Sprintf("%v", n.Priority))
	}
	if !n.Expiration.IsZero() {
		r.Header.Set("apns-expiration", fmt.Sprintf("%v", n.Expiration.Unix()))
	}
}

func(self *ApnsConn) close0(){
	if self.alive {
		self.alive = false
		self.conn.Close()
	}
}

func (self *ApnsConn) Destroy() {
	self.cancel()
	self.close0()

}
