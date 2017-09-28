package apns

import (
	"context"
	"fmt"
	"testing"
	"time"

	nproto "git.uneed.com/server/nimo/proto"
)

func TestApnsConn(t *testing.T) {

	certificate, _ := FromP12File("./push.p12", "xxxx")
	conn, err := NewApnsConn(context.Background(), certificate, URL_PRODUCTION, 20*time.Second)
	if nil != err {
		t.Errorf("NewApnsConn|FAIL|%v", err)
		t.FailNow()
	}

	for i := 0; i < 3; i++ {
		notify := &Notification{
			DeviceToken: "cb58e38a02b3cd438f2c00c23741953990b1a9cd6792900199cad4e129b520e5",
			Topic:       "com.blackbeans.apns",
			ApnsID:      nproto.NimoMessageid(),
			Payload: PayLoad{
				Aps: Aps{Alert: fmt.Sprintf("hello%d", i)}}}

		err := conn.SendMessage(notify)
		if nil != err {
			t.Logf("Recieve Push Response:%s\n", err)
			t.FailNow()
		}
		t.Logf("Recieve Push Response:%+v\n", notify.Response)
		time.Sleep(15 * time.Second)
	}

}

func TestConnPool(t *testing.T) {

	certificate, _ := FromP12File("./push.p12", "xxxx")
	pool, err := NewConnPool(10, 10, 10, 20*time.Second,
		func(ctx context.Context) (*ApnsConn, error) {
			conn, err := NewApnsConn(ctx, certificate, URL_PRODUCTION, 10*time.Second)
			if nil != err {
				t.Errorf("NewApnsConn|FAIL|%v", err)
				t.FailNow()
			}
			return conn, err
		})

	if nil != err {
		t.Errorf("Create ConnPool FAIL!")
		t.FailNow()
	}

	for i := 0; i < 3; i++ {
		notify := &Notification{
			DeviceToken: "cb58e38a02b3cd438f2c00c23741953990b1a9cd6792900199cad4e129b520e5",
			Topic:       "com.blackbeans.apns",
			ApnsID:      nproto.NimoMessageid(),
			Payload: PayLoad{
				Aps: Aps{Alert: fmt.Sprintf("hello%d", i)}}}

		c, err := pool.Get()
		if nil != err {
			t.Logf("Recieve Push Response:%s\n", err)
			t.FailNow()
		}
		err = c.SendMessage(notify)
		if nil != err {
			t.Logf("Recieve Push Response FAIL:%+v\n", notify.Response)
			t.FailNow()
			pool.ReleaseBroken(c)
		} else {
			pool.Release(c)
		}
	}

	pool.Shutdown()

}

//benchmark
func BenchmarkTestApns(b *testing.B) {

	b.StopTimer()

	certificate, _ := FromP12File("./push.p12", "xxx")
	pool, err := NewConnPool(10, 10, 10, 20*time.Second,
		func(ctx context.Context) (*ApnsConn, error) {
			conn, err := NewApnsConn(ctx, certificate, URL_PRODUCTION, 10*time.Second)
			if nil != err {
				b.Errorf("NewApnsConn|FAIL|%v", err)
				b.FailNow()
			}
			return conn, err
		})

	if nil != err {
		b.Errorf("Create ConnPool FAIL!")
		b.FailNow()
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		notify := &Notification{
			DeviceToken: "cb58e38a02b3cd438f2c00c23741953990b1a9cd6792900199cad4e129b520e5",
			Topic:       "com.blackbeans.apns",
			ApnsID:      nproto.NimoMessageid(),
			Payload: PayLoad{
				Aps: Aps{Alert: fmt.Sprintf("hello%d", i)}}}

		c, err := pool.Get()
		if nil != err {
			b.Logf("Recieve Push Response:%s\n", err)
			b.FailNow()
		}
		err = c.SendMessage(notify)
		if nil != err {
			b.Logf("Recieve Push Response FAIL:%+v\n", notify.Response)
			b.FailNow()
			pool.ReleaseBroken(c)
		} else {
			pool.Release(c)
		}
	}

	pool.Shutdown()
}
