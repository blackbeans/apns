package entry

import (
	_ "log"
	"sync"
)

//通用的存储发送message的接口
type IMessageStorage interface {
	//删除接口带过滤条件
	Remove(startId uint32, endId uint32, filter func(id uint32, msg *Message) bool) chan *Message
	Insert(id uint32, msg *Message)
	Get(id uint32) *Message //获取某个消息
	Length() int            // 返回长度
}

/**
*带有hash的循环链表，支持随机查询
*此循环链表用于在内存中记录一下已经发送的message
*友好地遍历数据的同时删除元素
*自动过滤message中的ttl为0的数据
 */
type node struct {
	id   uint32 //只使用在enhanced的情况下
	msg  *Message
	next *node
	pre  *node
}

//循环链表
type CycleLink struct {
	head        *node            //循环链表
	length      int              //当前节点联调的长度
	hash        map[uint32]*node //记录了hash的节点，方便定位
	mutex       sync.Mutex       //并发控制
	maxCapacity int              //最大节点数量
	maxttl      uint8            //最大生存周期
}

func NewCycleLink(maxttl uint8, maxCapacity int) *CycleLink {
	link := &CycleLink{}
	link.maxCapacity = maxCapacity
	link.hash = make(map[uint32]*node, maxCapacity/2)
	link.maxttl = maxttl
	link.head = nil

	return link
}

func (self *CycleLink) Get(id uint32) *Message {
	val, ok := self.hash[id]
	if ok {
		return val.msg
	} else {
		return nil
	}
}

func (self *CycleLink) Length() int {
	return self.length
}

func (self *CycleLink) Insert(id uint32, msg *Message) {
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
			//删除当前头结点，返回新的头结点
			self.innerRemove(self.head)
			// //将头结点的数据改为新的数据，并重新构建hash对应关系
			// log.Printf("CYCLE-LINK|OVERFLOW|%d|%t", self.length, self.head)
		}

		//最后统一执行写入
		self.innerInsert(self.head, n)

	} else {
		v.msg = msg
	}

}

func (self *CycleLink) innerInsert(h *node, n *node) {

	if n.msg.ttl <= 0 {
		//如果当前的写入的node中的msg如果ttl为0 那么直接丢弃
		return
	}

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
		self.head.pre.next = n
		self.head.pre = n
	}

	self.hash[n.id] = n
	self.length++
}

/**
*
*删除当前节点n ,并下一个节点
**/
func (self *CycleLink) innerRemove(n *node) *node {
	next := n.next

	//剩最后一个并且是头结点则直接删除
	if self.length == 1 && n == self.head {
		//释放空间
		n.next = nil
		n.pre = nil
		self.length--
		return nil
	}

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
	return next

}

/**
* 删除起始Id-->结束id的元素如果endId为-1 则全部删除
* 如果starId没有出现在则从头结点开始删除
* 带有skip过滤器形式的删除
**/
func (self *CycleLink) Remove(startId uint32, endId uint32, filter func(id uint32, msg *Message) bool) chan *Message {

	self.mutex.Lock()
	defer self.mutex.Unlock()
	ch := make(chan *Message, 100)
	start, ok_h := self.hash[startId]
	end, ok_e := self.hash[endId]
	// //如果endId为0那么就代表清空节点
	if endId == 0 {
		//end为head的pre
		end = self.head.pre
		ok_e = true
	} else if !ok_e {
		//如果不存在这样end 则直接返回
		close(ch)
		return ch
	}

	//如果起始坐标不存在则使用头节点
	if !ok_h {
		start = self.head
	}

	//一个接一个地获取并删除节点，endId为-1
	for n := start; nil != n; {
		next := n
		//如果filter不为空或者skip返回false则认为跳过
		if nil == filter || !filter(n.id, n.msg) {
			//对消息的ttl--
			n.msg.ttl--
			//写入channel 让另一侧重发
			ch <- n.msg
			next = self.innerRemove(n)
		}

		if n == end && endId != 0 {
			break
		}
		n = next

	}
	close(ch)
	return ch
}
