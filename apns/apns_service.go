package apns

import (
	"crypto/tls"
	"errors"
	"go-apns/entry"
	"log"
	"time"
)

//用于使用的api接口

type ApnsClient struct {
	factory         IConnFactory
	feedbackFactory IConnFactory //用于查询feedback的链接
	running         bool
}

func NewDefaultApnsClient(cert tls.Certificate,
	respChan chan<- *entry.Response, pushGateway string,
	feedbackChan chan<- *entry.Feedback, feedbackGateWay string) *ApnsClient {

	deadline := 10 * time.Second
	heartCheck := int32(1)
	err, factory := NewConnPool(10, 20, 50, 10*time.Second, func() (error, IConn) {
		err, apnsconn := NewApnsConnection(respChan, cert, pushGateway, deadline, heartCheck)
		return err, apnsconn
	})

	if nil != err {
		log.Panicf("APN SERVICE|CREATE CONNECTION POOL|FAIL|%s", err)
		return nil
	}
	err, feedbackFactory := NewConnPool(1, 2, 5, 10*time.Minute, func() (error, IConn) {
		err, conn := NewFeedbackConn(feedbackChan, cert, feedbackGateWay, deadline, heartCheck)
		return err, conn
	})
	if nil != err {
		log.Panicf("APN SERVICE|CREATE FEEDBACK CONNECTION POOL|FAIL|%s", err)
		return nil
	}

	return NewApnsClient(factory, feedbackFactory)
}

func NewApnsClient(factory IConnFactory, feedbackFactory IConnFactory) *ApnsClient {
	client := &ApnsClient{factory: factory, feedbackFactory: feedbackFactory, running: true}
	go func() {
		for client.running {
			aa, ac, am := factory.MonitorPool()
			fa, fc, fm := feedbackFactory.MonitorPool()
			log.Printf("APNS-POOL|%d/%d/%d\tFEEDBACK-POOL/%d/%d/%d\n", aa, ac, am, fa, fc, fm)
			time.Sleep(1 * time.Second)
		}
	}()
	return client
}

//发送简单的notification
func (self *ApnsClient) SendSimpleNotification(deviceToken string, payload entry.PayLoad) error {
	message := entry.NewMessage(entry.CMD_SIMPLE_NOTIFY)
	message.AddItem(entry.WrapDeviceToken(deviceToken), entry.WrapPayLoad(&payload))
	//直接发送的没有返回值
	return self.sendMessage(message)
}

func (self *ApnsClient) SendEnhancedNotification(identifier, expiriedTime uint32, deviceToken string, pl entry.PayLoad) error {
	message := entry.NewMessage(entry.CMD_ENHANCE_NOTIFY)
	payload := entry.WrapPayLoad(&pl)
	if nil == payload {
		return errors.New("SendEnhancedNotification|PAYLOAD|ENCODE|FAIL")
	}
	message.AddItem(entry.WrapNotifyIdentifier(identifier), entry.WrapExpirationDate(expiriedTime),
		entry.WrapDeviceToken(deviceToken), payload)
	return self.sendMessage(message)
}

func (self *ApnsClient) sendMessage(msg *entry.Message) error {

	err, conn := self.factory.Get(5 * time.Second)
	if nil != err {
		return err
	}
	defer self.factory.Release(conn)
	//直接发送的没有返回值
	err = conn.(*ApnsConnection).sendMessage(msg)
	if nil != err {
		log.Printf("APNSCLIENT|SEND MESSAGE|FAIL|%t\n", err)
		return err
	}
	return nil
}

func (self *ApnsClient) FetchFeedback(limit int) error {
	err, conn := self.feedbackFactory.Get(5 * time.Second)
	if nil != err {
		return err
	}
	feedbackconn := conn.(*FeedbackConn)
	defer func() {
		err := self.feedbackFactory.Release(conn)
		if nil != err {
			//这里如果有错误就是BUG，归还连接失败，就是说明有游离态的连接
			log.Printf("APNS SERVICE|RELEASE CONN|FAIL")
		}
	}()
	go func() {
		feedbackconn.readFeedBack(limit)
	}()
	return nil
}

func (self *ApnsClient) Destory() {
	self.feedbackFactory.Shutdown()
	self.factory.Shutdown()
	self.running = false
}
