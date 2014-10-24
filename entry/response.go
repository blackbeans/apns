package entry

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
)

const (
	ERROR_RESPONSE = 1 + 1 + 4
)

//------------------feedback
type Feedback struct {
	Time        uint32
	DeviceToken string
}

func (self *Feedback) Unmarshal(data []byte) {

	var tokenLength uint16
	tokenBuff := make([]byte, 32, 32)
	reader := bytes.NewReader(data)
	binary.Read(reader, binary.BigEndian, &self.Time)
	binary.Read(reader, binary.BigEndian, &tokenLength)
	binary.Read(reader, binary.BigEndian, &tokenBuff)

	self.DeviceToken = hex.EncodeToString(tokenBuff[0:tokenLength])
}

//-----------------error respons
type Response struct {
	Cmd        uint8
	Status     uint8
	Identifier uint32
}

func (self *Response) Unmarshal(data []byte) {
	reader := bytes.NewReader(data)
	binary.Read(reader, binary.BigEndian, &self.Cmd)
	binary.Read(reader, binary.BigEndian, &self.Status)
	binary.Read(reader, binary.BigEndian, &self.Identifier)
}
