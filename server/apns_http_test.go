package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

const (
	CERT_PATH       = "/Users/blackbeans/workspace/github/go-apns/pushcert.pem"
	KEY_PATH        = "/Users/blackbeans/workspace/github/go-apns/key.pem"
	PUSH_APPLE      = "gateway.push.apple.com:2195"
	FEED_BACK_APPLE = "feedback.push.apple.com:2196"
	apnsToken       = "b8cca5b914195a3441e09b69adc8e26c228e43d85be4f1d810481fca95b53d88"
	PROXY_URL       = "http://localhost:17070"
)

func TestApnsHttpServer(t *testing.T) {
	option := NewOption(":17070", CERT_PATH, KEY_PATH, RUNMODE_ONLINE)
	option.expiredTime = uint32(6 * 3600)
	server := NewApnsHttpServer(option)

	//测试发送
	innerApsnHttpServerSend(t)

	//测试获取feedback
	innerApsnHttpServerFeedback(t)

	defer server.Shutdown()
}

func innerApsnHttpServerSend(t *testing.T) {

	fmt.Println("innerApsnHttpServerSend is Starting")

	data := make(url.Values)
	data.Set("token", apnsToken)
	data.Set("sound", "ms.caf")
	data.Set("badge", "10")
	data.Set("body", "HTTP APNS SERVER TEST! ")
	data.Set("extArgs", "{\"name\":\"blackbeans\"}")

	//然后发起调用
	resp, err := http.PostForm(PROXY_URL+"/apns/push", data)
	if nil != err {
		t.Logf("HTTP POST PUSH FAIL!%s\n", err)
		t.Fail()
		return
	}

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if nil != err {
		fmt.Printf("HTTP READ RESPONSE FAIL !|%s", err)
		t.Fail()
		return
	}

	defer resp.Body.Close()

	var response response
	err = json.Unmarshal(body, &response)
	if nil != err {
		fmt.Printf("HTTP Unmarshal RESPONSE FAIL !|%s\n", body)
		t.Fail()
		return
	}
	fmt.Printf("--------------respose:%s\n", response)

	if response.Status != RESP_STATUS_SUCC {
		t.Fail()
		return
	}
}

func innerApsnHttpServerFeedback(t *testing.T) {

	fmt.Println("innerApsnHttpServerFeedback is Starting")

	resp, err := http.Get(PROXY_URL + "/apns/feedback?limit=50")

	if nil != err {
		t.Fail()
		t.Logf("HTTP GET FEEDBACK FAIL!%s\n", err)
		return
	}

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if nil != err {
		t.Logf("HTTP READ RESPONSE FAIL !|%s", err)
		t.Fail()
		return
	}
	defer resp.Body.Close()

	var response response
	err = json.Unmarshal(body, &response)
	if nil != err {
		t.Logf("HTTP Unmarshal RESPONSE FAIL !|%s\n", err)
		t.Fail()
		return
	}
	t.Logf("--------------respose:%s\n", response.Status)

	if response.Status != RESP_STATUS_SUCC {
		t.Fail()
		return
	} else {
		if reflect.TypeOf(response.Body).Kind() != reflect.Slice {
			t.Logf("--------------FEEDBACK IS NOT ARRAY TYPE :%s\n", response.Body)
			t.Fail()
			return
		} else {
			feedbacks := response.Body.([]interface{})
			t.Logf("--------------FEEDBACK: %s", feedbacks)
			if len(feedbacks) > 100 {
				t.Log("--------------FEEDBACK COUNT OVER FLOW 100")
				t.Fail()
				return
			}
		}

	}
}
