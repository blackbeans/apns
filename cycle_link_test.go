package apns

import (
	"fmt"
	"testing"
)

func BenchmarkInsert(t *testing.B) {

	link := NewCycleLink(3, 10000)
	for i := 0; i < t.N; i++ {
		msg := NewMessage(1, 3, MESSAGE_TYPE_SIMPLE)
		msg.IdentifierId = link.Insert(msg)
	}

	ch := make(chan *Message)

	go func() {
		link.Remove(0, 0, ch, func(id uint32, msg *Message) bool {
			return false
		})
	}()

	for {
		tmp := <-ch
		// t.Logf("GET REMOVE -------%t\n", tmp)
		if nil == tmp {
			close(ch)
			break
		}
	}

	t.Log("HEADER---------\n")

	// PrintLink(t, link)
}

func TestCycleLink(t *testing.T) {
	link := NewCycleLink(3, 3)
	msg1 := NewMessage(2, 3, MESSAGE_TYPE_SIMPLE)
	msg1.IdentifierId = link.Insert(msg1)
	msg2 := NewMessage(2, 3, MESSAGE_TYPE_SIMPLE)
	msg2.IdentifierId = link.Insert(msg2)
	msg3 := NewMessage(2, 3, MESSAGE_TYPE_SIMPLE)
	msg3.IdentifierId = link.Insert(msg3)
	msg4 := NewMessage(2, 3, MESSAGE_TYPE_SIMPLE)
	msg4.IdentifierId = link.Insert(msg4)
	fmt.Println("INSERT-----------")
	PrintLink(t, link)

	fmt.Printf("INSERT NODE |%d\n", link.length)
	if link.length != 3 {
		t.Fail()
		t.Logf("INSERT NODE FAIL|%d\n", link.length)
		return
	}

	if link.head.id != 2 {
		t.Fail()
		t.Logf("INSERT NODE HEAD IS NOT %d\n|%t\n", 2, link.head)
		return
	}

	//--------------测试过滤条件为true时删除

	ch := make(chan *Message, 2)
	go link.Remove(0, 0, ch, func(id uint32, msg *Message) bool {
		return true
	})
	//都过滤一个都不会删除
	for {
		tmp, ok := <-ch
		//只要pop出来一个不为空的或者ok还存活的那么就是错误的
		if nil != tmp || ok {
			t.Fail()
			break
		} else {
			fmt.Printf("Remove|Filter-----------%s|%t\n", tmp, ok)
			break
		}
	}

	ch = make(chan *Message)
	go link.Remove(0, 2, ch, func(id uint32, msg *Message) bool {
		return false
	})
	//删除的是2
	for {
		tmp := <-ch
		if nil == tmp {
			break
		}
		fmt.Printf("Remove-----------%d\n", tmp.IdentifierId)
	}

	//剩下3、4
	if link.length != 2 && link.head.id != 3 {
		fmt.Printf("REMOVE -----FIRST\t len:%d,head.id:%d----%t\n", link.length, link.head)
		t.Fail()
		return
	}

	fmt.Println("CYCLE-FIRST-----------")
	PrintLink(t, link)

	ch = make(chan *Message, 10)
	go link.Remove(0, 0, ch, func(id uint32, msg *Message) bool {
		fmt.Println("------------")
		return false
	})
	//全部删除了
	for {
		fmt.Println("GET REMOVE LEFT_Read")
		tmp := <-ch
		if nil == tmp {
			break
		}
	}

	fmt.Printf("CYCLE-LEFT-----------%d|%t\n", link.length, link)

	if link.length > 0 {
		PrintLink(t, link)
		t.Fail()
	} else if link.length != 0 {
		fmt.Printf("CYCLE-LEFT NULL FAIL|%d\n", link.length)
	}

	msg5 := NewMessage(5, 0, MESSAGE_TYPE_SIMPLE)
	msg5.IdentifierId = link.Insert(msg5)

	//---------5的ttl为0 则应该不插入
	if link.length != 0 {
		t.Fail()
		fmt.Printf("CYCLE-INSERT TTL 0|%t\n", link.length)
		return
	}

	//------插入5
	msg5 = NewMessage(5, 3, MESSAGE_TYPE_SIMPLE)
	msg5.IdentifierId = link.Insert(msg5)

	if link.length != 1 {
		t.Fail()
		return
	}

	ch = make(chan *Message, 10)
	go link.Remove(0, 0, ch, func(id uint32, msg *Message) bool {
		return false
	})

	for {
		tmp := <-ch
		if nil == tmp {
			break
		}
		t.Logf("GET REMOVE LEFT-------%d\n", tmp.IdentifierId)
	}

}

func PrintLink(t testing.TB, link *CycleLink) {
	h := link.head

	for {
		fmt.Printf("next---------%d\n", h)
		if nil == h {
			break
		}
		h = h.next
		if nil == h || (link.head == h) {
			break
		}
	}

	h = link.head

	for {
		fmt.Printf("pre---------%d\n", h)
		if nil == h {
			break
		}
		h = h.pre
		if nil == h {
			break
		} else {

			if link.head == h {
				break
			}
		}
	}
}
