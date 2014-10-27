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
	apnsToken       = "f232e31293b0d63ba886787950eb912168f182e6c91bc6bdf39d162bf5d7697d"
)

func TestSendMessage(t *testing.T) {

	cert, err := tls.LoadX509KeyPair(CERT_PATH, KEY_PATH)
	if nil != err {
		t.Logf("READ CERT FAIL|%s", err.Error())
		t.Fail()
		return
	}

	ch := make(chan *entry.Response, 1)
	conn := NewApnsConnection(ch, cert, PUSH_APPLE, 5*time.Second, 1)
	conn.open()

	feedback := NewApnsConnection(ch, cert, FEED_BACK_APPLE, 5*time.Second, 1)
	feedback.open()

	body := "hello apns"
	payload := entry.NewSimplePayLoad("ms.caf", 1, body)
	client := NewApnsClient(&ConnFacotry{conn: conn}, &ConnFacotry{conn: feedback})
	for i := 0; i < 1; i++ {
		err := client.SendEnhancedNotification(1, math.MaxUint32, apnsToken, *payload)
		// err := client.SendSimpleNotification(apnsToken, payload)
		t.Logf("SEND NOTIFY|%s\n", err)
	}

	tch := time.After(5 * time.Second)
	select {
	case <-tch:
	case resp := <-ch:
		t.Logf("===============%t|EXIT", resp)
		//如果有返回错误则说明发送失败的
		t.Fail()

	}

	fbch := make(chan *entry.Feedback, 1000)

	go func() {
		//测试feedback
		client.FetchFeedback(fbch)
	}()

	tch = time.After(5 * time.Second)
a:
	for {
		select {
		case <-tch:
			break a
		case fb := <-fbch:
			t.Logf("FEEDBACK===============%s|EXIT", fb)
			//如果有返回错误则说明发送失败的
		}
	}

	client.Destory() //
}

type ConnFacotry struct {
	conn *ApnsConnection
}

func (self ConnFacotry) get() *ApnsConnection {
	return self.conn
} //获取一个连接
func (self ConnFacotry) release(conn *ApnsConnection) {

} //释放对应的链接
func (self ConnFacotry) close() {
	self.conn.close()
} //关闭当前的
