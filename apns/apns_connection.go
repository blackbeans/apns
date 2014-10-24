package apns

import (
	"crypto/tls"
	"go-apns/entry"
	"log"
	"time"
)

const (
	CONN_READ_BUFFER_SIZE  = 256
	CONN_WRITE_BUFFER_SIZE = 512
)

//连接工厂
type IConnFactory interface {
	get() *ApnsConnection         //获取一个连接
	release(conn *ApnsConnection) //释放对应的链接
	close()                       //关闭当前的
}

type ApnsConnection struct {
	cert         tls.Certificate //ssl证书
	hostport     string
	deadline     time.Duration
	heartCheck   int32 //heart check
	conn         *tls.Conn
	responseChan chan<- *entry.Response
}

func NewApnsConnection(responseChan chan<- *entry.Response, certificates tls.Certificate, hostport string, deadline time.Duration, heartCheck int32) *ApnsConnection {

	return &ApnsConnection{cert: certificates,
		hostport:   hostport,
		deadline:   deadline,
		heartCheck: heartCheck}
}

func (self *ApnsConnection) open() error {
	err := self.dial()
	if nil != err {
		return err
	}
	//启动读取数据
	go self.waitRepsonse()
	return nil
}

func (self *ApnsConnection) waitRepsonse() {
	//这里需要优化是否同步读取结果
	buff := make([]byte, entry.ERROR_RESPONSE, entry.ERROR_RESPONSE)
	//同步读取当前conn的结果
	length, err := self.conn.Read(buff[:entry.ERROR_RESPONSE])
	if nil != err || length != len(buff) {
		log.Printf("CONNECTION|READ RESPONSE|FAIL|%s\n", err)
		self.responseChan <- nil
	} else {
		response := &entry.Response{}
		response.Unmarshal(buff)
		self.responseChan <- response
	}

	//已经读取到了错误信息直接关闭
	self.close()

}

func (self *ApnsConnection) dial() error {

	config := tls.Config{}
	config.Certificates = []tls.Certificate{self.cert}
	config.InsecureSkipVerify = true
	conn, err := tls.Dial("tcp", self.hostport, &config)
	if nil != err {
		//connect fail
		log.Printf("CONNECTION|DIAL CONNECT|FAIL|%s|%s\n", self.hostport, err.Error())
		return err
	}

	// conn.SetDeadline(0 * time.Second)
	for {
		state := conn.ConnectionState()
		if state.HandshakeComplete {
			log.Printf("CONNECTION|HANDSHAKE SUCC\n")
			break
		}
		time.Sleep(1 * time.Second)
	}

	self.conn = conn
	return nil
}

func (self *ApnsConnection) sendMessage(msg *entry.Message) error {

	err, packet := msg.Encode()
	log.Printf("%d|%t", len(packet), packet)
	if nil != err {
		return err
	}
	length, err := self.conn.Write(packet)
	if nil != err || length != len(packet) {
		log.Printf("CONNECTION|SEND MESSAGE|FAIL|%s\n", err)
		return err
	}
	return nil
}

func (self *ApnsConnection) close() {
	self.conn.Close()
}
