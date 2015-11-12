package apns

import (
	"crypto/tls"
	log "github.com/blackbeans/log4go"
	"go-apns/entry"
	"reflect"
	"time"
)

//feedback连接

type FeedbackConn struct {
	ApnsConnection
	feedbackChan chan<- *entry.Feedback
}

func NewFeedbackConn(feedbackChan chan<- *entry.Feedback, certificates tls.Certificate,
	hostport string, deadline time.Duration, id int32) (error, *FeedbackConn) {

	conn := &FeedbackConn{}
	conn.ApnsConnection.cert = certificates
	conn.ApnsConnection.hostport = hostport
	conn.ApnsConnection.deadline = deadline
	conn.feedbackChan = feedbackChan
	conn.connectionId = id
	return conn.Open(), conn
}

func (self *FeedbackConn) Open() error {
	err := self.dial()
	if nil != err {
		return err
	}
	//启动读取数据
	self.alive = true
	return nil
}

func (self *FeedbackConn) name() string {
	return reflect.TypeOf(*self).Name()
}

func (self *FeedbackConn) readFeedBack(limit int) {

	buff := make([]byte, entry.FEEDBACK_RESP, entry.FEEDBACK_RESP)
	for self.alive && limit > 0 {
		length, err := self.conn.Read(buff)
		//如果已经读完数据那么久直接退出
		if length == -1 || nil != err {
			self.feedbackChan <- nil
			break
		}

		//读取的数据
		feedback := entry.NewFeedBack(buff)
		self.feedbackChan <- feedback
		limit--
		buff = buff[:entry.FEEDBACK_RESP]
	}
	//本次读取完毕
	log.Debug("FEEDBACK CONNECTION|READ FEEDBACK|FINISHED!")

}
