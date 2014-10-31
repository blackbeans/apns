package server

import (
	"encoding/json"
	"errors"
	"go-apns/apns"
	"go-apns/entry"
	"log"
	"net/http"
	"reflect"
	"strconv"
)

type ApnsHttpServer struct {
	responseChan chan *entry.Response // 响应地channel
	feedbackChan chan *entry.Feedback //用于接收feedback的chan
	apnsClient   *apns.ApnsClient
	pushId       uint32
	expiredTime  uint32
}

func NewApnsHttpServer(option Option) *ApnsHttpServer {

	responseChan := make(chan *entry.Response, 100)
	feedbackChan := make(chan *entry.Feedback, 1000)

	//初始化apns
	apnsClient := apns.NewDefaultApnsClient(option.cert, chan<- *entry.Response(responseChan),
		option.pushAddr, chan<- *entry.Feedback(feedbackChan), option.feedbackAddr)

	server := &ApnsHttpServer{responseChan: responseChan, feedbackChan: feedbackChan,
		apnsClient: apnsClient, expiredTime: option.expiredTime}

	http.HandleFunc("/apns/push", server.handlePush)
	http.HandleFunc("/apns/feedback", server.handleFeedBack)
	go func() {
		err := http.ListenAndServe(option.bindAddr, nil)
		if nil != err {
			log.Panicf("APNSHTTPSERVER|LISTEN|FAIL|%s\n", err)
		} else {
			log.Panicf("APNSHTTPSERVER|LISTEN|SUCC|%s .....\n", option.bindAddr)
		}
	}()

	//读取一下响应结果
	// resp := <-responseChan
	return server
}

func (self *ApnsHttpServer) Shutdown() {
	self.apnsClient.Destory()
	log.Println("APNS HTTP SERVER SHUTDOWN SUCC ....")

}

func (self *ApnsHttpServer) handleFeedBack(resp http.ResponseWriter, req *http.Request) {
	response := response{}
	response.Status = RESP_STATUS_SUCC

	if req.Method == "GET" {
		//本次获取多少个feedback
		limitV := req.FormValue("limit")
		limit, _ := strconv.ParseInt(limitV, 10, 32)
		if limit > 100 {
			response.Status = RESP_STATUS_FETCH_FEEDBACK_OVER_LIMIT_ERROR
			response.Error = errors.New("Fetch Feedback Over limit 100 ")
		} else {
			//发起了获取feedback的请求
			err := self.apnsClient.FetchFeedback()
			if nil != err {
				response.Error = err
				response.Status = RESP_STATUS_ERROR
			} else {
				//等待feedback数据
				packet := make([]*entry.Feedback, 0, limit)
				var feedback *entry.Feedback
				for ; limit > 0; limit-- {
					feedback = <-self.feedbackChan
					if nil == feedback {
						break
					}
					packet = append(packet, feedback)
				}
				response.Body = packet
			}
		}
	} else {
		response.Status = RESP_STATUS_INVALID_PROTO
		response.Error = errors.New("Unsupport Post method Invoke!")
	}

	self.write(resp, response)
}

func (self *ApnsHttpServer) write(out http.ResponseWriter, resp response) {
	out.Header().Set("content-type", "text/json")
	out.Write(resp.Marshal())
}

//处理push
func (self *ApnsHttpServer) handlePush(resp http.ResponseWriter, req *http.Request) {

	response := response{}
	response.Status = RESP_STATUS_SUCC

	if req.Method == "GET" {
		//返回不支持的请求方式
		response.Status = RESP_STATUS_INVALID_PROTO
		response.Error = errors.New("Unsupport Get method Invoke!")

	} else if req.Method == "POST" {

		//pushType
		// pushType := req.PostFormValue("pushType") //先默认采用Enhanced方式

		token := req.PostFormValue("token")

		sound := req.PostFormValue("sound")

		badgeV := req.PostFormValue("badge")
		badge, _ := strconv.ParseInt(badgeV, 10, 32)

		body := req.PostFormValue("body")

		//是个大的Json数据即可
		extArgs := req.PostFormValue("extArgs")

		//拼接payload
		payload := *entry.NewSimplePayLoad(sound, int(badge), body)

		if len(extArgs) > 0 {
			var jsonMap map[string]interface{}
			err := json.Unmarshal([]byte(extArgs), &jsonMap)
			if nil != err {
				response.Status = RESP_STATUS_PAYLOAD_BODY_DECODE_ERROR
				response.Error = errors.New("PAYLOAD BODY DECODE ERROR!")
			} else {

				for k, v := range jsonMap {
					//如果存在数据嵌套则返回错误，不允许数据多层嵌套
					if reflect.TypeOf(v).Kind() == reflect.Map {
						response.Status = RESP_STATUS_PAYLOAD_BODY_DEEP_ITERATOR
						response.Error = errors.New("DEEP PAYLOAD BODY ITERATOR!")
						break
					} else {
						payload.AddExtParam(k, v)
					}
				}
			}
		}

		if RESP_STATUS_SUCC == response.Status {

			//能直接放在chan中异步发送
			var err error
			//如果有异常则重试发送
			for i := 0; i < 3; i++ {
				log.Println(payload)
				err = self.apnsClient.SendEnhancedNotification(self.pushId, self.expiredTime, token, payload)
				if nil == err {
					self.pushId++
					break
				}
			}

			if nil != err {
				log.Printf("APNS_HTTP_SERVER|SendEnhancedNotification|FAIL|IGNORED|%s|%s\n", payload, err)
				response.Status = RESP_STATUS_SEND_OVER_TRY_ERROR
				response.Error = err
			}
		}

	}

	self.write(resp, response)

}
