package entry

import (
	"bytes"
)

/*
 * 发送的一个数据包
 */
type IData interface {
	Marshal() []byte
}

//用于发送的push的Message
type Message struct {
	op     byte
	length int32
	items  []Item
}

func NewMessage() *Message {
	msg := &Message{op: CMD_POP}
	msg.items = make([]item, 0, 2)
	return msg
}

func (self *Message) AddItem(items ...*Item) {
	self.items = append(self.items, items)
}

func (self *Message) Encode() []byte {
	frame := make([]byte)
	buff := bytes.NewBuffer(frame)
	//frame 的command类型 2
	buff.WriteByte(self.op)

	dbuffer := bytes.NewBuffer(data)
	for _, v := range self.items {
		dbuffer.WriteByte(v.id & 0xFF)
		dbuffer.Write(v.length & 0xFFFF)
		dbuffer.Write(v.data)
	}
	//frame length  4 bytes
	buff.Write(dbuffer.Len() & 0xFFFFFFFF)
	//frame的data body体
	buff.Write(dbuffer.Bytes())

	return buff.Bytes()

}

//发送的item
//包含：device-token/payload/notification/expireation/priority
type Item struct {
	id     byte
	length int
	data   []byte
}
