package apns

import (
	"crypto/tls"
	"go-apns/entry"
	"log"
	"reflect"
	"time"
)

type IConn interface {
	Open() error
	IsAlive() bool
	Close()
	sendMessage(msg *entry.Message) error
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
	responseChan chan<- *entry.Response
	alive        bool  //是否存活
	connectionId int32 //当前连接的标识
}

func NewApnsConnection(responseChan chan<- *entry.Response, certificates tls.Certificate, hostport string, deadline time.Duration, connectionId int32) (error, *ApnsConnection) {

	conn := &ApnsConnection{cert: certificates,
		hostport:     hostport,
		deadline:     deadline,
		connectionId: connectionId}
	return conn.Open(), conn
}

func (self *ApnsConnection) Open() error {
	err := self.dial()
	if nil != err {
		return err
	}
	//启动读取数据
	go self.waitRepsonse()
	self.alive = true
	return nil
}

func (self *ApnsConnection) waitRepsonse() {
	//这里需要优化是否同步读取结果
	buff := make([]byte, entry.ERROR_RESPONSE, entry.ERROR_RESPONSE)
	//同步读取当前conn的结果
	length, err := self.conn.Read(buff[:entry.ERROR_RESPONSE])
	if nil != err || length != len(buff) {
		log.Printf("CONNECTION|%s|READ RESPONSE|FAIL|%s\n", self.name(), err)
	} else {
		response := &entry.Response{}
		response.Unmarshal(buff)
		self.responseChan <- response
	}

	//已经读取到了错误信息直接关闭
	self.Close()

}

func (self *ApnsConnection) name() string {
	return reflect.TypeOf(*self).Name()
}

func (self *ApnsConnection) dial() error {

	config := tls.Config{}
	config.Certificates = []tls.Certificate{self.cert}
	config.InsecureSkipVerify = true
	conn, err := tls.Dial("tcp", self.hostport, &config)
	if nil != err {
		//connect fail
		log.Printf("CONNECTION|%s|DIAL CONNECT|FAIL|%s|%s\n", self.name(), self.hostport, err.Error())
		return err
	}

	// conn.SetDeadline(0 * time.Second)
	for {
		state := conn.ConnectionState()
		if state.HandshakeComplete {
			log.Printf("CONNECTION|%s|HANDSHAKE SUCC\n", self.name())
			break
		}
		time.Sleep(1 * time.Second)
	}
	self.conn = conn
	return nil
}

func (self *ApnsConnection) sendMessage(msg *entry.Message) error {
	//将当前的msg强制设置为当前conn的id作为标识
	msg.ProcessId = self.connectionId

	err, packet := msg.Encode()
	if nil != err {
		return err
	}
	//单链接重发3次
	var sendErr error
	for i := 0; i < 3; i++ {
		length, err := self.conn.Write(packet)
		if nil != err || length != len(packet) {
			sendErr = err
			log.Printf("CONNECTION|SEND MESSAGE|FAIL|%s|tryCount:%d|%s\n", err, i, msg)
		} else {
			break
		}
	}

	return sendErr
}

func (self *ApnsConnection) IsAlive() bool {
	return self.alive
}

func (self *ApnsConnection) Close() {

	self.alive = false
	self.conn.Close()
	log.Printf("APNS CONNECTION|%s|CLOSED ...", self.name())
}
