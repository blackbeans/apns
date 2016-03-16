package server

import (
	"encoding/json"
	"fmt"
	log "github.com/blackbeans/log4go"
	"github.com/go-errors/errors"
	"go-apns/apns"
	"go-apns/entry"
	"go-apns/server"
	"net/http"
	"strconv"
	"sync"
)

type response struct {
	Status int         `json:"status,omitempty"`
	Error  error       `json:"error,omitempty"`
	Body   interface{} `json:"body,omitempty"` //只有在response的时候才会有
}

func (self *response) Marshal() []byte {
	data, err := json.Marshal(self)
	if nil != err {
		//就是数据哦有问题了
		return nil
	} else {
		return data
	}
}

type ApnsHttpServer struct {
	feedbackChan chan *entry.Feedback //用于接收feedback的chan
	apnsClient   *apns.ApnsClient
	mutex        sync.Mutex
	expiredTime  uint32
	httpserver   *MomoHttpServer
}

func NewApnsHttpServer(logxml string, option server.Option,
	feedbackChan chan *entry.Feedback,
	apnsClient *apns.ApnsClient) *ApnsHttpServer {

	//加载log4go的配置
	log.LoadConfiguration(logxml)

	server := &ApnsHttpServer{feedbackChan: feedbackChan,
		apnsClient: apnsClient, expiredTime: option.ExpiredTime}

	//创建http
	server.httpserver = NewMomoHttpServer(option.BindAddr, nil)

	go server.dial(option.BindAddr)

	return server
}

func (self *ApnsHttpServer) dial(hp string) {

	log.InfoLog("push_handler", "APNS HTTPSERVER IS STARTING ....")
	http.HandleFunc("/apns/push", self.handlePush)
	http.HandleFunc("/apns/feedback", self.handleFeedBack)
	http.HandleFunc("/apns/stat", self.handleStat)

	err := self.httpserver.ListenAndServe()
	if nil != err {
		log.ErrorLog("push_handler", "APNSHTTPSERVER|LISTEN|FAIL|%s", err)
	} else {
		log.InfoLog("push_handler", "APNSHTTPSERVER|LISTEN|SUCC|%s .....", hp)
	}

}

func (self *ApnsHttpServer) Shutdown() {
	self.httpserver.Shutdonw()
	self.apnsClient.Destory()
	log.InfoLog("push_handler", "APNS HTTP SERVER SHUTDOWN SUCC ....")

}

func (self *ApnsHttpServer) handleStat(out http.ResponseWriter, req *http.Request) {

	//获取状态
	status := self.apnsClient.Monitor()
	jsonData, _ := json.Marshal(status)
	out.Header().Set("content-type", "text/json")
	out.Write(jsonData)
}

func (self *ApnsHttpServer) handleFeedBack(out http.ResponseWriter, req *http.Request) {
	response := &response{}
	response.Status = server.RESP_STATUS_SUCC

	if req.Method == "GET" {
		//本次获取多少个feedback
		limitV := req.FormValue("limit")
		limit, _ := strconv.ParseInt(limitV, 10, 32)
		if limit > 100 {
			response.Status = server.RESP_STATUS_FETCH_FEEDBACK_OVER_LIMIT_ERROR
			response.Error = errors.New("Fetch Feedback Over limit 100 ")
		} else {
			//发起了获取feedback的请求
			err := self.apnsClient.FetchFeedback(int(limit))
			if nil != err {
				response.Error = err
				response.Status = server.RESP_STATUS_ERROR
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
		response.Status = server.RESP_STATUS_INVALID_PROTO
		response.Error = errors.New("Unsupport Post method Invoke!")
	}

	self.write(out, response)
}

//处理push
func (self *ApnsHttpServer) handlePush(out http.ResponseWriter, req *http.Request) {

	resp := &response{}
	resp.Status = server.RESP_STATUS_SUCC
	if req.Method == "GET" {
		//返回不支持的请求方式
		resp.Status = server.RESP_STATUS_INVALID_PROTO
		resp.Error = errors.New("Unsupport Get method Invoke!")

	} else if req.Method == "POST" {

		//pushType
		pushType := req.PostFormValue("pt") //先默认采用Enhanced方式
		//接卸对应的token和payload
		token, payload := self.decodePayload(req, resp)

		trace := req.PostFormValue("trace")

		expiredSeconds := req.PostFormValue("expiredSeconds")
		expiredTime := self.expiredTime
		if len(expiredSeconds) > 0 {
			t, err := strconv.ParseInt(expiredSeconds, 10, 32)
			if nil == err {
				expiredTime = uint32(t)
			}
		}

		//----------------如果依然是成功状态则证明当前可以发送
		if server.RESP_STATUS_SUCC == resp.Status {

			func() {
				defer func() {
					if re := recover(); nil != re {
						stack := re.(*errors.Error).ErrorStack()
						log.ErrorLog("push_handler", "ApnsHttpServer|handlePush|SEND|FAIL|%s|%s|%s", stack, payload, trace)
						resp.Status = server.RESP_STATUS_ERROR
						resp.Error = errors.New(fmt.Sprintf("%s", re))
						self.write(out, resp)
					}
				}()
				self.innerSend(pushType, token, payload, resp, expiredTime)
				self.write(out, resp)
				log.DebugLog("push_handler", "ApnsHttpServer|handlePush|SUCC|%s|%s|%s", resp, payload, trace)
			}()

		} else {
			log.WarnLog("push_handler", "ApnsHttpServer|handlePush|FAIL|%s|%s|%s", resp, payload, trace)
			self.write(out, resp)
		}

	}

}

func (self *ApnsHttpServer) write(out http.ResponseWriter, resp *response) {
	out.Header().Set("content-type", "text/json")
	out.Write(resp.Marshal())
}
