package lb

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/blackbeans/go-zookeeper/zk"
	log "github.com/blackbeans/log4go"
	"regexp"
	"sort"
	"sync"
)

const (
	// /moa/service/redis/service/relation-service/localhost:13000?timeout=1000&protocol=redis
	ZK_MOA_ROOT_PATH  = "/moa/service"
	ZK_ROOT           = "/"
	ZK_PATH_DELIMITER = "/"
)

type zookeeper struct {
	serviceUri  []string
	zkManager   *ZKManager
	uri2Hosts   map[string][]string
	lock        sync.RWMutex
	serverModel bool
}

func NewZookeeper(regAddr string, uris []string) *zookeeper {

	zkManager := NewZKManager(regAddr)
	uri2Hosts := make(map[string][]string, 2)

	zoo := &zookeeper{}
	zoo.serviceUri = uris
	zoo.zkManager = zkManager
	zoo.uri2Hosts = uri2Hosts
	zoo.serverModel = true

	if len(uris) > 0 {
		// client
		zoo.serverModel = false
		for _, uri := range uris {

			// 初始化，由于客户端订阅延迟，需要主动监听节点事件，然后主动从zk上拉取一次，放入缓存
			servicePath := concat(ZK_MOA_ROOT_PATH, ZK_PATH_DELIMITER, PROTOCOL, uri)
			flag := zkManager.RegisteWatcher(servicePath, zoo)
			if !flag {
				log.ErrorLog("config_center", "zookeeper|NewZookeeper|RegisteWather|FAIL|%s", uri)
			}
			hosts, _, _, err := zkManager.session.ChildrenW(servicePath)
			if err != nil {
				log.ErrorLog("config_center", "zookeeper|NewZookeeper|init uri2hosts|FAIL|%s", uri)
			} else {
				sort.Strings(hosts)
				uri2Hosts[uri] = hosts
			}
		}
	} else {
		// server
		zkManager.RegisteWatcher(ZK_MOA_ROOT_PATH, zoo)
	}

	return zoo
}

func (self zookeeper) RegisteService(serviceUri, hostport, protoType string) bool {
	// /moa/service/redis/service/relation-service/localhost:13000?timeout=1000&protocol=redis
	// hostport = "localhost:13000" //test
	servicePath := concat(ZK_MOA_ROOT_PATH, ZK_PATH_DELIMITER, protoType, serviceUri)
	svAddrPath := concat(servicePath, ZK_PATH_DELIMITER, hostport)

	conn := self.zkManager.session

	// 创建持久服务节点 /moa/service/redis/service/relation-service
	exist, _, err := conn.Exists(servicePath)
	if err != nil {
		conn.Close()
		panic("无法创建" + servicePath + err.Error())
	}
	if !exist {
		err = self.zkManager.CreateNode(conn, servicePath)
		if err != nil {
			panic("NewZookeeper|RegisteService|FAIL|" + servicePath + "|" + err.Error())
		}
	}

	// 创建临时服务地址节点 /moa/service/redis/service/relation-service/localhost:13000?timeout=1000&protocol=redis
	// 先删除，后创建吧。不然zk不通知，就坐等坑爹吧。蛋碎了一地。/(ㄒoㄒ)/~~

	conn.Delete(svAddrPath, 0)
	_, err = conn.Create(svAddrPath, nil, zk.CreateEphemeral, zk.WorldACL(zk.PermAll))
	if err != nil {
		panic("NewZookeeper|RegisteService|FAIL|" + svAddrPath + "|" + err.Error())
	}
	log.InfoLog("config_center", "zookeeper|RegisteService|SUCC|%s|%s|%s", hostport, serviceUri, protoType)
	return true
}

