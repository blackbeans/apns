package apns

import (
	"crypto/tls"
	"errors"
	"go-apns/entry"
	"log"
	"math/rand"
	"time"
)

//用于使用的api接口

type ApnsClient struct {
	factory         IConnFactory
	feedbackFactory IConnFactory //用于查询feedback的链接
	running         bool
	maxttl          uint8
	storage         entry.IMessageStorage
	sendCounter     *entry.Counter
	failCounter     *entry.Counter
}

func NewDefaultApnsClient(cert tls.Certificate, pushGateway string,
	feedbackChan chan<- *entry.Feedback, feedbackGateWay string,
	storage entry.IMessageStorage) *ApnsClient {

	//发送失败后的响应channel
	respChan := make(chan *entry.Response, 1000)

	deadline := 10 * time.Second
	err, factory := NewConnPool(10, 20, 50, 10*time.Second, func(id int32) (error, IConn) {
		err, apnsconn := NewApnsConnection(respChan, cert, pushGateway, deadline, id)
		return err, apnsconn
	})

	if nil != err {
		log.Panicf("APN SERVICE|CREATE CONNECTION POOL|FAIL|%s", err)
		return nil
	}
	err, feedbackFactory := NewConnPool(1, 2, 5, 10*time.Minute, func(id int32) (error, IConn) {
		err, conn := NewFeedbackConn(feedbackChan, cert, feedbackGateWay, deadline, id)
		return err, conn
	})
	if nil != err {
		log.Panicf("APN SERVICE|CREATE FEEDBACK CONNECTION POOL|FAIL|%s", err)
		return nil
	}

	return newApnsClient(factory, feedbackFactory, storage, respChan)
}

func NewApnsClient(factory IConnFactory, feedbackFactory IConnFactory, storage entry.IMessageStorage) *ApnsClient {
	//发送失败后的响应channel
	respChan := make(chan *entry.Response, 1000)
	return newApnsClient(factory, feedbackFactory, storage, respChan)
}

func newApnsClient(factory IConnFactory, feedbackFactory IConnFactory,
	storage entry.IMessageStorage, responseChannel chan *entry.Response) *ApnsClient {

	client := &ApnsClient{factory: factory, feedbackFactory: feedbackFactory,
		running: true, maxttl: 3, storage: storage, sendCounter: &entry.Counter{}, failCounter: &entry.Counter{}}
	go func() {
		for client.running {
			aa, ac, am := factory.MonitorPool()
			fa, fc, fm := feedbackFactory.MonitorPool()
			storageCap := client.storage.Length()

			log.Printf("APNS-POOL|%d/%d/%d\tFEEDBACK-POOL/%d/%d/%d\tstorageLen:%d\tdeliver/fail:%d/%d\n", aa, ac, am, fa, fc, fm, storageCap,
				client.sendCounter.Changes(), client.failCounter.Changes())
			time.Sleep(1 * time.Second)
		}
	}()
	//启动获取响应数据读取，并重发
	go client.onErrorResponseRecieve(responseChannel)

	return client

}

//发送简单的notification
func (self *ApnsClient) SendSimpleNotification(deviceToken string, payload entry.PayLoad) error {
	message := entry.NewMessage(entry.CMD_SIMPLE_NOTIFY, self.maxttl, entry.MESSAGE_TYPE_SIMPLE)
	message.AddItem(entry.WrapDeviceToken(deviceToken), entry.WrapPayLoad(&payload))
	//直接发送的没有返回值
	return self.sendMessage(message)
}

//发送rich型的notification内部会重试
func (self *ApnsClient) SendEnhancedNotification(identifier, expiriedTime uint32, deviceToken string, pl entry.PayLoad) error {
	id := entry.WrapNotifyIdentifier(identifier)
	message := entry.NewMessage(entry.CMD_ENHANCE_NOTIFY, self.maxttl, entry.MESSAGE_TYPE_ENHANCED)
	payload := entry.WrapPayLoad(&pl)
	if nil == payload {
		return errors.New("SendEnhancedNotification|PAYLOAD|ENCODE|FAIL")
	}
	message.AddItem(id, entry.WrapExpirationDate(expiriedTime),
		entry.WrapDeviceToken(deviceToken), payload)

	return self.sendMessage(message)
}

func (self *ApnsClient) sendMessage(msg *entry.Message) error {

	err, conn := self.factory.Get(5 * time.Second)
	if nil != err {
		return err
	}
	defer self.factory.Release(conn)

	//将当前enchanced发送的数据写入到storage中
	if nil != self.storage &&
		msg.MsgType == entry.MESSAGE_TYPE_ENHANCED {
		//正常发送的记录即可
		self.storage.Insert(entry.UmarshalIdentifier(msg), msg)
		// if rand.Intn(100) == 0 {
		// 	log.Printf("APNSCLIENT|sendMessage|RECORD MESSAGE|%s\n", msg)
		// }
	} else {
		//否则丢弃不开启重发........
	}

	for i := 0; i < 3; i++ {
		self.sendCounter.Incr(1)
		//直接发送的没有返回值
		err = conn.sendMessage(msg)

		if nil != err {
			self.failCounter.Incr(1)
			log.Printf("APNSCLIENT|SEND MESSAGE|FAIL|%s|tryCount:%d\n", err, i)
		} else {
			break
		}
	}
	return err
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
			log.Printf("APNSCLIENT|RELEASE CONN|FAIL")
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
