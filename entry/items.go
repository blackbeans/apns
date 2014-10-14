package entry

import (
	"bytes"
	"encoding/json"
	"log"
)

type Alert struct {
	Body         string        `json:"body"`
	ActionLocKey string        `json:"action-loc-key"`
	LocKey       string        `json:"loc-key"`
	LocArgs      []interface{} `json:"loc-args"`
}

type Aps struct {
	Alert Alert  `json:"alert"` //提醒的内容
	Badge string `json:"badge"` //显示气泡数
	Sound string `json:"sound"` //控制push弹出的声音
}

type PayLoad struct {
	IData
	aps       Aps
	extParams map[string]interface{} //扩充字段
}

func NewPayLoad(sound, badge string, alert Alert) *PayLoad {
	aps := Aps{Alert: alert, Sound: sound, Badge: badge}
	return &PayLoad{aps: aps, extParams: make(map[string]interface{})}
}

func (self *PayLoad) addExtParam(key string, val interface{}) *PayLoad {
	self.extParams[key] = val
	return self
}

func (self *PayLoad) Marshal() []byte {

	encoddata := make(map[string]interface{}, 2)
	encoddata["aps"] = self.aps
	for k, v := range self.extParams {
		encoddata[k] = v
	}

	data, err := json.Marshal(encoddata)
	if nil != err {
		log.Println("encode payload fail !")
		return nil
	}
	return buffer.Bytes()
}

func WrapPayLoad(payload *PayLoad) *Item {
	return &Item{id: PAY_LOAD, data: payload.Marshal()}
}

func WrapDeviceToken(token string) *Item {
	data := make([]byte, 0, 32)
	bytes.NewBuffer(data)
	return &Item{id: DEVICE_TOKEN, length: 32, data: []byte(token)}
}

func WrapNotifyIdentifier(id int32) *Item {
	return &Item{id: NOTIFY_IDENTIYFIER, lenght: 4, data: int32Tobytes(id)}
}

func int32Tobytes(num int32) []byte {
	data := [4]byte{}
	data[0] = (num >> 24) & 0xFF
	data[1] = (num >> 16) & 0xFF
	data[2] = (num >> 8) & 0xFF
	data[3] = (num) & 0xFF
	return data
}

func WrapExpirationDate(expirateDate int32) *Item {
	return &Item{id: EXPIRATED_DATE, length: 4, data: int32Tobytes(expirateDate)}
}

func WrapPriority(priority byte) *Item {
	return &Item{id: PRIORITY, length: 1, data: [1]byte{priority}}
}
