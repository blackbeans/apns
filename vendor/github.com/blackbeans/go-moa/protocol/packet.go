package protocol

import (
	"encoding/json"
	"github.com/blackbeans/turbo/packet"
	"time"
)

type MoaReqPacket struct {
	ServiceUri string `json:"action"`
	Params     struct {
		Method string        `json:"m"`
		Args   []interface{} `json:"args"`
	} `json:"params"`
	Timeout time.Duration    `json:"-"`
	Channel chan interface{} `json:"-"`
}

//moa请求协议的包
type MoaRawReqPacket struct {
	ServiceUri string `json:"action"`
	Params     struct {
		Method string            `json:"m"`
		Args   []json.RawMessage `json:"args"`
	} `json:"params"`
	Timeout time.Duration    `json:"-"`
	Source  string           `json:"-"`
	Channel chan interface{} `json:"-"`
}

//moa响应packet
type MoaRespPacket struct {
	ErrCode int         `json:"ec"`
	Message string      `json:"em"`
	Result  interface{} `json:"result"`
}

//moa响应packet
type MoaRawRespPacket struct {
	ErrCode int             `json:"ec"`
	Message string          `json:"em"`
	Result  json.RawMessage `json:"result"`
}

func MoaRequest2Raw(req *MoaReqPacket) *MoaRawReqPacket {
	raw := &MoaRawReqPacket{}
	raw.ServiceUri = req.ServiceUri

	raw.Params.Method = req.Params.Method
	rawArgs := make([]json.RawMessage, 0, len(req.Params.Args))
	for _, a := range req.Params.Args {
		rw, _ := json.Marshal(a)
		rawArgs = append(rawArgs, json.RawMessage(rw))
	}

	raw.Params.Args = rawArgs
	raw.Channel = req.Channel
	raw.Timeout = req.Timeout
	return raw
}

func Wrap2MoaRawRequest(data []byte) (*MoaRawReqPacket, error) {
	var req MoaRawReqPacket
	err := json.Unmarshal(data, &req)
	if nil != err {
		return nil, err
	} else {
		// mrp := Command2MoaRequest(req)
		return &req, nil
	}

}

func Wrap2ResponsePacket(p *packet.Packet, resp interface{}) (*packet.Packet, error) {
	v, ok := resp.(string)
	var data []byte
	var err error = nil
	if ok {
		data = []byte(v)
	} else {
		data, err = json.Marshal(resp)
	}

	respPacket := packet.NewRespPacket(p.Header.Opaque, p.Header.CmdType, data)
	return respPacket, err
}

func MoaRsponse2Raw(resp *MoaRespPacket) *MoaRawRespPacket {
	raw := &MoaRawRespPacket{}
	raw.ErrCode = resp.ErrCode
	raw.Message = resp.Message
	rw, _ := json.Marshal(resp.Result)
	raw.Result = json.RawMessage(rw)
	return raw
}
