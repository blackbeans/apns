package apns

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestApnsPool(t *testing.T) {

	certificate, err := FromP12File("./cert/push.p12", "")
	if nil != err {
		t.Error(err)
		t.FailNow()
	}

	pool, err := NewConnPool(2, context.TODO(), func(ctx context.Context) (*ApnsConn, error) {
		apnsConn, err := NewApnsConn(context.TODO(), certificate, URL_PRODUCTION, 60*time.Second)
		return apnsConn, err
	})

	if nil != err {
		t.Error(err)
		t.FailNow()
	}

	conn, _ := pool.Get()

	pl := PayLoad{
		Aps: Aps{
			Sound: "calling_yuni.mp3",
			Badge: 1,
			//Alert: &Alert{
			//	Title: "与你消息",
			//	Body:  "收到了一条通话消息",
			//},
			ContentAvailable: 1},
	}

	notification := &Notification{
		ApnsID:      uuid.New().String(),
		DeviceToken: "",
		Topic:       "com.uneed.yuni",
		Payload:     pl,
		Expiration:  time.Now().Add(5 * time.Minute),
		ExtParams:   map[string]string{"call_type": "2"}}

	conn.SendMessage(notification)

	time.Sleep(1 * time.Minute)
}
