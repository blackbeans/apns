package apns

import (
	"go-apns/entry"
	"log"
	"time"
)

//接受错误的响应并触发重发
func (self *ApnsClient) onErrorResponseRecieve(responseChannel chan *entry.Response) {

	//获取响应并且触发重发操作
	var resp *entry.Response
	ch := make(chan *entry.Message, 100)
	for self.running {
		//顺序处理每一个连接的错误数据发送
		resp = <-responseChannel
		//只有 prcessing error 和 shutdown的两种id才会进行重发
		switch resp.Status {

		case entry.RESP_SHUTDOWN, entry.RESP_ERROR, entry.RESP_UNKNOW:
			//只有这三种才重发
			self.resend(ch, resp.Identifier, func(id uint32, msg *entry.Message) bool {
				expiredTime := int64(entry.UmarshalExpiredTime(msg))

				//过滤掉 不是当前连接ID的消息 或者 当前相同ID的消息 或者 (有过期时间结果已经过期的消息)
				return msg.ProcessId != resp.ProccessId ||
					id == resp.Identifier ||
					(0 != expiredTime && (time.Now().Unix()-expiredTime >= 0))

			})

		case entry.RESP_INVALID_TOKEN, entry.RESP_INVALID_TOKEN_SIZE:
			//将错误的token记录在存储中，备后续的过滤使用
			msg := self.storage.Get(resp.Identifier)
			//从msg中拿出token用于记录
			token := entry.UmarshalToken(msg)
			self.storeInvalidToken(token)
		}

	}
}

//重发逻辑
func (self *ApnsClient) resend(ch chan *entry.Message, id uint32,
	filter func(id uint32, msg *entry.Message) bool) {
	go func() {
		self.storage.Remove(id, 0, ch, filter)
	}()
	//获取需要重发的msg
	var msg *entry.Message
	for {
		msg = <-ch
		if nil == msg {
			break
		}
		//发送之......
		self.sendMessage(msg)
		log.Printf("APNSCLIENT|RESEND|%s\n", msg)
	}
}

func (self *ApnsClient) storeInvalidToken(token string) {
	//这里是里面最后存储不合法的token
	log.Printf("APNSCLIENT|UnImplement StoreInvalidToken|%s\n", token)
}
