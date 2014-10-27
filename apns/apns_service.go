package apns

import (
	"errors"
	"go-apns/entry"
	"log"
)

//用于使用的api接口

type ApnsClient struct {
	factory         IConnFactory
	feedbackFactory IConnFactory //用于查询feedback的链接
}

func NewApnsClient(factory IConnFactory, feedbackFactory IConnFactory) *ApnsClient {
	client := &ApnsClient{factory: factory, feedbackFactory: feedbackFactory}
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

	conn := self.factory.get()
	defer self.factory.release(conn)
	//直接发送的没有返回值
	err := conn.sendMessage(msg)
	if nil != err {
		log.Printf("APNSCLIENT|SEND MESSAGE|FAIL|%t\n", err)
		return err
	}
	return nil
}

func (self *ApnsClient) FetchFeedback(ch chan<- *entry.Feedback) {
	conn := self.feedbackFactory.get()
	defer self.feedbackFactory.release(conn)
	conn.readFeedBack(ch)
}

func (self *ApnsClient) Destory() {
	self.factory.close()
	self.feedbackFactory.close()
}
