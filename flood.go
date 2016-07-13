package main

import (
	//"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"time"
)

var badconncount int
var badconncount1 int
var badconncount2 int

func main() {
	badconncount = 0
	badconncount1 = 0
	badconncount2 = 0
	ch := make(chan int)
	datastr := "ee01363233343536373800403134303931373031202020202020202020202020202020202020202020200000000105050505050530303cbeca3011f80000000000b9"
	data, err := hex.DecodeString(datastr)
	if err != nil {
		fmt.Println("无法解码datastr")
		return
	}

	for i := 0; i < 50000; i++ {
		go action(data)
		if i%1000 == 0 {
			//time.Sleep(time.Second * 1)
		}
	}
	<-ch

}

func action(data []byte) {
	conn, err := net.Dial("tcp", "202.127.26.244:48080")
	if err != nil {
		badconncount1 = badconncount1 + 1
		fmt.Println("拨不上的数量", badconncount1, err)
		return
	}
	badconncount2 = badconncount2 + 1
	fmt.Println("连接上的数量", badconncount2)
	defer func() {
		conn.Close()
		badconncount = badconncount + 1
		fmt.Println("连接断的数量", badconncount, err)

	}()
	for {

		time.Sleep(time.Second * 30)
		_, err := conn.Write(data)
		if err != nil {

			return
		}
	}
}
