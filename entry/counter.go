package entry

import (
	"sync/atomic"
)

//没有要求那么严格所以就不加锁了
type Counter struct {
	counter     int64
	lasterCount int64
}

func (self *Counter) Incr(num int64) {
	atomic.AddInt64(&self.counter, num)
}

func (self *Counter) Changes() int {
	changes := int(self.counter - self.lasterCount)
	self.lasterCount = self.counter

	return changes
}
