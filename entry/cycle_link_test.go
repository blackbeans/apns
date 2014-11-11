package entry

import (
	"fmt"
	"testing"
)

func BenchmarkInsert(t *testing.B) {

	link := NewCycleLink(3, 10000)
	for i := 0; i < t.N; i++ {
		msg := NewMessage(1, 3, MESSAGE_TYPE_SIMPLE)
		link.Insert(uint32(i), msg)
	}

	ch := make(chan *Message)

	go func() {
		link.Remove(0, 0, ch, func(msg *Message) bool {
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
	msg1 := NewMessage(1, 3, MESSAGE_TYPE_SIMPLE)
	link.Insert(1, msg1)
	msg2 := NewMessage(2, 3, MESSAGE_TYPE_SIMPLE)
	link.Insert(2, msg2)
	msg3 := NewMessage(3, 3, MESSAGE_TYPE_SIMPLE)
	link.Insert(3, msg3)
	msg4 := NewMessage(4, 3, MESSAGE_TYPE_SIMPLE)
	link.Insert(4, msg4)
	fmt.Println("INSERT-----------")
	PrintLink(t, link)

	t.Logf("INSERT NODE |%d\n", link.length)
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

	ch := make(chan *Message)

	go func() {
		link.Remove(1, 2, ch, func(msg *Message) bool {
			return false
		})
	}()

	for {
		tmp := <-ch
		t.Logf("GET REMOVE -------%t\n", tmp)
		if nil == tmp {
			break
		}
	}

	//剩下一个
	if link.length != 2 && link.head.id != 3 {
		t.Logf("REMOVE -----FIRST\t len:%d,head.id:%d----%t\n", link.length, link.head)
		t.Fail()
		return
	}

	fmt.Println("CYCLE-FIRST-----------")
	PrintLink(t, link)

	go func() {
		//删除最后一个
		link.Remove(3, 0, ch, func(msg *Message) bool {
			return false
		})
	}()

	for {
		tmp := <-ch
		t.Logf("GET REMOVE LEFT-------%t|%d\n", tmp, link.length)
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
	link.Insert(5, msg5)

	//---------5的ttl为0 则应该不插入
	if link.length != 0 {
		t.Fail()
		fmt.Printf("CYCLE-INSERT TTL 0|%t\n", link.length)
		return
	}

	//------插入5
	msg5 = NewMessage(5, 3, MESSAGE_TYPE_SIMPLE)
	link.Insert(5, msg5)

	if link.length != 1 {
		t.Fail()
		return
	}

	go func() {
		//删除最后一个
		link.Remove(3, 0, ch, func(msg *Message) bool {
			return false
		})
	}()

	for {
		tmp := <-ch
		t.Logf("GET REMOVE LEFT-------%t\n", tmp)
		if nil == tmp {
			break
		}
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
