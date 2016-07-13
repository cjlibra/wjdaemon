package main

import (
	"crypto/rand"
	"math/big"
	//"bufio"
	"encoding/hex"
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
	datastr := "ee013632333435363738393031323334353637380040313430393137303120202020202020202020202020202020202020202020000000000000000000000000010105050505050530303cbeca3011f80000000000b9"

	data, err := hex.DecodeString(datastr)
	if err != nil {
		fmt.Println("无法解码datastr")
		return
	}
	var ss string
	go readinfo()
	for i := 1; i <= 50000; i++ {
		ss = fmt.Sprintf("%018d", i)
		copy(data[2:2+18], []byte(ss))

		data[len(data)-1] = CalcChecksum(data, len(data))
		//fmt.Println(hex.EncodeToString(data))
		//continue
		data1 := make([]byte, len(data))
		copy(data1, data)
		go action(data1)

	}

	<-ch

}
func readinfo() {
	for {
		fmt.Println("连接上的数量", badconncount)
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

	rnd, _ := rand.Int(rand.Reader, big.NewInt(10))

	time.Sleep(time.Duration(rnd.Int64()) * 1 * time.Second)
	for {

		time.Sleep(time.Second * 30)
		_, err := conn.Write(data)
		if err != nil {

			return
		}
	}
}
