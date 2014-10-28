package apns

import (
	"container/list"
	"errors"
	"log"
	"sync"
	"time"
)

//连接工厂
type IConnFactory interface {
	Get(timeout time.Duration) (error, IConn) //获取一个连接
	Release(conn IConn) error                 //释放对应的链接
	Shutdown()                                //关闭当前的
	MonitorPool() (int, int, int)
}

//apnsconn的连接池
type ConnPool struct {
	dialFunc     func() (error, IConn)
	maxPoolSize  int //最大尺子大小
	minPoolSize  int //最小连接池大小
	corepoolSize int //核心池子大小
	// activePoolSize int //当前正在运行的client

	idletime time.Duration //空闲时间

	idlePool *list.List //连接的队列

	checkOutPool *list.List //已经获取的poolsize

	mutex sync.Mutex

	running bool
}

type IdleConn struct {
	conn        IConn
	expiredTime time.Time
}

func NewConnPool(minPoolSize, corepoolSize,
	maxPoolSize int, idletime time.Duration,
	dialFunc func() (error, IConn)) (error, *ConnPool) {

	idlePool := list.New()
	checkOutPool := list.New()
	pool := &ConnPool{
		maxPoolSize:  maxPoolSize,
		corepoolSize: corepoolSize,
		minPoolSize:  minPoolSize,
		idletime:     idletime,
		idlePool:     idlePool,
		dialFunc:     dialFunc,
		checkOutPool: checkOutPool,
		running:      true}

	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	//初始化一下最小的Poolsize,让入到idlepool中
	for i := 0; i < pool.minPoolSize; i++ {
		j := 0
		var err error
		var conn IConn
		for ; j < 3; j++ {
			err, conn = dialFunc()
			if nil != err {
				log.Printf("POOL_FACTORY|CREATE CONNECTION|INIT|FAIL|%s\n", err)

			} else {
				break
			}
		}

		if j >= 3 {
			return errors.New("POOL_FACTORY|CREATE CONNECTION|INIT|FAIL|%s" + err.Error()), nil
		}

		idleconn := &IdleConn{conn: conn, expiredTime: (time.Now().Add(pool.idletime))}
		pool.idlePool.PushFront(idleconn)
		// pool.activePoolSize++
	}
	return nil, pool
}

func (self *ConnPool) MonitorPool() (int, int, int) {
	return self.activePoolSize(), self.corePoolSize(), self.maxPoolSize
}

func (self *ConnPool) Get(timeout time.Duration) (error, IConn) {

	if !self.running {
		return errors.New("flume pool has been stopped!"), nil
	}

	//***如果在等待的时间内没有获取到client则超时
	var conn IConn
	clientch := make(chan IConn, 1)
	defer close(clientch)
	go func() {
		conn := self.innerGet()
		clientch <- conn
	}()

	select {
	case conn = <-clientch:
		return nil, conn
		break
	case <-time.After(time.Second * timeout):
		return errors.New("POOL|GET CONN|TIMEOUT|FAIL!"), nil
		break
	}
	//here is a bug
	return nil, nil
}

//返回当前的corepoolszie
func (self *ConnPool) corePoolSize() int {
	return self.idlePool.Len() + self.checkOutPool.Len()

}

func (self *ConnPool) activePoolSize() int {
	return self.checkOutPool.Len()
}

//释放坏的资源
func (self *ConnPool) ReleaseBroken(conn IConn) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	_, err := self.innerRelease(conn)
	return err

}

func (self *ConnPool) innerRelease(conn IConn) (bool, error) {
	for e := self.checkOutPool.Back(); nil != e; e = e.Prev() {
		conn := e.Value.(IConn)
		if conn == conn {
			self.checkOutPool.Remove(e)
			// log.Println("client return pool ")
			return true, nil
		}
	}

	//如果到这里，肯定是Bug，释放了一个游离态的客户端
	return false, errors.New("invalid connection , this is not managed by pool")

}

