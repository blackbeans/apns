package lb

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/blackbeans/go-moa/protocol"
	log "github.com/blackbeans/log4go"
	"gopkg.in/redis.v3"
)

const (
	MK_LOOKUP       = "/service/lookup"
	MK_SERVICE_URI  = "/service/moa-admin"
	MK_REG_METHOD   = "registerService"
	MK_UNREG_METHOD = "unregisterService"
	MK_GET_METHOD   = "getService"
)

type momokeeper struct {
	regAddr      string
	lookupAddr   string
	regClient    *redis.Client
	lookupClient *redis.Client
	serviceUri   string
}

func NewMomokeeper(regAddr, lookupAddr string) *momokeeper {
	regClient := redis.NewClient(&redis.Options{
		Addr:        regAddr,
		Password:    "", // no password set
		DB:          0,  // use default DB
		DialTimeout: 30 * time.Second})

	lookupClient := redis.NewClient(&redis.Options{
		Addr:        lookupAddr,
		Password:    "", // no password set
		DB:          0,  // use default DB
		DialTimeout: 30 * time.Second})

	return &momokeeper{
		regAddr,
		lookupAddr,
		regClient,
		lookupClient,
		MK_SERVICE_URI}
}

type RegisterResp struct {
	Result     interface{} `json:"result"`
	ErrMessage string      `json:"em"`
	ErrCode    int         `json:"ec"`
}

func (self momokeeper) RegisteService(serviceUri, hostport, protoType string) bool {
	cmd := &protocol.MoaReqPacket{}
	cmd.ServiceUri = MK_SERVICE_URI
	cmd.Params.Method = MK_REG_METHOD
	args := make([]interface{}, 0, 3)
	args = append(args, serviceUri)
	args = append(args, hostport)
	args = append(args, protoType)
	args = append(args, make(map[string]interface{}, 0))
	cmd.Params.Args = args
	_, err := self.invokeResponse(self.regClient, MK_REG_METHOD, cmd)
	if nil == err {
		log.InfoLog("config_center", "momokeeper|RegisteService|SUCC|%s|%s|%s", hostport, serviceUri, protoType)
		return true
	} else {
		log.ErrorLog("config_center", "momokeeper|RegisteService|FAIL|%s|%s|%s|%s", err, hostport, serviceUri, protoType)
	}
	return false

}

func (self momokeeper) invokeResponse(c *redis.Client, method string, req *protocol.MoaReqPacket) (interface{}, error) {
	data, _ := json.Marshal(req)
	val, err := c.Get(string(data)).Result()

	if nil != err {
		log.ErrorLog("config_center", "momokeeper|%s|Get|FAIL|%s|%s", method, err, string(data))
		return "", err
	} else {
		var resp RegisterResp
		err = json.Unmarshal([]byte(val), &resp)
		if nil != err {
			log.ErrorLog("config_center", "momokeeper|%s|Unmarshal|FAIL|%s|%s", method, err, val)
			return "", err
		} else {
			if resp.ErrCode == 0 || resp.ErrCode == 200 {
				return resp.Result, nil
			} else {
				log.WarnLog("config_center", "momokeeper|%s|FAIL|%s|%s", method, resp, string(data))
				return "", errors.New(resp.ErrMessage)
			}
		}
	}
}

func (self momokeeper) UnRegisteService(serviceUri, hostport, protoType string) bool {
	cmd := &protocol.MoaReqPacket{}
	cmd.ServiceUri = MK_SERVICE_URI
	cmd.Params.Method = MK_UNREG_METHOD
	args := make([]interface{}, 0, 3)
	args = append(args, serviceUri)
	args = append(args, hostport)
	args = append(args, protoType)
	args = append(args, make(map[string]interface{}, 0))
	cmd.Params.Args = args
	_, err := self.invokeResponse(self.regClient, MK_UNREG_METHOD, cmd)
	if nil == err {
		log.InfoLog("config_center", "momokeeper|UnRegisteService|SUCC|%s|%s|%s", hostport, serviceUri, protoType)
		return true
	} else {
		log.ErrorLog("config_center", "momokeeper|UnRegisteService|FAIL|%s|%s|%s|%s", err, hostport, serviceUri, protoType)
	}
	return false

}

func (self momokeeper) GetService(serviceUri, protoType string) ([]string, error) {
	cmd := &protocol.MoaReqPacket{}
	cmd.ServiceUri = MK_LOOKUP
	cmd.Params.Method = MK_GET_METHOD
	args := make([]interface{}, 0, 3)
	args = append(args, serviceUri)
	args = append(args, protoType)
	cmd.Params.Args = args
	resp, err := self.invokeResponse(self.lookupClient, MK_GET_METHOD, cmd)

	if nil == err && nil != resp {
		result, asOk := resp.(map[string]interface{})
		if !asOk {
			log.WarnLog("config_center", "momokeeper|GetService|Assert|FAIL|%s|%s|%s", serviceUri, protoType, resp)
			return nil, err
		}

		hosts, ok := result["hosts"]
		if ok {
			uris, assert := hosts.([]interface{})
			if !assert {
				log.ErrorLog("config_center", "momokeeper|GetService|Hosts|FAIL|%s|%s|hosts:%s", serviceUri, protoType, hosts)
				return nil, err
			} else {
				arr := make([]string, 0, len(uris))
				for _, uri := range uris {
					arr = append(arr, uri.(string))
				}
				// log.InfoLog("config_center", "momokeeper|GetService|Hosts|SUCC|%s|%s|hosts:%s", serviceUri, protoType, uris)
				return arr, nil
			}
		} else {
			log.WarnLog("config_center", "momokeeper|GetService|NO Hosts|FAIL|%s|%s|hosts:%s", serviceUri, protoType, result)
		}
	}

	return nil, errors.New("No Hosts! " + serviceUri + "?protocol=" + protoType)

}

func (self momokeeper) Destroy() {

}
