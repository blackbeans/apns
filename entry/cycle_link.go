package entry

import (
	"fmt"
	"sync"
)

/**
*带有hash的循环链表，支持随机查询
*此循环链表用于在内存中记录一下已经发送的message
*友好地遍历数据的同时删除元素
*自动过滤message中的ttl为0的数据
 */
type node struct {
	id   int32 //只使用在enhanced的情况下
	msg  *Message
	pre  *node
	next *node
}

//循环链表
type CycleLink struct {
	head        *node           //循环链表
	length      int             //当前节点联调的长度
	hash        map[int32]*node //记录了hash的节点，方便定位
	mutex       sync.Mutex      //并发控制
	maxCapacity int             //最大节点数量
	maxttl      uint8           //最大生存周期
}

func NewCycleLink(maxttl uint8, maxCapacity int) *CycleLink {
	link := &CycleLink{}
	link.maxCapacity = maxCapacity
	link.hash = make(map[int32]*node, maxCapacity/2)
	link.maxttl = maxttl
	link.head = nil

	return link
}

func (self *CycleLink) Insert(id int32, msg *Message) {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	//如果已经存在该id对应的数据则覆盖
	if msg.ttl > self.maxttl {
		msg.ttl = self.maxttl
	} else if msg.ttl <= 0 {
		//如果ttl到达0则不进行存储抛弃
		return
	}

	v, ok := self.hash[id]
	if !ok {

		n := &node{id: id, msg: msg}
		//这里判断一下是否达到了最大的容量，如果达到了就覆盖头节点的数据，否则就pushback
		if self.length >= self.maxCapacity {

			//取出head的数据，并删除hash中的Key-Value
			delete(self.hash, self.head.id)
			//将头结点的数据改为新的数据，并重新构建hash对应关系
			self.head.id = id
			self.head.msg = msg
			self.hash[id] = n

		} else {

			//如果还么有初始化
			if self.length <= 0 {
				n.next = n
				n.pre = n
				self.head = n

			} else {

				//直接将n的pre 指向tail,将next指向 tail.next
				n.pre = self.head.pre
				n.next = self.head
				//header的pre指向 n
				self.head.pre = n
				n.pre.next = n

			}

			self.hash[id] = n
			self.length++

		}
	} else {
		v.msg = msg
	}

}

func (self *CycleLink) innerRemove(n *node) *node {

	tmp := n.next
	//如果n为head节点，这时候需要移动Head节点
	if n == self.head {
		self.head = n.next
	}
	//如果还有下一个数据，则进行断开指针操作
	if nil != n.next {
		//从当前链表中取出n
		n.next.pre = n.pre
		n.pre.next = n.next
	}
	//删除map中保留的索引
	delete(self.hash, n.id)
	self.length--

	//释放空间
	n.next = nil
	n.pre = nil
	n = nil

	return tmp

}

//删除元素
func (self *CycleLink) Remove(startId int32, endId int32, ch chan<- *Message) {

	self.mutex.Lock()
	defer self.mutex.Unlock()

	start, ok_h := self.hash[startId]
	end, ok_e := self.hash[endId]
	// //如果endId为-1那么就代表清空节点
	if endId == -1 {
		//end为head的pre
		end = self.head.pre
		ok_e = true
	}
	//如果不存在这样start 则直接返回
	if !(ok_h && ok_e) {
		ch <- nil
		return
	}

	//一个接一个地获取并删除节点，endId为-1
	for n := start; nil != n && func() bool {
		if endId != -1 {
			return n != end
		} else {
			return true
		}
	}(); n = self.innerRemove(n) {

		n.msg.ttl--
		//写入channel 让另一侧重发
		ch <- n.msg
		fmt.Printf("CY|REMOVE|%t\n", self.head)
	}

	ch <- nil
}
