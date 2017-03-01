package apns

import (
	"crypto/tls"
	"errors"
	"reflect"
	"time"

	log "github.com/blackbeans/log4go"
)

type IConn interface {
	Open() error
	IsAlive() bool
	Close()
	sendMessage(msg *Message) error
}

const (
	CONN_READ_BUFFER_SIZE  = 256
	CONN_WRITE_BUFFER_SIZE = 512
)

type ApnsConnection struct {
	cert         tls.Certificate //ssl证书
	hostport     string
	deadline     time.Duration
	conn         *tls.Conn
	responseChan chan<- *Response
	alive        bool  //是否存活
	connectionId int32 //当前连接的标识
}

func NewApnsConnection(responseChan chan<- *Response,
	certificates tls.Certificate, hostport string, deadline time.Duration, connectionId int32) (error, *ApnsConnection) {

	conn := &ApnsConnection{
		cert:         certificates,
		hostport:     hostport,
		deadline:     deadline,
		responseChan: responseChan,
		connectionId: connectionId}
	return conn.Open(), conn
}

func (self *ApnsConnection) Open() error {
	conn, err := self.dial()
	if nil != err {
		return err
	}
	self.conn = conn
	self.alive = true
	go self.waitRepsonse()
	return nil
}

func (self *ApnsConnection) waitRepsonse() {
	//这里需要优化是否同步读取结果
	buff := make([]byte, ERROR_RESPONSE, ERROR_RESPONSE)
	for self.alive {
		//同步读取当前conn的结果
		length, err := self.conn.Read(buff[:ERROR_RESPONSE])
		if nil != err {
			log.InfoLog("push_client", "CONNECTION|%s|READ RESPONSE|FAIL|%v|%d/%d",
				self.conn.RemoteAddr().String(),
				err, length, len(buff))
			break
		}

		if length > 0 {
			response := &Response{}
			response.Unmarshal(self.connectionId, buff)
			//如果状态吗是10则关闭
			if response.Status == 10 {
				self.Close()
			}
			self.responseChan <- response
		}
	}

	//已经读取到了错误信息直接关闭
	self.Close()
}

func (self *ApnsConnection) name() string {
	return reflect.TypeOf(*self).Name()
}

func (self *ApnsConnection) dial() (*tls.Conn, error) {

	config := tls.Config{}
	config.Certificates = []tls.Certificate{self.cert}
	config.InsecureSkipVerify = true

	conn, err := tls.Dial("tcp", self.hostport, &config)
	if nil != err {
		//connect fail
		log.WarnLog("push_client", "CONNECTION|%s|DIAL CONNECT|FAIL|%s|%s", self.name(), self.hostport, err.Error())
		return nil, err
	}

	return conn, nil
}

func (self *ApnsConnection) sendMessage(msg *Message) error {
	if !self.alive {
		//存活但是不适合握手完成状态则失败
		return errors.New("CONNECTION|SEND MESSAGE|FAIL|Connection Closed!")
	}

	err, packet := msg.Encode()
	if nil != err {
		return err
	}
	//消息使用当前连接发送做记录
	msg.ConnectionId = self.connectionId
	length, sendErr := self.conn.Write(packet)
	if nil != sendErr || length != len(packet) {
		log.WarnLog("push_client", "CONNECTION|SEND MESSAGE|FAIL|%s", sendErr)
	} else {
		log.DebugLog("push_client", "CONNECTION|SEND MESSAGE|SUCC")

	}
	return sendErr

}

func (self *ApnsConnection) IsAlive() bool {
	return self.alive
}

func (self *ApnsConnection) Close() {

	self.alive = false
	self.conn.Close()
	log.InfoLog("push_client", "APNS CONNECTION|%s|CLOSED ...", self.name())
}
