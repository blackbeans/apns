package apns

import (
	log "github.com/blackbeans/log4go"
	"go-apns/entry"
	"math/rand"
	"time"
)

//接受错误的响应并触发重发
func (self *ApnsClient) onErrorResponseRecieve(responseChannel chan *entry.Response) {

	resendCh := make(chan *entry.Message, 10000)
	tokenCh := make(chan string, 1000)
	//启动重发任务
	go self.resend(resendCh)
	go self.storeInvalidToken(tokenCh)

	//开始启动
	for self.running {
		//顺序处理每一个连接的错误数据发送
		resp := <-responseChannel
		//只有 prcessing error 和 shutdown的两种id才会进行重发
		switch resp.Status {

		case entry.RESP_SHUTDOWN, entry.RESP_ERROR, entry.RESP_UNKNOW, entry.RESP_INVALID_TOKEN, entry.RESP_INVALID_TOKEN_SIZE:

			//只有这三种才重发
			ch := make(chan *entry.Message, 100)
			go self.storage.Remove(resp.Identifier, 0, ch, func(id uint32, msg *entry.Message) bool {
				expiredTime := int64(entry.UmarshalExpiredTime(msg))

				//过滤掉 不是当前连接ID的消息 或者 当前相同ID的消息 或者 (有过期时间结果已经过期的消息)
				return msg.ConnectionId != resp.ConnectionId ||
					id == resp.Identifier ||
					(0 != expiredTime && (time.Now().Unix()-expiredTime >= 0))
			})

			for {

				tmp, ok := <-ch
				//如果删除成功并且消息不为空则重发
				if nil != tmp {
					resendCh <- tmp
				} else if !ok {
					break
				}
			}

			log.InfoLog("push_client", "APNSCLIENT|onErrorResponseRecieve|ERROR|%d", resp.Status)

			//非法的token，那么就存储起来
			switch resp.Status {
			case entry.RESP_INVALID_TOKEN, entry.RESP_INVALID_TOKEN_SIZE:
				//将错误的token记录在存储中，备后续的过滤使用
				msg := self.storage.Get(resp.Identifier)
				if nil != msg {
					//从msg中拿出token用于记录
					token := entry.UmarshalToken(msg)
					tokenCh <- token
					// self.storeInvalidToken(token)
					log.WarnLog("push_client", "APNSCLIENT|INVALID TOKEN|%s", resp.Identifier)
				}
			}
		}

	}
}

//重发逻辑
func (self *ApnsClient) resend(ch chan *entry.Message) {

	for self.running {
		select {
		case <-time.After(5 * time.Second):
		case msg := <-ch:
			//发送之......
			msg.IdentifierId = 0
			self.sendMessage(msg)
			self.resendCounter.Incr(1)
			if rand.Intn(100) == 0 {
				log.InfoLog("push_client", "APNSCLIENT|RESEND|%s\n", msg)
			}
		}
	}

}

func (self *ApnsClient) storeInvalidToken(ch chan string) {
	batch := make([]string, 0, 10)
	for self.running {
		var token string
		select {
		case token = <-ch:
		case <-time.After(5 * time.Second):
		}

		func() {
			if nil != self.tokenStorage {
				defer func() {
					if err := recover(); nil != err {
					}
				}()
				//达到最大的batch数量或者token为空说明超时了提交
				if len(batch) >= cap(batch) || len(token) <= 0 {
					self.tokenStorage.Save(batch)
					//这里是里面最后存储不合法的token
					log.WarnLog("push_client", "APNSCLIENT|UnImplement StoreInvalidToken|%v", batch)
					batch = batch[:0]
				}

				if len(token) > 0 {
					batch = append(batch, token)
				}
			}
		}()

	}
}
