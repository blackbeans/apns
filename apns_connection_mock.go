package apns

import (
	"crypto/tls"
	_ "log"
	"time"
)

type ApnsConnectionMock struct {
	ApnsConnection
}

func NewApnsConnectionMock(responseChan chan<- *Response, certificates tls.Certificate,
	hostport string, deadline time.Duration, id int32) (error, *ApnsConnectionMock) {

	conn := &ApnsConnectionMock{}
	conn.ApnsConnection.cert = certificates
	conn.ApnsConnection.hostport = hostport
	conn.ApnsConnection.deadline = deadline
	conn.responseChan = responseChan
	conn.connectionId = id
	return conn.Open(), conn
}

func (self *ApnsConnectionMock) sendMessage(msg *Message) error {
	//do nothing
	// log.Println("ApnsConnectionMock|sendMessage|SUCC!")
	return nil
}
