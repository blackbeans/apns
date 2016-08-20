package lb

import (
	"github.com/blackbeans/go-moa/proxy"
	log "github.com/blackbeans/log4go"
	"strings"
	"time"
)

const (
	PROTOCOL            = "redis"
	REGISTRY_MOMOKEEPER = "momokeeper"
	REGISTRY_ZOOKEEPER  = "zookeeper"
)

type ConfigCenter struct {
	registry IRegistry
	services []proxy.Service
	hostport string
}

type IRegistry interface {
	RegisteService(serviceUri, hostport, protoType string) bool
	UnRegisteService(serviceUri, hostport, protoType string) bool
	GetService(serviceUri, protoType string) ([]string, error)
	Destroy()
}

//用于创建
func NewConfigCenter(registryType string, registryAddr string,
	hostport string, services []proxy.Service) *ConfigCenter {
	var reg IRegistry
	if registryType == REGISTRY_MOMOKEEPER {
		split := strings.Split(registryAddr, ",")
		if len(split) > 1 {
			reg = NewMomokeeper(split[0], split[1])
		} else {
			reg = NewMomokeeper(split[0], split[0])
		}

	} else if registryType == REGISTRY_ZOOKEEPER {
		reg = NewZookeeper(registryAddr, []string{})
	}
	center := &ConfigCenter{registry: reg, services: services, hostport: hostport}
	//如果是momokeeper则定时注册服务
	if registryType == REGISTRY_MOMOKEEPER {
		go func() {
			for {
				time.Sleep(1 * time.Minute)
				func() {
					defer func() {
						if err := recover(); nil != err {

						}
					}()
					//注册一下服务
					center.RegisteAllServices()
				}()
			}
		}()
	} else if registryType == REGISTRY_ZOOKEEPER {
		//	 zookeeper发布一次吧
		center.RegisteAllServices()
	}
	return center
}

func (self ConfigCenter) RegisteAllServices() {
	//注册服务
	for _, s := range self.services {
		succ := self.RegisteService(s.ServiceUri, self.hostport, PROTOCOL)
		if !succ {
			panic("ConfigCenter|RegisteAllServices|FAIL|" + s.ServiceUri)
		}
	}

}

func (self ConfigCenter) RegisteService(serviceUri, hostport, protoType string) bool {
	return self.registry.RegisteService(serviceUri, hostport, protoType)
}

func (self ConfigCenter) UnRegisteService(serviceUri, hostport, protoType string) bool {
	return self.registry.UnRegisteService(serviceUri, hostport, protoType)
}

func (self ConfigCenter) GetService(serviceUri, protoType string) ([]string, error) {
	return self.registry.GetService(serviceUri, protoType)
}

func (self ConfigCenter) Destroy() {
	//注册服务
	for _, s := range self.services {
		succ := self.UnRegisteService(s.ServiceUri, self.hostport, PROTOCOL)
		if succ {
			log.InfoLog("config_center", "ConfigCenter|Destroy|UnRegisteService|SUCC|%s", s.ServiceUri)
		} else {
			log.InfoLog("config_center", "ConfigCenter|Destroy|UnRegisteService|FAIL|%s", s.ServiceUri)
		}
	}
	self.registry.Destroy()
}
