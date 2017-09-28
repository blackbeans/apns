package apns

import (
	"container/list"
	"context"
	"errors"
	"sync"
	"time"

	"log"
)

//connection pool
type ConnPool struct {
	ctx          context.Context
	dialFunc     func(ctx context.Context) (*ApnsConn, error)
	maxPoolSize  int           //最大尺子大小
	minPoolSize  int           //最小连接池大小
	corepoolSize int           //核心池子大小
	idletime     time.Duration //空闲时间

	workPool *list.List //当前正在工作的client
	idlePool *list.List //空闲连接

	running bool

	mutex sync.Mutex //全局锁
}

func NewConnPool(minPoolSize, corepoolSize,
	maxPoolSize int, idletime time.Duration,
	dialFunc func(ctx context.Context) (*ApnsConn, error)) (*ConnPool, error) {

	pool := &ConnPool{
		ctx:          context.Background(),
		maxPoolSize:  maxPoolSize,
		corepoolSize: corepoolSize,
		minPoolSize:  minPoolSize,
		idletime:     idletime,
		dialFunc:     dialFunc,
		running:      true,
		idlePool:     list.New(),
		workPool:     list.New()}

	err := pool.enhancedPool(pool.corepoolSize)
	if nil != err {
		return nil, err
	}

	//启动链接过期
	go pool.evict()

	return pool, nil
}

func (self *ConnPool) enhancedPool(size int) error {

	for i := 0; i < size; i++ {
		j := 0
		var err error
		var conn *ApnsConn
		for ; j < 3; j++ {
			conn, err = self.dialFunc(self.ctx)
			if nil != err || nil == conn {
				log.Printf("POOL_FACTORY|CREATE CONNECTION|INIT|FAIL|%s", err)
				continue
			} else {
				break
			}
		}

		if j >= 3 || nil == conn {
			return errors.New("POOL_FACTORY|CREATE CONNECTION|INIT|FAIL|%s" + err.Error())
		}
		self.idlePool.PushFront(conn)
	}

	return nil
}

func (self *ConnPool) evict() {
	for self.running {
		time.Sleep(self.idletime)
		self.checkIdle()
	}
}

//检查idle的数据
func (self *ConnPool) checkIdle() {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	for e := self.idlePool.Back(); nil != e; e = e.Prev() {
		idleconn := e.Value.(*ApnsConn)
		//too long time idle
		isExpired := time.Since(idleconn.worktime) >= self.idletime
		if !idleconn.alive || (isExpired &&
			(self.idlePool.Len()+self.workPool.Len()) > self.corepoolSize) {
			idleconn.Close()
			self.idlePool.Remove(e)
			idleconn = nil
		}
	}

	//create more connection
	enhanceSize := self.corepoolSize - (self.idlePool.Len() + self.workPool.Len())
	if enhanceSize > 0 {
		self.enhancedPool(enhanceSize)
	}
}

func (self *ConnPool) MonitorPool() (int, int, int) {
	return self.workPool.Len(), self.idlePool.Len(), (self.workPool.Len() + self.idlePool.Len())
}

func (self *ConnPool) Get() (*ApnsConn, error) {

	if !self.running {
		return nil, errors.New("POOL_FACTORY|POOL IS SHUTDOWN")
	}

	self.mutex.Lock()
	defer self.mutex.Unlock()
	var conn *ApnsConn
	var err error
	//先从Idealpool中获取如果存在那么就直接使用
	for e := self.idlePool.Back(); nil != e; e = e.Prev() {
		conn := e.Value.(*ApnsConn)
		//从idle列表中移除要么是存活的
		//要么是不存活都需要移除
		self.idlePool.Remove(e)
		if conn.alive {
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
			conn, err = self.dialFunc(self.ctx)
			if nil != err {
				conn = nil
			}
		} else {
			return nil, errors.New("POOLFACTORY|POOL|FULL!")
		}
	}
	//放入到工作连接池中去
	if nil != conn {
		self.workPool.PushBack(conn)
	}

	return conn, err
}

//释放坏的资源
func (self *ConnPool) ReleaseBroken(conn *ApnsConn) error {

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
func (self *ConnPool) Release(conn *ApnsConn) error {

	if nil != conn {
		self.mutex.Lock()
		defer self.mutex.Unlock()
		for e := self.workPool.Back(); nil != e; e = e.Prev() {
			if e.Value == conn {
				//move connection from workpool to idlepool
				self.workPool.Remove(e)
				//存活的才写入idle
				if conn.alive {
					self.idlePool.PushFront(conn)
				}
				break
			}
		}
	}

	return nil
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

		log.Printf("CONNECTION POOL|CLOSEING|WORK POOL SIZE|:%d\n", self.workPool.Len())
		i++
	}

	var idleconn *ApnsConn
	//关闭掉空闲的client
	for e := self.idlePool.Front(); e != nil; e = e.Next() {
		idleconn = e.Value.(*ApnsConn)
		idleconn.Close()
		self.idlePool.Remove(e)
		idleconn = nil
	}

	log.Println("CONNECTION_POOL|SHUTDOWN")
}
