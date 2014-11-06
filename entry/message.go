package entry

import (
	"bytes"
	"encoding/binary"
	"log"
	"reflect"
)

//用于发送的push的Message
type Message struct {
	op     byte
	length int32
	items  []*Item
	ttl    uint8 //存活次数
}

func NewMessage(op byte, ttl uint8) *Message {
	msg := &Message{op: op, ttl: ttl}
	msg.items = make([]*Item, 0, 2)
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

//发送的item
//包含：device-token/payload/notification/expireation/priority
type Item struct {
	id     uint8
	length uint16
	data   interface{}
}
