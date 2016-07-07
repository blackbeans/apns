package protocol

import (
	"bufio"
	"bytes"
	b "encoding/binary"
	// "errors"
	"fmt"
	log "github.com/blackbeans/log4go"
	"github.com/blackbeans/turbo/packet"
	"github.com/go-errors/errors"
	"strconv"
)

var BYTES_PING []byte
var BYTES_GET []byte
var BYTES_INFO []byte

func init() {

	BYTES_INFO = []byte("INFO")
	BYTES_PING = []byte("PING")
	BYTES_GET = []byte("GET")
}

const (
	PADDING = byte(0x00)
	//*1\r\n$4\r\nPING\r\n
	//*2\r\n$3\r\nGET\r\n${0}\r\n{"method":"","service-uri":""}\r\n
	//*1\r\n$4\r\nINFO\r\n
	GET  = byte(0x01)
	PING = byte(0x02)
	INFO = byte(0x03)
)

type RedisGetCodec struct {
	MaxFrameLength int
}

//读取数据
func (self RedisGetCodec) Read(reader *bufio.Reader) (*bytes.Buffer, error) {

	defer func() {
		if err := recover(); nil != err {
			er, ok := err.(*errors.Error)
			if ok {
				stack := er.ErrorStack()
				log.ErrorLog("moa-server", "RedisGetCodec|Read|ERROR|%s", stack)
			} else {
				log.ErrorLog("moa-server", "RedisGetCodec|Read|ERROR|%v", er)
			}
		}

	}()

	line, _, err := reader.ReadLine()
	if nil != err {
		return nil, err
	}

	if line[0] == '*' {
		//略过\r\n
		ac, _ := strconv.Atoi(string(line[1:]))
		//获取到共有多少个\r\n
		params := make([][]byte, 0, ac)
		for i := 0; i < ac; i++ {
			//去掉第一个字符'+'或者'*'或者'$'
			reader.Discard(1)
			//获取count
			length, err := self.getCount(reader)
			if nil != err {
				return nil, err
			}
			buff, err := self.readData(length, reader)
			if nil != err {
				return nil, err
			}
			params = append(params, buff)
		}

		cmdType := bytes.ToUpper(params[0][4 : len(params[0])-1])
		flag := PADDING
		if bytes.Equal(BYTES_PING, cmdType) {
			flag = PING
		} else if bytes.Equal(BYTES_GET, cmdType) {
			flag = GET
		} else if bytes.Equal(BYTES_INFO, cmdType) {
			flag = INFO
		}
		params[ac-1][len(params[ac-1])-1] = flag
		//得到了get和Ping数据将数据返回出去
		return bytes.NewBuffer(params[ac-1]), nil
	} else {
		return nil, errors.New("RedisGetCodec Error Packet Prototol Is Not Get " + string(line))
	}
}

func (self RedisGetCodec) getCount(reader *bufio.Reader) (int, error) {
	//获取count
	line, err := reader.ReadSlice('\n')
	if nil != err {
		return -1, errors.New("RedisGetCodec Read Command Len Packet Err " + err.Error())
	}
	end := bytes.IndexByte(line, '\r')
	//获取到数据的长度，读取数据
	length, _ := strconv.Atoi(string(line[:end]))
	if length <= 0 || length >= self.MaxFrameLength {
		return 0, errors.New(fmt.Sprintf("RedisGetCodec Err Packet Len %d/%d", length, self.MaxFrameLength))
	}
	return length, nil
}

func (self RedisGetCodec) readData(length int, reader *bufio.Reader) ([]byte, error) {
	//bodyLen+body+CommandType
	//4B+body+1B 是为了给将长度协议类型附加在dataBuff的末尾
	buff := make([]byte, 4+length+1)
	b.BigEndian.PutUint32(buff[0:4], uint32(length))
	dl := 0
	for {

		l, err := reader.Read(buff[dl+4 : 4+length])
		if nil != err {
			return buff, errors.New("RedisGetCodec Read Command Data Packet Err " + err.Error())
		}

		dl += l
		//如果超过了给定的长度则忽略
		if dl > length {
			return buff, errors.New(fmt.Sprintf("RedisGetCodec Invalid Packet Data readData:[%d/%d]\t%s ",
				dl, length, string(buff[4:dl])))
		} else if dl == length {
			//略过\r\n
			break
		}
	}
	reader.Discard(2)
	return buff, nil
}

//反序列化
//包装为packet，但是头部没有信息
func (self RedisGetCodec) UnmarshalPacket(buff *bytes.Buffer) (*packet.Packet, error) {

	buf := buff.Bytes()
	l := int(b.BigEndian.Uint32(buf[0:4]))
	data := buf[4 : 4+l]
	cmdType := buf[len(buf)-1]
	return packet.NewPacket(cmdType, data), nil
}

var ERROR = []byte("-Error message\r\n")

//序列化
//直接获取data
//GET $+n+\r\n+ [data]+\r\n
//PING +PONG
func (self RedisGetCodec) MarshalPacket(packet *packet.Packet) []byte {
	body := string(packet.Data)

	if packet.Header.CmdType == GET {
		lstr := strconv.Itoa(len(body))
		buff := bytes.NewBuffer(make([]byte, 0, 1+len(lstr)+2+len(body)+2))
		buff.WriteString("$")
		buff.WriteString(lstr)
		buff.WriteString("\r\n")
		buff.WriteString(body)
		buff.WriteString("\r\n")
		return buff.Bytes()
	} else if packet.Header.CmdType == PING || packet.Header.CmdType == INFO {
		buff := bytes.NewBuffer(make([]byte, 0, 1+len(body)))
		buff.WriteString("+")
		buff.WriteString(body)
		buff.WriteString("\r\n")
		return buff.Bytes()
	}

	return ERROR
}
