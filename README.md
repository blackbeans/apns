#### go实现的apns 提供https、redis协议方式的发送IOS Push服务
    单个连接的流控
    连接池的实现
    结合feedback过滤当前现有不合法的apnstoken提高送达率
    提供Https方式的发送ios push
    more.....

============
#### 安装
    go get github.com/blackbeans/go-apns
    go install github/com/blackbeans/go-apns

quick start
============

#### 加载证书初始化apns
    cert, err := tls.LoadX509KeyPair(CERT_PATH, KEY_PATH)
    if nil != err {
    t.Logf("READ CERT FAIL|%s", err.Error())
		  t.Fail()
		  return
		}

	responseChan := make(chan entry.Response, 10)
	feedbackChan := make(chan entry.Feedback, 1000)

	body := "hello apns"
	payload := entry.NewSimplePayLoad("ms.caf", 1, body)
	client := NewDefaultApnsClient(cert, responseChan, PUSH_APPLE, feedbackChan, FEED_BACK_APPLE)
	
	
#####开始发送
	//发送enchanced push
	client.SendEnhancedNotification(1, math.MaxUint32, apnsToken, *payload)
	//调用读取feedback
	err := client.FetchFeedback()
	
##### 读取feedback和response结果

	//从channel 中读取数据
	select {
		case resp := <-responseChan:
			t.Logf("RESPONSE===============%t", resp)
			//如果有返回错误则说明发送失败的
		case fb := <-feedbackChan:
			i := 0
			for i < 100 {
				t.Logf("FEEDBACK===============%s", fb)
				i++
			}
	}
	
	

