package entry

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"testing"
	"time"
)

const (
	sample = "645d2af69a2f491bb0c1a65064dedd4736fe710840bc38bf98d9892d34acae76"
)

func Test_FeedBackMarshal(t *testing.T) {
	now := uint32(time.Now().Unix())

	token, _ := hex.DecodeString(sample)
	t.Logf("token len :%d", len(token))
	buff := bytes.NewBuffer([]byte{})
	binary.Write(buff, binary.BigEndian, uint32(now))
	binary.Write(buff, binary.BigEndian, uint16(len(token)))
	binary.Write(buff, binary.BigEndian, token)

	data := buff.Bytes()
	t.Logf("--------%t ", data)

	feedback := &Feedback{}
	feedback.Unmarshal(data)

	t.Logf("--------%d,%s ", feedback.Time, feedback.DeviceToken)

	if feedback.Time != now || feedback.DeviceToken != sample {
		t.Fail()
	}
}

func Test_ResponseMarshal(t *testing.T) {
	now := uint32(time.Now().Unix())
	arr := make([]byte, 0, 256)
	buff := bytes.NewBuffer(arr)
	binary.Write(buff, binary.BigEndian, uint8(CMD_RESP_ERR))
	binary.Write(buff, binary.BigEndian, uint8(RESP_SUCC))
	binary.Write(buff, binary.BigEndian, uint32(now))

	data := buff.Bytes()
	t.Logf("--------%t,len:%d", data, len(data))

	resp := &Response{}
	resp.Unmarshal(data)

	t.Logf("--------%d,%d,%d", resp.Cmd, resp.Status, resp.Identifier)

	if resp.Cmd != CMD_RESP_ERR || resp.Status != RESP_SUCC || resp.Identifier != now {
		t.Fail()
	}
}

func TestWrapToken(t *testing.T) {
	token, _ := hex.DecodeString(sample)
	tokenStr := hex.EncodeToString(token)
	t.Logf("token=%s", tokenStr)
	if sample != tokenStr {
		t.Fail()
	}
}
