package main

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	w "h"
	"math/big"
	"strconv"
	"time"
)

func aa() {
	var a [2]byte
	a[0] = 0x11
	a[1] = 0x22
	go b(a)
	a[1] = 0x33
}
func b(a [2]byte) {
	time.Sleep(time.Second * 10)
	fmt.Println(a)

}

type Student struct {
	id   int
	name string
}
type person struct {
	id   int
	name string
}

func ff() {

	vv := "aa"
	fmt.Println(vv[:3])

}
func f() {
	fmt.Println("a")
	ff()
	fmt.Println("b")
	fmt.Println("f")
}

func main() {
	cx := []byte{12, 13, 14}
	cx1 := []byte{12, 13, 14}
	fmt.Println(string(cx) == string(cx1))
	return
	fmt.Println(time.Now().Local())
	return

	defer func() { // 必须要先声明defer，否则不能捕获到panic异常
		fmt.Println("c")
		if err := recover(); err != nil {
			fmt.Println(err) // 这里的err其实就是panic传入的内容，55
			main()

		} else {
			fmt.Println("d")
		}

	}()
	f()

	return
	w.Aaab()
	return
	rnd, _ := rand.Int(rand.Reader, big.NewInt(10))
	fmt.Println(rnd.Int64())
	return
	chc := time.After(3 * time.Second)
	for {
		select {
		default:
			fmt.Println("default")

		case <-chc:
			fmt.Println("after10")
			return

		}

	}
	return
	aa1 := "10"
	num, _ := strconv.ParseInt(aa1, 16, 0)
	fmt.Println(num)

	return
	fmt.Println("hello")

	chan1 := time.After(time.Second * 4)

	fmt.Println("World")
	<-chan1
	fmt.Println("World5555")
	return
	ccc1 := make(chan int, 1)
	ccc1 <- 1
	select {
	case m := <-ccc1:
		fmt.Println(ccc1, m)
	case <-time.After(5 * time.Second):
		fmt.Println("timed out")
	}
	return

	ch1 := make(chan int, 1)
	ch2 := make(chan int, 1)
	go func() {
		//time.Sleep(time.Second * 10)
		ch2 <- 4
	}()
	//time.Sleep(time.Microsecond * 5)
	//ch3 := make(chan int, 1)

	select {
	case <-ch1:
		fmt.Println("ch1 pop one element")
	case <-ch2:
		fmt.Println("ch2 pop one element")
	default:
		fmt.Println("default")
	}

	return

	persons := make(map[int]person)
	persons[0] = person{1, "a"}
	p := persons[0]
	p.id = 2
	persons[0] = p
	fmt.Println(persons)
	return
	s_1 := new(Student)
	s_1.id = 100
	s_1.name = "cat"
	var s_2 Student
	s_2.id = 101
	s_2.name = "cat1"
	fmt.Println(s_1, s_2, s_1 == &s_2)
	return
	var a [3]byte
	var b [3]byte
	var ccc chan int
	ccc <- 1
	a[0] = 0x01
	a[1] = 0x11
	a[2] = 0x22

	b[0] = 0x01
	b[1] = 0x11
	b[2] = 21
	myfunc(b[0:3], 1)
	fmt.Println(b)
	copy(a[:3], b[:3])
	b[2] = 41
	fmt.Println(10 - 10/3*3)
	return
	aa()
	time.Sleep(time.Second * 100)
	return
	v := 1000
	bb := make([]byte, 2)
	binary.LittleEndian.PutUint16(bb, uint16(v))

	fmt.Println(bb[0])
	fmt.Println(bb[1])

	//fmt.Println(bb[2])
	//fmt.Println(bb[3])
	return

	a[0] = 0x01
	a[1] = 0x11
	a[2] = 0x22

	b[0] = 0x01
	b[1] = 0x11
	b[2] = 0x21
	a[2] = a[2] - 1

	c := make([]byte, 10)
	copy(c[0:3], b[:3])
	fmt.Println(a == b)
	fmt.Println(bytes.Equal(b[:3], a[0:3]))

	mydata := []byte("abcd")
	myfunc(mydata, 4)
	fmt.Println(mydata)

}

func myfunc(buffer []byte, n int) {

	buffer[0] = 3
}
