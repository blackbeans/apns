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
	idletime     time.Duration //空闲时间

	workPool *list.List //当前正在工作的client
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

	pool := &ConnPool{
		maxPoolSize:  maxPoolSize,
		corepoolSize: corepoolSize,
		minPoolSize:  minPoolSize,
		idletime:     idletime,
		dialFunc:     dialFunc,
		running:      true,
		connectionId: 1,
		idlePool:     list.New(),
		workPool:     list.New()}

	err := pool.enhancedPool(pool.corepoolSize)
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
			if nil != err || nil == conn {
				log.Warn("POOL_FACTORY|CREATE CONNECTION|INIT|FAIL|%s", err)
				continue

			} else {
				break
			}
		}

		if j >= 3 || nil == conn {
			return errors.New("POOL_FACTORY|CREATE CONNECTION|INIT|FAIL|%s" + err.Error())
		}

		idleconn := &IdleConn{conn: conn, expiredTime: (time.Now().Add(self.idletime))}
		self.idlePool.PushFront(idleconn)
	}

	return nil
}

func (self *ConnPool) evict() {
	for self.running {

		select {
		case <-time.After(self.idletime):
			self.mutex.Lock()
			defer self.mutex.Unlock()
			for e := self.idlePool.Back(); nil != e; e = e.Prev() {
				idleconn := e.Value.(*IdleConn)
				//如果当前时间在过期时间之后或者活动的链接大于corepoolsize则关闭
				isExpired := idleconn.expiredTime.Before(time.Now())
				if isExpired ||
					(self.idlePool.Len()+self.workPool.Len()) > self.corepoolSize {
					idleconn.conn.Close()
					idleconn = nil
					self.idlePool.Remove(e)
					log.Debug("POOL_FACTORY|evict|Expired|%d/%d/%d",
						self.workPool.Len(), self.idlePool.Len(), (self.workPool.Len() + self.idlePool.Len()))
				}
			}

			//检查当前的连接数是否满足corepoolSize,不满足则创建
			enhanceSize := self.corepoolSize - (self.idlePool.Len() + self.workPool.Len())
			if enhanceSize > 0 {
				//创建这个数量的连接
				self.enhancedPool(enhanceSize)
			}

		}
	}
}

func (self *ConnPool) MonitorPool() (int, int, int) {
	return self.workPool.Len(), self.idlePool.Len(), (self.workPool.Len() + self.idlePool.Len())
}

func (self *ConnPool) Get() (error, IConn) {

	if !self.running {
		return errors.New("POOL_FACTORY|POOL IS SHUTDOWN"), nil
	}

	self.mutex.Lock()
	defer self.mutex.Unlock()
	var conn IConn
	var err error
	//先从Idealpool中获取如果存在那么就直接使用
	for e := self.idlePool.Back(); nil != e; e = e.Prev() {
		idle := e.Value.(*IdleConn)
		conn = idle.conn
		//从idle列表中移除要么是存活的
		//要么是不存活都需要移除
		self.idlePool.Remove(e)
		if conn.IsAlive() {
			break
		} else {
			//归还broken Conn
			conn = nil
		}
	}

	//如果当前依然是conn
	if nil == conn {
		//只有当前活动的链接小于最大的则创建
		if (self.idlePool.Len() + self.workPool.Len()) < self.maxPoolSize {
			//如果没有可用链接则创建一个
			err, conn = self.dialFunc(self.id())
			if nil != err {
				conn = nil
			}
		} else {
			return errors.New("POOLFACTORY|POOL|FULL!"), nil
		}
	}
	//放入到工作连接池中去
	if nil != conn {
		self.workPool.PushBack(conn)
	}

	return err, conn
}

//释放坏的资源
func (self *ConnPool) ReleaseBroken(conn IConn) error {

	self.mutex.Lock()
	defer self.mutex.Unlock()
	if nil != conn {
		for e := self.workPool.Back(); nil != e; e = e.Prev() {
			if e.Value == conn {
				self.workPool.Remove(e)
				break
			}
		}
		conn.Close()
		conn = nil
	}

	return nil
}

/**
* 归还当前的连接
**/
func (self *ConnPool) Release(conn IConn) error {

	if nil != conn && conn.IsAlive() {
		idleconn := &IdleConn{conn: conn, expiredTime: (time.Now().Add(self.idletime))}
		self.mutex.Lock()
		defer self.mutex.Unlock()
		for e := self.workPool.Back(); nil != e; e = e.Prev() {
			if e.Value == conn {
				//如果和当前的一样则从workpool中删除并放入到idle中
				self.workPool.Remove(e)
				self.idlePool.PushFront(idleconn)
				break
			}
		}
	}

	return nil
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
		if self.workPool.Len() <= 0 {
			break
		}

		log.Info("CONNECTION POOL|CLOSEING|WORK POOL SIZE|:%d", self.workPool.Len())
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
