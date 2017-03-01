package apns

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
)

const (
	ERROR_RESPONSE = 1 + 1 + 4
	FEEDBACK_RESP  = 4 + 2 + 32
)

//------------------feedback
type Feedback struct {
	Time        uint32
	DeviceToken string
}

func NewFeedBack(data []byte) *Feedback {
	feedback := &Feedback{}
	feedback.Unmarshal(data)

	return feedback
}

func (self *Feedback) Unmarshal(data []byte) {

	var tokenLength uint16
	tokenBuff := make([]byte, 32, 32)
	reader := bytes.NewReader(data)
	binary.Read(reader, binary.BigEndian, &self.Time)
	binary.Read(reader, binary.BigEndian, &tokenLength)
	binary.Read(reader, binary.BigEndian, &tokenBuff)

	self.DeviceToken = hex.EncodeToString(tokenBuff)
}

//-----------------error respons
type Response struct {
	Cmd          uint8
	Status       uint8
	Identifier   uint32
	ConnectionId int32
}

func (self *Response) Unmarshal(connectionId int32, data []byte) {
	reader := bytes.NewReader(data)
	binary.Read(reader, binary.BigEndian, &self.Cmd)
	binary.Read(reader, binary.BigEndian, &self.Status)
	binary.Read(reader, binary.BigEndian, &self.Identifier)
	self.ConnectionId = connectionId
}
