package apns

import (
	"crypto/tls"
	"go-apns/entry"
	"log"
	"time"
)

//仅用于测试用的client 不走苹果发送，只看tps
func NewMockApnsClient(cert tls.Certificate, pushGateway string,
	feedbackChan chan<- *entry.Feedback, feedbackGateWay string,
	storage entry.IMessageStorage) *ApnsClient {

	//发送失败后的响应channel
	respChan := make(chan *entry.Response, 1000)

	deadline := 10 * time.Second
	err, factory := NewConnPool(10, 20, 50, 10*time.Second, func(id int32) (error, IConn) {
		err, apnsconn := NewApnsConnectionMock(respChan, cert, pushGateway, deadline, id)
		return err, apnsconn
	})

	if nil != err {
		log.Panicf("APN SERVICE|CREATE MOCK CONNECTION POOL|FAIL|%s", err)
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
