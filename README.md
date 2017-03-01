
go-apns is apple apns libary providing redis and http protocol to use 

####  feature:

connection pool 
    
[go-moa](https://github.com/blackbeans/go-apns) interface
    
http protocol interface
    
support Invalid token filter
    
message resend 

============
#### install

quick start
============

#### create  apns client

 ```golang   
    apnsConf := apns.Config{}
    apnsOption := apns.NewApnsOption(apnsConf)

	feedback := make(chan *apns.Feedback, 1000)
	//初始化apns
	apnsClient := apns.NewDefaultApnsClient(apnsOption.Cert,
		apnsOption.PushAddr, chan<- *apns.Feedback(feedback),
		apnsOption.FeedbackAddr,
		apns.NewCycleLink(3, apnsOption.StorageCapacity))
```
	
#### build Payload 

```golang 

    aps := apns.Aps{}
	aps.Sound = 
	aps.Badge = 
	aps.Alert = 
	
	//payload
	payload := apns.NewSimplePayLoadWithAps(aps)

``` 

##### Try Send Push

```golang

	// send enchanced push
	apnsClient.SendEnhancedNotification(1, math.MaxUint32, apnsToken, *payload)

```
	


#### Donate

![image](https://github.com/blackbeans/kiteq/blob/master/doc/qcode.png)

#### Contact us 

Mail: blackbeans.zc@gmail.com

QQ: 136448723






