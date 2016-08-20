package core

import (
	"errors"
	log "github.com/blackbeans/log4go"
	"github.com/naoina/toml"
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"strings"
	"time"
)

type HostPort struct {
	Hosts string
}

//配置信息
type Option struct {
	Env struct {
		Name             string
		RunMode          string
		BindAddress      string
		RegistryType     string
		ServiceUriSuffix string
	}

	//使用的环境
	Registry map[string]HostPort //momokeeper的配置
	Clusters map[string]Cluster  //各集群的配置
}

//----------------------------------------
//Cluster配置
type Cluster struct {
	Env               string //当前环境使用的是dev还是online
	ProcessTimeout    int    //处理超时 5 s单位
	MaxDispatcherSize int    //=8000//最大分发处理协程数
	ReadBufferSize    int    //=16 * 1024 //读取缓冲大小
	WriteBufferSize   int    //=16 * 1024 //写入缓冲大小
	WriteChannelSize  int    //=1000 //写异步channel长度
	ReadChannelSize   int    //=1000 //读异步channel长度
	LogFile           string //log4go的文件路径
}

//---------最终需要的Option
type MOAOption struct {
	name              string
	registryType      string
	registryHosts     string
	hostport          string
	processTimeout    time.Duration
	maxDispatcherSize int           //=8000//最大分发处理协程数
	readBufferSize    int           //=16 * 1024 //读取缓冲大小
	writeBufferSize   int           //=16 * 1024 //写入缓冲大小
	writeChannelSize  int           //=1000 //写异步channel长度
	readChannelSize   int           //=1000 //读异步channel长度
	idleDuration      time.Duration //=60s //连接空闲时间
	serviceUriSuffix  string        //serviceUri后缀
}

func LoadConfiruation(path string) (*MOAOption, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buff, rerr := ioutil.ReadAll(f)
	if nil != rerr {
		return nil, rerr
	}
	// log.DebugLog("application", "LoadConfiruation|Parse|toml:%s", string(buff))
	//读取配置
	var option Option
	err = toml.Unmarshal(buff, &option)
	if nil != err {
		return nil, err
	}

	cluster, ok := option.Clusters[option.Env.RunMode]
	if !ok {
		return nil, errors.New("no cluster config for " + option.Env.RunMode)
	}

	//加载Log4go
	log.LoadConfiguration(cluster.LogFile)
	reg, exist := option.Registry[option.Env.RunMode]
	if !exist {
		return nil, errors.New("no reg  for " + option.Env.RunMode + ":" + cluster.Env)
	}

	if cluster.MaxDispatcherSize <= 0 {
		cluster.MaxDispatcherSize = 8000 //最大分发处理协程数
	}

	if cluster.ReadBufferSize <= 0 {
		cluster.ReadBufferSize = 16 * 1024 //读取缓冲大小
	}

	if cluster.WriteBufferSize <= 0 {
		cluster.WriteBufferSize = 16 * 1024 //写入缓冲大小
	}

	if cluster.WriteChannelSize <= 0 {
		cluster.WriteChannelSize = 1000 //写异步channel长度
	}

	if cluster.ReadChannelSize <= 0 {
		cluster.ReadChannelSize = 1000 //读异步channel长度

	}

	if len(option.Env.ServiceUriSuffix) <= 0 {
		option.Env.ServiceUriSuffix = ""
	}
	//------------寻找匹配的网卡IP段，进行匹配
	split := strings.Split(option.Env.BindAddress, ":")
	regx := split[0]

	inters, err := net.Interfaces()
	if nil != err {
		panic(err)
	} else {
		hasMatched := false
		//如果没有IP匹配表达式则用默认的
		if len(regx) <= 0 {
			option.Env.BindAddress = "0.0.0.0:" + split[1]
			hasMatched = true
		} else {
			for _, inter := range inters {
				addrs, _ := inter.Addrs()
				for _, addr := range addrs {
					if ip, ok := addr.(*net.IPNet); ok && !ip.IP.IsLoopback() {
						if nil != ip.IP.To4() {
							match, _ := regexp.MatchString(regx, ip.IP.To4().String())
							if match {
								option.Env.BindAddress = ip.IP.To4().String() + ":" + split[1]
								hasMatched = true
								break
							}
						}
					}
				}
			}
		}
		//没有匹配的IP直接用0.0.0.0的IP绑定
		if !hasMatched {
			for _, inter := range inters {
				addrs, _ := inter.Addrs()
				loopback := false
				for _, addr := range addrs {
					ip, ok := addr.(*net.IPNet)
					if ok && ip.IP.IsLoopback() {
						loopback = true
						//skipped
						break
					}
				}

				if !loopback && len(addrs) > 0 {
					for _, addr := range addrs {
						if ip, ok := addr.(*net.IPNet); ok &&
							!ip.IP.IsLoopback() && nil != ip.IP.To4() {
							option.Env.BindAddress = ip.IP.To4().String() + ":" + split[1]
							hasMatched = true
							break
						}
					}
				}
			}

			if !hasMatched {
				option.Env.BindAddress = "0.0.0.0" + ":" + split[1]

			}
		}
	}

	//拼装为可用的MOA参数
	mop := &MOAOption{}
	mop.name = option.Env.Name
	mop.serviceUriSuffix = option.Env.ServiceUriSuffix
	mop.hostport = option.Env.BindAddress
	mop.registryType = option.Env.RegistryType
	mop.registryHosts = reg.Hosts
	mop.processTimeout = time.Duration(int64(cluster.ProcessTimeout) * int64(time.Second))
	mop.maxDispatcherSize = cluster.MaxDispatcherSize //最大分发处理协程数
	mop.readBufferSize = cluster.ReadBufferSize       //读取缓冲大小
	mop.writeBufferSize = cluster.WriteBufferSize     //写入缓冲大小
	mop.writeChannelSize = cluster.WriteChannelSize   //写异步channel长度
	mop.readChannelSize = cluster.ReadChannelSize     //读异步channel长度
	mop.idleDuration = 60 * time.Second
	return mop, nil

}
