package apns

import (
	"crypto/tls"
	"go-apns/entry"
	"math"
	"testing"
	"time"
)

const (
	CERT_PATH  = "/Users/blackbeans/workspace/github/go-apns/pushcert.pem"
	KEY_PATH   = "/Users/blackbeans/workspace/github/go-apns/key.pem"
	PUSH_APPLE = "gateway.push.apple.com:2195"
	apnsToken  = "你自己的apnstoken"
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

	body := "hello apns"
	payload := entry.NewSimplePayLoad("ms.caf", 1, body)
	client := NewApnsClient(&ConnFacotry{conn: conn})
	for i := 0; i < 1; i++ {
		err := client.SendEnhancedNotification(1, math.MaxUint32, apnsToken, *payload)
		// err := client.SendSimpleNotification(apnsToken, payload)
		t.Logf("SEND NOTIFY|%s\n", err)
	}

	time.Sleep(10 * time.Second)
	client.Destory() //
	tch := time.After(5 * time.Second)
	select {
	case <-tch:
	case resp := <-ch:
		t.Logf("===============%t|EXIT", resp)
		//如果有返回错误则说明发送失败的
		t.Fail()

	}

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
