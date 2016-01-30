package apns

import (
	"crypto/tls"
	"errors"
	"fmt"
	log "github.com/blackbeans/log4go"
	"go-apns/entry"
	_ "math/rand"
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
	resendCounter   *entry.Counter
}

func NewDefaultApnsClient(cert tls.Certificate, pushGateway string,
	feedbackChan chan<- *entry.Feedback, feedbackGateWay string,
	storage entry.IMessageStorage) *ApnsClient {

	//发送失败后的响应channel
	respChan := make(chan *entry.Response, 1000)

	deadline := 10 * time.Second
	err, factory := NewConnPool(20, 30, 50, 10*time.Minute, func(id int32) (error, IConn) {
		err, apnsconn := NewApnsConnection(respChan, cert, pushGateway, deadline, id)
		return err, apnsconn
	})

	if nil != err {
		log.CriticalLog("push_client", "APN SERVICE|CREATE CONNECTION POOL|FAIL|%s", err)
		return nil
	}
	err, feedbackFactory := NewConnPool(1, 2, 5, 10*time.Minute, func(id int32) (error, IConn) {
		err, conn := NewFeedbackConn(feedbackChan, cert, feedbackGateWay, deadline, id)
		return err, conn
	})
	if nil != err {
		log.CriticalLog("push_client", "APN SERVICE|CREATE FEEDBACK CONNECTION POOL|FAIL|%s", err)
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
		running: true, maxttl: 3, storage: storage, sendCounter: &entry.Counter{}, failCounter: &entry.Counter{}, resendCounter: &entry.Counter{}}
	go func() {
		for client.running {
			aa, ac, am := factory.MonitorPool()
			fa, fc, fm := feedbackFactory.MonitorPool()
			storageCap := client.storage.Length()
			log.InfoLog("apns_pool", "APNS-POOL|%d/%d/%d\tFEEDBACK-POOL/%d/%d/%d\tdeliver/fail:%d/%d\tstorageLen:%d\tresend:%d",
				aa, ac, am, fa, fc, fm,
				client.sendCounter.Changes(), client.failCounter.Changes(), storageCap, client.resendCounter.Changes())
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
	token, err := entry.WrapDeviceToken(deviceToken)
	if nil != err {
		return err
	}
	pl, err := entry.WrapPayLoad(&payload)
	if nil != err {
		return err
	}
	message.AddItem(token, pl)
	//直接发送的没有返回值
	return self.sendMessage(message)
}

//发送rich型的notification内部会重试
func (self *ApnsClient) SendEnhancedNotification(expiriedTime uint32, deviceToken string, pl entry.PayLoad) error {

	message := entry.NewMessage(entry.CMD_ENHANCE_NOTIFY, self.maxttl, entry.MESSAGE_TYPE_ENHANCED)
	payload, err := entry.WrapPayLoad(&pl)
	if nil == payload || nil != err {
		return errors.New(fmt.Sprintf("SendEnhancedNotification|PAYLOAD|ENCODE|FAIL|%s", err))
	}

	token, err := entry.WrapDeviceToken(deviceToken)
	if nil != err {
		return err
	}
	expiry := uint32(time.Now().Add(time.Duration(int64(expiriedTime) * int64(time.Second))).Unix())
	message.AddItem(entry.WrapExpirationDate(expiry), token, payload)

	return self.sendMessage(message)
}

func (self *ApnsClient) sendMessage(msg *entry.Message) error {
	var sendError error
	//重发逻辑
	for i := 0; i < 3; i++ {

		err, conn := self.factory.Get()
		if nil != err || nil == conn || !conn.IsAlive() {
			_, json := msg.Encode()
			log.ErrorLog("push_client", "APNSCLIENT|SEND MESSAGE|FAIL|GET CONN|FAIL|%s|%s", err, string(json))
			sendError = errors.New("GET APNS CONNECTION FAIL")
			continue
		}

		//将当前enchanced发送的数据写入到storage中
		if msg.MsgType == entry.MESSAGE_TYPE_ENHANCED {
			id := uint32(0)
			if nil != self.storage {
				//正常发送的记录即可
				id = self.storage.Insert(msg)
				if id <= 0 {
					_, json := msg.Encode()
					log.WarnLog("push_client", "APNSCLIENT|SEND MESSAGE|FAIL|Store FAIL|ID Zero|Try Send|%s", string(json))
				}
			}
			msg.IdentifierId = id
			// if rand.Intn(100) == 0 {
			// 	log.Printf("APNSCLIENT|sendMessage|RECORD MESSAGE|%s\n", msg)
			// }
		} else {
			//否则丢弃不开启重发........
		}

		//直接发送的没有返回值
		sendError = conn.sendMessage(msg)
		self.sendCounter.Incr(1)
		if nil != sendError {
			self.failCounter.Incr(1)
			//连接有问题直接销毁
			releaseErr := self.factory.ReleaseBroken(conn)
			log.ErrorLog("push_client", "APNSCLIENT|SEND MESSAGE|FAIL|RELEASE BROKEN CONN|FAIL|%s|%s", sendError, releaseErr)

		} else {
			//发送成功归还连接
			self.factory.Release(conn)
			break
		}
	}

	return sendError
}

func (self *ApnsClient) FetchFeedback(limit int) error {
	err, conn := self.feedbackFactory.Get()
	if nil != err {
		return err
	}
	feedbackconn := conn.(*FeedbackConn)
	defer func() {
		err := self.feedbackFactory.Release(conn)
		if nil != err {
			//这里如果有错误就是BUG，归还连接失败，就是说明有游离态的连接
			log.ErrorLog("push_client", "APNSCLIENT|RELEASE CONN|FAIL")
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
