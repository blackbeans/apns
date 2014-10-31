package apns

import (
	"crypto/tls"
	"go-apns/entry"
	"math"
	"testing"
	"time"
)

const (
	CERT_PATH       = "/Users/blackbeans/workspace/github/go-apns/pushcert.pem"
	KEY_PATH        = "/Users/blackbeans/workspace/github/go-apns/key.pem"
	PUSH_APPLE      = "gateway.push.apple.com:2195"
	FEED_BACK_APPLE = "feedback.push.apple.com:2196"
	apnsToken       = "bb9182d786ecc6878eae2a3a4c87e3fc1b4e8f861717210af8810384aa67369d"
)

func TestSendMessage(t *testing.T) {

	cert, err := tls.LoadX509KeyPair(CERT_PATH, KEY_PATH)
	if nil != err {
		t.Logf("READ CERT FAIL|%s", err.Error())
		t.Fail()
		return
	}

	ch := make(chan *entry.Response, 1)
	err, conn := NewApnsConnection(ch, cert, PUSH_APPLE, 5*time.Second, 1)

	if nil != err {
		t.Fail()
		return
	}

	feedbackChan := make(chan *entry.Feedback, 100)
	err, feedback := NewFeedbackConn(feedbackChan, cert, FEED_BACK_APPLE, 5*time.Second, 1)
	if nil != err {
		t.Fail()
		return
	}

	body := "hello apns"
	payload := entry.NewSimplePayLoad("ms.caf", 1, body)
	client := NewApnsClient(&ConnFacotry{conn: conn}, &ConnFacotry{conn: feedback})
	for i := 0; i < 1; i++ {
		err := client.SendEnhancedNotification(1, math.MaxUint32, apnsToken, *payload)
		// err := client.SendSimpleNotification(apnsToken, payload)
		t.Logf("SEND NOTIFY|%s\n", err)
	}

	go func() {
		//测试feedback
		err := client.FetchFeedback()
		if nil != err {
			t.Logf("FETCH FEEDBACK|FAIL |%s\n", err)
		}

	}()

	for i := 0; i < 2; i++ {
		select {
		case <-time.After(20 * time.Second):
		case resp := <-ch:
			t.Logf("===============%t|EXIT", resp)
			//如果有返回错误则说明发送失败的
			t.Fail()
		case fb := <-feedbackChan:
			i := 0
			for i < 100 {
				t.Logf("FEEDBACK===============%s|EXIT", fb)
				i++
			}
			//如果有返回错误则说明发送失败的
		}
	}

	client.Destory() //
}

func TestPoolSendMessage(t *testing.T) {
	cert, err := tls.LoadX509KeyPair(CERT_PATH, KEY_PATH)
	if nil != err {
		t.Logf("READ CERT FAIL|%s", err.Error())
		t.Fail()
		return
	}

	responseChan := make(chan *entry.Response, 10)
	feedbackChan := make(chan *entry.Feedback, 1000)

	body := "hello apns"
	payload := entry.NewSimplePayLoad("ms.caf", 1, body)
	client := NewDefaultApnsClient(cert, responseChan, PUSH_APPLE, feedbackChan, FEED_BACK_APPLE)
	for i := 0; i < 1; i++ {
		err := client.SendEnhancedNotification(1, math.MaxUint32, apnsToken, *payload)
		// err := client.SendSimpleNotification(apnsToken, payload)
		t.Logf("SEND NOTIFY|%s\n", err)
	}

	go func() {
		//测试feedback
		err := client.FetchFeedback()
		if nil != err {
			t.Logf("FETCH FEEDBACK|FAIL |%s\n", err)
		}

	}()

	for i := 0; i < 2; i++ {
		select {
		case <-time.After(20 * time.Second):
		case resp := <-responseChan:
			t.Logf("===============%t|EXIT", resp)
			//如果有返回错误则说明发送失败的
			t.Fail()
		case fb := <-feedbackChan:
			i := 0
			for i < 100 {
				t.Logf("FEEDBACK===============%s|EXIT", fb)
				i++
			}
			//如果有返回错误则说明发送失败的
		}
	}

	client.Destory() //
}

type ConnFacotry struct {
	conn IConn
}

func (self ConnFacotry) MonitorPool() (int, int, int) {
	return 1, 1, 1
}

func (self ConnFacotry) Get(timeout time.Duration) (error, IConn) {
	return nil, self.conn
} //获取一个连接
func (self ConnFacotry) Release(conn IConn) error {
	return nil
} //释放对应的链接
func (self ConnFacotry) Shutdown() {
	self.conn.Close()
} //关闭当前的