func (self zookeeper) UnRegisteService(serviceUri, hostport, protoType string) bool {

	servicePath := concat(ZK_MOA_ROOT_PATH, ZK_PATH_DELIMITER, protoType, serviceUri, ZK_PATH_DELIMITER, hostport)
	conn := self.zkManager.session
	if flag, _, err := conn.Exists(servicePath); err != nil {
		log.ErrorLog("config_center", "zookeeper|UnRegisteService|ERROR|%s|%s|%s", serviceUri, hostport, protoType)
		return false
	} else {
		if flag {
			err := conn.Delete(servicePath, 0)
			if err != nil {
				log.ErrorLog("config_center", "zookeeper|UnRegisteService|ERROR|%s|%s|%s", serviceUri, hostport, protoType)
				return false
			}
		}
	}
	log.InfoLog("config_center", "zookeeper|UnRegisteService|SUCC|%s|%s|%s", hostport, serviceUri, protoType)
	return true
}

func (self zookeeper) GetService(serviceUri, protoType string) ([]string, error) {
	// log.WarnLog("config_center", "zookeeper|GetService|SUCC|%s|%s|%s", serviceUri, protoType, self.addrManager.uri2Hosts)
	self.lock.RLock()
	defer self.lock.RUnlock()
	hosts, ok := self.uri2Hosts[serviceUri]
	if !ok {
		if len(hosts) < 1 {
			return nil, errors.New(fmt.Sprintf("No Hosts! /moa/service/%s%s", protoType, serviceUri))
		}
	}
	return hosts, nil
}

//会话超时时，需要重新订阅/推送watcher
func (self zookeeper) OnSessionExpired() {
	if !self.serverModel {
		// 服务端 需要重新推送
		conn := self.zkManager.session
		for uri, hosts := range self.uri2Hosts {
			servicePath := concat(ZK_MOA_ROOT_PATH, ZK_PATH_DELIMITER, PROTOCOL, uri)
			for _, host := range hosts {
				svAddrPath := concat(servicePath, ZK_PATH_DELIMITER, host)
				conn.Delete(svAddrPath, 0)
				_, err := conn.Create(svAddrPath, nil, zk.CreateEphemeral, zk.WorldACL(zk.PermAll))
				if err != nil {
					panic("ReSubZkServer|FAIL|" + svAddrPath + "|" + err.Error())
				}
			}
		}
		log.InfoLog("config_center", "zookeeper|OnSessionExpired|%v", self.serverModel)
	} else {
		// 客户端需要重新订阅
		conn := self.zkManager.session
		for _, uri := range self.serviceUri {
			servicePath := concat(ZK_MOA_ROOT_PATH, ZK_PATH_DELIMITER, PROTOCOL, uri)
			conn.ChildrenW(servicePath)
		}
		log.InfoLog("config_center", "zookeeper|OnSessionExpired|%v", self.serverModel)
	}
}

// 用户客户端监听服务节点地址发生变化时触发
func (self zookeeper) NodeChange(path string, eventType ZkEvent, addrs []string) {
	reg, _ := regexp.Compile(`/moa/service/redis([^\s]*)`)
	uri := reg.FindAllStringSubmatch(path, -1)[0][1]

	needChange := true
	//对比变化
	func() {
		self.lock.RLock()
		defer self.lock.RUnlock()

		sort.Strings(addrs)
		oldAddrs, ok := self.uri2Hosts[uri]
		if ok {
			if len(oldAddrs) > 0 &&
				len(oldAddrs) == len(addrs) {
				for j, v := range addrs {
					//如果是最后一个并且相等那么就应该不需要更新
					if oldAddrs[j] == v && j == len(addrs)-1 {
						needChange = false
						break
					}
				}
			}
		}
	}()
	//变化则更新
	if needChange {
		self.lock.Lock()
		self.uri2Hosts[uri] = addrs
		self.lock.Unlock()
	}
	log.WarnLog("config_center", "zookeeper|NodeChange|%s|%s", uri, addrs)

}

// 拼接字符串
func concat(args ...string) string {
	var buffer bytes.Buffer
	for _, arg := range args {
		buffer.WriteString(arg)
	}
	return buffer.String()
}

func (self zookeeper) Destroy() {
	self.zkManager.Close()
}