/**
* 归还当前的连接
**/
func (self *ConnPool) Release(conn IConn) error {

	idleconn := &IdleConn{conn: conn, expiredTime: (time.Now().Add(self.idletime))}
	self.mutex.Lock()
	defer self.mutex.Unlock()

	//从checkoutpool中移除
	succ, err := self.innerRelease(conn)
	if nil != err {
		return err
	}

	//如果当前的corepoolsize 是大于等于设置的corepoolssize的则直接销毁这个client
	if self.corePoolSize() >= self.corepoolSize {

		idleconn.conn.Close()
		conn = nil

		//并且从idle
	} else if succ {
		self.idlePool.PushFront(idleconn)
	} else {
		conn.Close()
		conn = nil
	}

	return nil

}

//从现有队列中获取，没有了就创建、有就获取达到上限就阻塞
func (self *ConnPool) innerGet() IConn {

	var conn IConn
	//首先检查一下当前空闲连接中是否有需要关闭的
	self.mutex.Lock()
	defer self.mutex.Unlock()
	for back := self.idlePool.Back(); back != nil; back = back.Prev() {
		// push ---> front ----> back 最旧的client
		idle := (back.Value).(*IdleConn)

		//如果已经挂掉直接移除
		if !idle.conn.IsAlive() {
			self.idlePool.Remove(back)
			idle.conn.Close()
			continue
		}

		//只有在corepoolsize>最小的池子大小，才去检查过期连接
		if self.corePoolSize() > self.minPoolSize {
			//如果过期时间实在当前时间之后那么后面的都不过期
			if idle.expiredTime.After(time.Now()) {
				//判断一下当前连接的状态是否为alive 否则直接销毁
				self.idlePool.Remove(back)
				idle.conn.Close()
			}
		} else {
			//如果小于等于Minpoolsize时，如果过期就将时间重置

			if idle.expiredTime.After(time.Now()) {
				idle.expiredTime = time.Now().Add(self.idletime)
			}
		}
	}

	//优先从空闲链接中获取链接
	for i := 0; i < self.idlePool.Len(); i++ {
		back := self.idlePool.Back()
		idle := back.Value.(*IdleConn)
		conn = idle.conn
		self.checkOutPool.PushFront(conn)
		self.idlePool.Remove(back)
		break
	}

	//如果client还是没有那么久创建链接
	if nil == conn {
		//工作连接数和空闲连接数已经达到最大的连接数上限
		if self.corePoolSize() >= self.maxPoolSize {
			log.Printf("CONNECTION POOL|minPoolSize:%d,maxPoolSize:%d,corePoolSize:%d,activePoolSize:%d\n ",
				self.minPoolSize, self.maxPoolSize, self.corePoolSize(), self.activePoolSize())
			return conn
		} else {
			//如果没有可用链接则创建一个
			err, tmpconn := self.dialFunc()
			if nil != err {

			} else {
				self.checkOutPool.PushFront(tmpconn)
				conn = tmpconn
			}
		}
	}

	//检查是否corepool>= minpool,否则就创建连接
	if self.corePoolSize() < self.minPoolSize {
		for i := self.corePoolSize(); i <= self.minPoolSize; i++ {
			//如果没有可用链接则创建一个
			err, tmpconn := self.dialFunc()
			if nil != err {
				log.Printf("POOL|CORE(%d) < MINI(%d)|CREATE CLIENT FAIL|%s",
					self.corePoolSize(), self.minPoolSize, err.Error())
			} else {
				idleconn := &IdleConn{conn: tmpconn, expiredTime: (time.Now().Add(self.idletime))}
				self.idlePool.PushFront(idleconn)
			}
		}
	}

	return conn
}

func (self *ConnPool) Shutdown() {
	self.mutex.Lock()
	self.running = false
	self.mutex.Unlock()

	for i := 0; i < 3; {
		//等待五秒中结束
		time.Sleep(5 * time.Second)
		if self.activePoolSize() <= 0 {
			break
		}

		log.Printf("CONNECTION POOL|CLOSEING|ACTIVE POOL SIZE|:%d\n", self.activePoolSize())
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
	var conn IConn
	//关闭掉已经
	for e := self.checkOutPool.Front(); e != nil; e = e.Next() {
		conn = e.Value.(IConn)
		conn.Close()
		self.checkOutPool.Remove(e)
		conn = nil
	}

	log.Printf("CONNECTION_POOL|SHUTDOWN")
}
