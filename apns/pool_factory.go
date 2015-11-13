package apns

import (
	"container/list"
	"errors"
	log "github.com/blackbeans/log4go"
	"sync"
	"sync/atomic"
	"time"
)

//连接工厂
type IConnFactory interface {
	Get() (error, IConn)            //获取一个连接
	Release(conn IConn) error       //释放对应的链接
	ReleaseBroken(conn IConn) error //释放掉坏的连接
	Shutdown()                      //关闭当前的
	MonitorPool() (int, int, int)
}

//apnsconn的连接池
type ConnPool struct {
	dialFunc     func(connectionId int32) (error, IConn)
	maxPoolSize  int           //最大尺子大小
	minPoolSize  int           //最小连接池大小
	corepoolSize int           //核心池子大小
	numActive    int           //当前正在存活的client
	numWork      int           //当前正在工作的client
	idletime     time.Duration //空闲时间

	idlePool *list.List //空闲连接

	running bool

	connectionId int32 //链接的Id

	mutex sync.Mutex //全局锁
}

type IdleConn struct {
	conn        IConn
	expiredTime time.Time
}

func NewConnPool(minPoolSize, corepoolSize,
maxPoolSize int, idletime time.Duration,
dialFunc func(connectionId int32) (error, IConn)) (error, *ConnPool) {

	idlePool := list.New()
	pool := &ConnPool{
		maxPoolSize:  maxPoolSize,
		corepoolSize: corepoolSize,
		minPoolSize:  minPoolSize,
		idletime:     idletime,
		idlePool:     idlePool,
		dialFunc:     dialFunc,
		running:      true,
		connectionId: 1}

	err := pool.enhancedPool(pool.minPoolSize)
	if nil != err {
		return err, nil
	}

	//启动链接过期
	go pool.evict()

	return nil, pool
}

func (self *ConnPool) enhancedPool(size int) error {

	//初始化一下最小的Poolsize,让入到idlepool中
	for i := 0; i < size; i++ {
		j := 0
		var err error
		var conn IConn
		for ; j < 3; j++ {
			err, conn = self.dialFunc(self.id())
			if nil != err {
				log.Warn("POOL_FACTORY|CREATE CONNECTION|INIT|FAIL|%s", err)

			} else {
				break
			}
		}

		if j >= 3 {
			return errors.New("POOL_FACTORY|CREATE CONNECTION|INIT|FAIL|%s" + err.Error())
		}

		idleconn := &IdleConn{conn: conn, expiredTime: (time.Now().Add(self.idletime))}
		self.idlePool.PushFront(idleconn)
		self.numActive++
	}

	return nil
}

func (self *ConnPool) evict() {
	for self.running {

		select {
		case <-time.After(self.idletime):
			self.mutex.Lock()
			for e := self.idlePool.Back(); nil != e; e = e.Prev() {
				idleconn := e.Value.(*IdleConn)
				//如果当前时间在过期时间之后或者活动的链接大于corepoolsize则关闭
				isExpired := idleconn.expiredTime.Before(time.Now())
				if isExpired ||
				self.numActive >= self.corepoolSize {
					idleconn.conn.Close()
					idleconn = nil
					self.idlePool.Remove(e)
					//并且该表当前的active数量
					self.numActive--
				}
			}

			//检查当前的连接数是否满足corepoolsize,不满足则创建
			enhanceSize := self.corepoolSize - self.numActive
			if enhanceSize > 0{
				//创建这个数量的连接
				self.enhancedPool(enhanceSize);
			}


			self.mutex.Unlock()
		}
	}
}

func (self *ConnPool) MonitorPool() (int, int, int) {
	return self.numWork, self.numActive, self.idlePool.Len()
}

func (self *ConnPool) Get() (error, IConn) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if !self.running {
		return errors.New("POOL_FACTORY|POOL IS SHUTDOWN"), nil
	}

	var conn IConn
	var err error
	//先从Idealpool中获取如果存在那么就直接使用
	if self.idlePool.Len() > 0 {
		e := self.idlePool.Back()
		idle := e.Value.(*IdleConn)
		self.idlePool.Remove(e)
		conn = idle.conn
	}

	//如果当前依然是conn
	if nil == conn {
		//只有当前活动的链接小于最大的则创建
		if self.numActive < self.maxPoolSize {
			//如果没有可用链接则创建一个
			err, conn = self.dialFunc(self.id())
			if nil != err {
				conn = nil
			} else {
				self.numActive++
			}
		} else {
			return errors.New("POOLFACTORY|POOL|FULL!"), nil
		}
	}

	if nil != conn {
		self.numWork++
	}

	return err, conn
}

//释放坏的资源
func (self *ConnPool) ReleaseBroken(conn IConn) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	if nil != conn {
		conn.Close()
		conn = nil
	}

	var err error
	//只有当前的存活链接和当前工作链接大于0的时候才会去销毁
	if self.numActive > 0 && self.numWork > 0 {
		self.numWork--
		self.numActive--

	} else {
		err = errors.New("POOL|RELEASE BROKEN|INVALID CONN")
	}

	//判断当前是否连接不是最小连接
	incrCount := self.minPoolSize - self.numActive
	if incrCount > 0 {
		//如果不够最小连接则创建
		err = self.enhancedPool(incrCount)
	}

	return err
}

/**
* 归还当前的连接
**/
func (self *ConnPool) Release(conn IConn) error {

	idleconn := &IdleConn{conn: conn, expiredTime: (time.Now().Add(self.idletime))}

	self.mutex.Lock()
	defer self.mutex.Unlock()

	if self.numWork > 0 {
		//放入ideal池子中
		self.idlePool.PushFront(idleconn)
		//工作链接数量--
		self.numWork--
		return nil
	} else {
		conn.Close()
		conn = nil
		log.Warn("POOL|RELEASE|FAIL|%d\n", self.numActive)
		return errors.New("POOL|RELEASE|INVALID CONN")
	}

}

//生成connectionId
func (self *ConnPool) id() int32 {
	return atomic.AddInt32(&self.connectionId, 1)
}

func (self *ConnPool) Shutdown() {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	self.running = false

	for i := 0; i < 3; {
		//等待五秒中结束
		time.Sleep(5 * time.Second)
		if self.numWork <= 0 {
			break
		}

		log.Info("CONNECTION POOL|CLOSEING|WORK POOL SIZE|:%d", self.numWork)
		i++
	}

	var idleconn *IdleConn
	//关闭掉空闲的client
	for e := self.idlePool.Front(); e != nil; e = e.Next() {
		idleconn = e.Value.(*IdleConn)
		idleconn.conn.Close()
		self.idlePool.Remove(e)
		idleconn = nil
	}

	log.Info("CONNECTION_POOL|SHUTDOWN")
}
