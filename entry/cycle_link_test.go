package entry

import (
	"fmt"
	"testing"
)

func TestCycleLink(t *testing.T) {
	link := NewCycleLink(3, 4)
	msg1 := NewMessage(1, 3)
	link.Insert(1, msg1)
	msg2 := NewMessage(2, 3)
	link.Insert(2, msg2)
	msg3 := NewMessage(3, 3)
	link.Insert(3, msg3)
	msg4 := NewMessage(4, 3)
	link.Insert(4, msg4)
	fmt.Println("INSERT-----------")
	PrintLink(t, link)

	t.Logf("INSERT NODE |%d\n", link.length)
	if link.length != 4 {
		t.Fail()
		t.Logf("INSERT NODE FAIL|%d\n", link.length)
		return
	}

	ch := make(chan *Message)

	go func() {
		link.Remove(1, 2, ch)
	}()

	for {
		tmp := <-ch
		t.Logf("GET REMOVE -------%t\n", tmp)
		if nil == tmp {
			break
		}
	}

	//剩下一个
	if link.length != 3 && link.head.id != 2 {
		t.Logf("REMOVE -----FIRST\t len:%d,head.id:%d----%t\n", link.length, link.head.id)
		t.Fail()
		return
	}

	fmt.Println("CYCLE-FIRST-----------")
	PrintLink(t, link)

	go func() {
		//删除最后一个
		link.Remove(2, -1, ch)
	}()

	for {
		tmp := <-ch
		t.Logf("GET REMOVE LEFT-------%t\n", tmp)
		if nil == tmp {
			break
		}
	}

	fmt.Println("CYCLE-LEFT-----------")

	if link.length > 0 {
		PrintLink(t, link)
		t.Fail()
	} else {
		fmt.Println("CYCLE-LEFT NULL SUCC-----------")
	}

}

func PrintLink(t *testing.T, link *CycleLink) {
	h := link.head
	for {

		fmt.Printf("next---------%d\n", h.id)
		h = h.next
		if nil == h || (link.head == h) {
			break
		}
	}

	h = link.head

	for {
		h = h.pre

		if nil == h {
			break
		} else {
			fmt.Printf("pre---------%d\n", h.id)
			if link.head == h {
				break
			}
		}
	}
}
