package entry

import (
	"bytes"
	"errors"
)

type Response struct {
	id         byte
	status     byte
	identifier int32
}

func DecodeResponse(data []byte) (Response, error) {
	if len(data) == 8 {
		reader := bytes.NewReader(data)
		op := reader.ReadByte()
		status := reader.ReadByte()
		identifier := int32(0) | (reader.ReadByte()<<32 | reader.ReadByte()<<24 | reader.ReadByte()<<16 | reader.ReadByte()<<8)
		return Response{id: op, status: status, identifier: identifier}, nil
	} else {
		return nil, errors.New("error response packet")
	}
}
