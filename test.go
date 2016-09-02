package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	//w "h"
	"math/big"
	"strconv"
	"time"

	"github.com/golang/glog"
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
func crc8(cmdBuf []byte) byte {
	bufLen := len(cmdBuf)
	glog.V(2).Infoln("bufLen:", bufLen, hex.EncodeToString(cmdBuf))
	CRC8_Table := []byte{
		0, 94, 188, 226, 97, 63, 221, 131, 194, 156, 126, 32, 163, 253, 31, 65,
		157, 195, 33, 127, 252, 162, 64, 30, 95, 1, 227, 189, 62, 96, 130, 220,
		35, 125, 159, 193, 66, 28, 254, 160, 225, 191, 93, 3, 128, 222, 60, 98,
		190, 224, 2, 92, 223, 129, 99, 61, 124, 34, 192, 158, 29, 67, 161, 255,
		70, 24, 250, 164, 39, 121, 155, 197, 132, 218, 56, 102, 229, 187, 89, 7,
		219, 133, 103, 57, 186, 228, 6, 88, 25, 71, 165, 251, 120, 38, 196, 154,
		101, 59, 217, 135, 4, 90, 184, 230, 167, 249, 27, 69, 198, 152, 122, 36,
		248, 166, 68, 26, 153, 199, 37, 123, 58, 100, 134, 216, 91, 5, 231, 185,
		140, 210, 48, 110, 237, 179, 81, 15, 78, 16, 242, 172, 47, 113, 147, 205,
		17, 79, 173, 243, 112, 46, 204, 146, 211, 141, 111, 49, 178, 236, 14, 80,
		175, 241, 19, 77, 206, 144, 114, 44, 109, 51, 209, 143, 12, 82, 176, 238,
		50, 108, 142, 208, 83, 13, 239, 177, 240, 174, 76, 18, 145, 207, 45, 115,
		202, 148, 118, 40, 171, 245, 23, 73, 8, 86, 180, 234, 105, 55, 213, 139,
		87, 9, 235, 181, 54, 104, 138, 212, 149, 203, 41, 119, 244, 170, 72, 22,
		233, 183, 85, 11, 136, 214, 52, 106, 43, 117, 151, 201, 74, 20, 246, 168,
		116, 42, 200, 150, 21, 75, 169, 247, 182, 232, 10, 84, 215, 137, 107, 53}
	var crc8 byte = 0

	for i := 0; i < bufLen; i++ {
		crc8 = CRC8_Table[crc8^cmdBuf[i]]
		//查表得到CRC码
		//cmdBuf1 = cmdBuf[i:]
	}

	return crc8
}

func main() {
	sockport := "6767"
	netListen, err := net.Listen("tcp", ":"+sockport)
	if err != nil {
		glog.V(1).Infoln("Listen出错，可能端口占用：", sockport)
		return
	}

	defer netListen.Close()

	glog.Infoln("后台服务启动于端口 " + sockport)

	conn, err := netListen.Accept()
	if err != nil {
		glog.Infoln("netListen.Accept() err ")
		return
	}

	glog.Infoln(conn.RemoteAddr().String(), "->", "TCP连接成功")
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	for {

		n, err := conn.Read(buffer)
		//	conn.SetReadDeadline(time.Time{})
		if err != nil {
			glog.Infoln(conn.RemoteAddr().String(), "连接出错: ", err)
			continue
		}
		glog.Infoln("this is：", buffer[:n], n)
	}

	return
	type OUTUPDATE struct {
		FirmSerial     [18]byte
		Procedure      int
		FirmFileCount  int
		AllFramesCount int
		PartPercent    int
		WholeChecksum  byte
		DoTime         time.Time
	}
	var stroutupdate OUTUPDATE
	fmt.Println(stroutupdate)
	return
	az1 := time.Now()
	//az2, _ := time.Parse("2006-01-02 15:04:05", "2016-08-30 12:00:00")
	az2, _ := time.ParseInLocation("2006-01-02 15:04:05", "2016-08-30 12:00:00", time.Local)
	fmt.Println(az1)
	fmt.Println(az2)
	//az2 = az2.Local()
	//fmt.Println(az2)
	fmt.Println(az1.Sub(az2).Hours())
	//fmt.Println(az1.Add(-az2).String())
	return
	var n1 time.Duration = 8
	fmt.Println(n1.Nanoseconds(), time.Second.Nanoseconds())
	time.Sleep((8 * 1000000000))
	return
	//ss := "aa97aaffbc1843ff02e588e6d159e5891991581aee"
	ss1 := "aa9701e58972ea5a3dee"
	ss1 = "aa9701e589e6f652e8ee"
	bcb, _ := hex.DecodeString(ss1)
	fmt.Println(crc8(bcb[1 : len(bcb)-2]))
	return
	aass := "as"
	fmt.Println(fmt.Sprintf("%018s", aass))
	return
	s := "sha1 this string"

	h := sha1.New()

	h.Write([]byte(s))

	bs := h.Sum(nil)

	fmt.Println(s)
	fmt.Printf("%x\n", bs)

	return
	//hs := sha1.New()
	//io.Copy(hs, file)
	//hashString := hs.Sum(nil)

	return
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
	//w.Aaab()
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
