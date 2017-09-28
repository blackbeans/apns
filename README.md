
apns is an apple apns libary for http2

####  feature:

* support connection pool 

* support ping frame using check connection alive

============
#### install

```shell

	go get golang.org/x/net/http2

	go get github.com/blackbeans/apns

```

quick start
============

#### create  apns client

 ```golang   
	
	certificate, _ := FromP12File("./push.p12", "xxxx")

	pool, err := NewConnPool(10, 10, 10, 20*time.Second,
		func(ctx context.Context) (*ApnsConn, error) {
			conn, err := NewApnsConn(ctx, certificate, URL_PRODUCTION, 10*time.Second)
			if nil != err {
				log.Printf("Create Apns Conn Fail %v\n",err)
			}
			return conn, err
	})

	//build notification 

	notify := &Notification{
			DeviceToken: "your device token",
			Topic:       "bundleid ",
			ApnsID:      "uuid",
			Payload: PayLoad{
				Aps: Aps{Alert: fmt.Sprintf("hello%d", i)}}}
	
	//send push

	c,err:= pool.Get()
	if nil!=err{
		//
	}

	//note : release connection
	defer pool.Release(c)

	err =c.SendMessage(notify)
	
	if nil!=err{
		//encounter error
	}else{
		if notify.Response.Status != 200{
			//may send error
			log.Printf("Response Err %s",notify.Response.Reason)

			//maybe u need resent
		}
	}

```





