package main

import (
	"crypto/tls"
	"fmt"
	"go-apns/apns"
	"go-apns/entry"
)

const (
	CERT_PATH       = "/Users/blackbeans/pushcert.pem"
	KEY_PATH        = "/Users/blackbeans/key.pem"
	PUSH_APPLE      = "gateway.push.apple.com:2195"
	FEED_BACK_APPLE = "feedback.push.apple.com:2196"
	apnsToken       = "f232e31293b0d63ba886787950eb912168f182e6c91bc6bdf39d162bf5d7697d"
)

func main() {
	feedbackChan := make(chan *entry.Feedback, 1000)
	cert, _ := tls.LoadX509KeyPair(CERT_PATH, KEY_PATH)
	client := apns.NewMockApnsClient(cert, PUSH_APPLE, feedbackChan, FEED_BACK_APPLE, entry.NewCycleLink(3, 100000))

	payload := entry.NewSimplePayLoad("ms.caf", int(1), "hello")
	for i := 0; i < 100; i++ {
		go func(a int) {
			for j := 0; j < 1000000; j++ {
				client.SendEnhancedNotification(1, 1, apnsToken, *payload)
			}
			fmt.Printf("finish:%d\n", a)
		}(i)
	}

	select {}
}
