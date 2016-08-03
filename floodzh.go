package main

import (
	//"crypto/rand"
	//	"math/big"
	//"bufio"
	//"encoding/hex"
	"fmt"
	"net"
	"time"
)

var badconncount int

func CalcChecksum(buffer []byte, n int) byte {
	var tmp byte
	tmp = 0
	for i := 0; i < n-1; i++ {
		tmp = tmp + buffer[i]
	}
	return tmp

}
func main() {
	badconncount = 0

	ch := make(chan int)
	//datastr := "ee013632333435363738393031323334353637380040313430393137303120202020202020202020202020202020202020202020000000000000000000000000010105050505050530303cbeca3011f80000000000b9"

	//data, err := hex.DecodeString(datastr)

	data := []byte{2, 3, 4, 5, 0, 25, 0, 22, 65, 135, 1, 1, 90, 5, 88, 24, 161, 121, 80, 255, 31, 49, 196, 68, 151}
	//if err != nil {
	//	fmt.Println("无法解码datastr")
	//	return
	//}
	go readinfo()
	for i := 0; i < 10000; i++ {
		go action(data)
	}

	<-ch

}
func readinfo() {
	for {
		fmt.Println("连接上的数量读取", badconncount)
		time.Sleep(time.Second * 30)
	}
}

func action(data []byte) {
	conn, err := net.Dial("tcp", "202.127.26.244:48080")
	if err != nil {

		return
	}
	badconncount = badconncount + 1
	fmt.Println("连接上的数量", badconncount)
	defer func() {
		conn.Close()
		badconncount = badconncount - 1

	}()

	for {

		time.Sleep(time.Second * 10)
		_, err := conn.Write(data)
		if err != nil {

			return
		}
	}
}
