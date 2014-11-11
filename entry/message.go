package entry

import (
	"bytes"
	"encoding/binary"
	"log"
	"reflect"
)

const (
	MESSAGE_TYPE_SIMPLE   = byte(0) //简单的消息类型
	MESSAGE_TYPE_ENHANCED = byte(1) //复杂的消息类型
)

//用于发送的push的Message
type Message struct {
	op        byte
	length    int32
	items     []*Item
	ttl       uint8 //存活次数
	MsgType   byte
	ProcessId int32 //被处理的Id 是不可变的
}

func NewMessage(op byte, ttl uint8, msgType byte) *Message {
	msg := &Message{op: op, ttl: ttl}
	msg.items = make([]*Item, 0, 2)
	msg.MsgType = msgType
	return msg
}

func (self *Message) AddItem(items ...*Item) {
	self.items = append(self.items, items...)
}

func (self *Message) Encode() (error, []byte) {

	framebuff := new(bytes.Buffer)
	//write item body
	for _, v := range self.items {
		//如果是采用tlv形式的字节编码则写入类型、长度
		datat := reflect.TypeOf(v.data).Kind()
		if datat != reflect.Uint8 && datat != reflect.Uint16 &&
			datat != reflect.Uint32 && datat != reflect.Uint64 {
			binary.Write(framebuff, binary.BigEndian, v.length)
		}

		err := binary.Write(framebuff, binary.BigEndian, v.data)
		if nil != err {
			log.Printf("MESSAGE|ENCODE|FAIL|%s|%s", err.Error(), v)
			return err, nil
		}
	}

	buff := make([]byte, 0, 1+framebuff.Len())
	bytebuff := bytes.NewBuffer(buff)
	//frame 的command类型
	binary.Write(bytebuff, binary.BigEndian, uint8(self.op))
	//frame body
	binary.Write(bytebuff, binary.BigEndian, framebuff.Bytes())
	return nil, bytebuff.Bytes()

}

//从message重umarshaltoken
func UmarshalIdentifier(msg *Message) uint32 {
	if msg.MsgType == MESSAGE_TYPE_ENHANCED {
		//enchanced 的token位于第三个item
		id := msg.items[0]
		return id.data.(uint32)

	}

	//this is a bug
	return 0
}

//从message重umarshaltoken
func UmarshalToken(msg *Message) string {
	if msg.MsgType == MESSAGE_TYPE_ENHANCED {
		//enchanced 的token位于第三个item
		tokenItem := msg.items[2]
		return tokenItem.data.(string)

	} else if msg.MsgType == MESSAGE_TYPE_SIMPLE {
		//simple类型的token位于第一个item
		tokenItem := msg.items[0]
		return tokenItem.data.(string)
	}
	//this is a bug
	return ""
}

//发送的item
//包含：device-token/payload/notification/expireation/priority
type Item struct {
	id     uint8
	length uint16
	data   interface{}
}
