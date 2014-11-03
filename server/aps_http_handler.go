package server

import (
	"encoding/json"
	"errors"
	"go-apns/entry"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"sync/atomic"
)

var regx *regexp.Regexp

func init() {
	regx, _ = regexp.Compile("\\w+")
}

func (self *ApnsHttpServer) decodePayload(req *http.Request, resp *response) (string, *entry.PayLoad) {

	tokenV := req.PostFormValue("token")
	sound := req.PostFormValue("sound")
	badgeV := req.PostFormValue("badge")
	body := req.PostFormValue("body")

	//-----------检查参数
	valid := checkArguments(tokenV, sound, badgeV, body)
	if !valid {
		resp.Status = RESP_STATUS_PUSH_ARGUMENTS_INVALID
		resp.Error = errors.New("Notification Params are Invalid!")
		return "", nil
	}

	tokenSplit := regx.FindAllString(tokenV, -1)
	var token string = ""
	for _, v := range tokenSplit {
		token += v
	}

	badge, _ := strconv.ParseInt(badgeV, 10, 32)
	//拼接payload
	payload := entry.NewSimplePayLoad(sound, int(badge), body)

	//是个大的Json数据即可
	extArgs := req.PostFormValue("extArgs")
	if len(extArgs) > 0 {
		var jsonMap map[string]interface{}
		err := json.Unmarshal([]byte(extArgs), &jsonMap)
		if nil != err {
			resp.Status = RESP_STATUS_PAYLOAD_BODY_DECODE_ERROR
			resp.Error = errors.New("PAYLOAD BODY DECODE ERROR!")
		} else {
			for k, v := range jsonMap {
				//如果存在数据嵌套则返回错误，不允许数据多层嵌套
				if reflect.TypeOf(v).Kind() == reflect.Map {
					resp.Status = RESP_STATUS_PAYLOAD_BODY_DEEP_ITERATOR
					resp.Error = errors.New("DEEP PAYLOAD BODY ITERATOR!")
					break
				} else {
					payload.AddExtParam(k, v)
				}
			}
		}
	}

	return token, payload
}

//内部发送代码
func (self *ApnsHttpServer) innerSend(pushType string, token string, payload *entry.PayLoad, resp *response) {

	var sendFunc func(err error) error
	if NOTIFY_SIMPLE_FORMAT == pushType {
		//如果为简单
		sendFunc = func(err error) error {
			return self.apnsClient.SendEnhancedNotification(self.identifierId(err),
				self.expiredTime, token, *payload)
		}
	} else if NOTIFY_ENHANCED_FORMAT == pushType {
		//如果为扩展的
		sendFunc = func(err error) error {
			return self.apnsClient.SendSimpleNotification(token, *payload)
		}
	} else {
		resp.Status = RESP_STATUS_INVALID_NOTIFY_FORMAT
		resp.Error = errors.New("Invalid notification format " + pushType)
	}

	//能直接放在chan中异步发送
	var err error
	//如果有异常则重试发送
	for i := 0; i < 3 && (RESP_STATUS_SUCC == resp.Status); i++ {
		err = sendFunc(err)
		if nil == err {
			break
		}
	}
	if nil != err {
		log.Printf("APNS_HTTP_SERVER|SendNotification|FORMATE:%d|FAIL|IGNORED|%s|%s\n", pushType, payload, err)
		resp.Status = RESP_STATUS_SEND_OVER_TRY_ERROR
		resp.Error = err
	}
}

func checkArguments(args ...string) bool {
	for _, v := range args {
		if len(v) <= 0 {
			return false
		}
	}

	return true
}

func (self *ApnsHttpServer) identifierId(err error) uint32 {
	if nil == err {
		self.pushId = atomic.AddUint32(&self.pushId, 1)
	}
	return self.pushId
}
