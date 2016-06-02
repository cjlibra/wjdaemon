package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/golang/glog"
	"gopkg.in/mgo.v2"
	//"gopkg.in/mgo.v2/bson"
)

type Person struct {
	Name  string
	Phone string
	Achar [2]byte
}

func dbopen() *mgo.Session {
	session, err := mgo.Dial("202.127.26.247")
	if err != nil {
		panic(err)
	}
	return session

}
func dbInsertheart(StrucPack PackageStruct) {
	session := dbopen()
	//defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	//session.SetMode(mgo.Monotonic, true)
	/*
		c := session.DB("test").C("people")
		err = c.Insert(&Person{"Ale", "+55 53 8116 9639", [2]byte{0x64, 0x63}},
			&Person{"Cla", "+55 53 8402 8510", [2]byte{0x98, 0x97}})
		if err != nil {
			log.Fatal(err)
		}

		//result := Person{}
		var result1 []Person
		err = c.Find(bson.M{"name": "Ale"}).All(&result1)
		if err != nil {
			log.Fatal(err)
		}

		for i, value := range result1 {
			fmt.Println("Phone:", value.Phone, i, value.Achar[0])
		}
	*/
	c := session.DB("heart").C("info")
	err := c.Insert(&StrucPack)
	if err != nil {
		log.Fatal(err)
	}
}

func SocketServer(sockport string) {

	netListen, err := net.Listen("tcp", ":"+sockport)
	CheckError(err)

	defer netListen.Close()

	Log("后台服务启动于端口 " + sockport)
	for {
		conn, err := netListen.Accept()
		if err != nil {
			continue
		}

		Log(conn.RemoteAddr().String(), " tcp connect success")
		go handleConnection(conn)
	}

}
func foundserialinpoolbynum(serialnum []byte) int{
	for index ,value := range linesinfos {
		if value.FirmSerialno == serialnum[:6] {
			return index
		}
	}
	return -1
}
func getparmfromfrontafter(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	FirmSerial := r.FormValue("FirmSerial")
	if len(r.Form["FirmSerial"]) <= 0 {
		glog.V(3).Infoln("FirmSerial请求参数缺失")
		w.Write([]byte("{status:'1001'}"))
		return
	}
	if len(FirmSerial) <= 0 {
		glog.V(3).Infoln("FirmSerial请求参数内容缺失")
		w.Write([]byte("{status:'1002'}"))
		return
	}
	binFirmSerial, err := hex.DecodeString(FirmSerial)
	if err != nil {
		glog.V(3).Infoln("FirmSerial DecodeString出错")
		w.Write([]byte("{status:'1003'}"))
		return
	}
	if bytes.Equal(oneStrucPack.FirmSerailno[:6] ,binFirmSerial[:6]) != true {
		glog.V(3).Infoln("找不到Firmserialno")
		w.Write([]byte("{status:'1004'}"))
		return
	}
	b, err := json.Marshal(oneStrucPack)
	if err != nil {
		glog.V(2).Infoln("json编码问题" ，err)
		w.Write([]byte("{status:'1005'}"))
		return
	}
	 
	glog.V(2).Infoln(string(b))
	w.Write(b)
	
	
}
func getparmfromfront(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	FirmSerial := r.FormValue("FirmSerial")
	if len(r.Form["FirmSerial"]) <= 0 {
		glog.V(3).Infoln("FirmSerial请求参数缺失")
		w.Write([]byte("{status:'1001'}"))
		return
	}
	if len(FirmSerial) <= 0 {
		glog.V(3).Infoln("FirmSerial请求参数内容缺失")
		w.Write([]byte("{status:'1002'}"))
		return
	}
	binFirmSerial, err := hex.DecodeString(FirmSerial)
	if err != nil {
		glog.V(3).Infoln("FirmSerial DecodeString出错")
		w.Write([]byte("{status:'1003'}"))
		return
	}
	poolgetnum :=foundserialinpoolbynum(binFirmSerial)
	if poolgetnum == -1 {
		glog.V(3).Infoln("客户端未连接上来")
		w.Write([]byte("{status:'1004'}"))
		return
	}
	buffer_getparm := make([]byte , 1024)
	buffer_getparm[0] = 0xEE
	buffer_getparm[1] = 0x80
	copy(buffer_getparm[2:2+6] , binFirmSerial[:6])
	buffer_getparm[9] = 0
	buffer_getparm[10] = CalcChecksum(buffer_getparm,11)
	
	
	sendcount ,err := linesinfos[poolgetnum].Conn.Write(buffer_getparm[:11])
	if err != nil {
		glog.V(3).Infoln("无法发送0x80数据包" ，buffer_getparm[:11])
		w.Write([]byte("{status:'1005'}"))
		return
	}
	

    glog.V(3).Infoln("成功发送0x80数据包" ，buffer_getparm[:11])
	w.Write([]byte("{status:'0'}"))
	return
}

func setparmtofrontafter(w http.ResponseWriter, r *http.Request) {
	
	
}

func setparmtofront(w http.ResponseWriter, r *http.Request) {
	
	
}
func main() {

	sockport := flag.Int("p", 48080, "socket server port")
	webport := flag.Int("wp", 58080, "socket server port")
	flag.Parse()

	go SocketServer(fmt.Sprintf("%d", *sockport))

    http.HandleFunc("/getparmfromfrontafter", getparmfromfrontafter)
	http.HandleFunc("/getparmfromfront", getparmfromfront)
	
	http.HandleFunc("/setparmtofrontafter", setparmtofrontafter)
	http.HandleFunc("/setparmtofront", setparmtofront)
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./htmlsrc/"))))

	glog.Info("WEB程序启动，开始监听" + fmt.Sprintf("%d", *webport) + "端口")
	err := http.ListenAndServe(":"+fmt.Sprintf("%d", *webport), nil)
	if err != nil {

		glog.Info("ListenAndServer: ", err)

	}
}
func CalcChecksum(buffer []byte, n int) byte {
	var tmp byte
	tmp = 0
	for i := 0; i < n-1; i++ {
		tmp = tmp + buffer[i]
	}
	return tmp

}
func IsEqualChecksum(buffer []byte, n int) int {
	if buffer[n-1] == CalcChecksum(buffer, n) {
		return 0
	}

	return 1
}

type PackageStruct struct {
	CMDchar          byte
	FirmSerailno     string
	EquipID          string
	PhoneNum         string
	SysEnergy        byte
	ConnStatus       byte
	ReadWriterStatus byte
	FirmVersion      string
	SoftVersion      string
	OutsideAntena    byte
	InsideAntena     byte
	ServerIpPort     string
	OtherStatus      string
	//CloseBit         byte
}
var oneStrucPack PackageStruct
func DealWithParmGet(buffer []byte,n int) int {
	CMDchar := buffer[1]
	if CMDchar != 0x00 {
		return 1
	}
	
	
	oneStrucPack.CMDchar = buffer[1]
	oneStrucPack.FirmSerailno = string(buffer[2 : 2+6])
	oneStrucPack.EquipID = string(buffer[10 : 10+30])
	oneStrucPack.PhoneNum = string(buffer[40 : 40+11])
	oneStrucPack.SysEnergy = buffer[40+11]
	oneStrucPack.ConnStatus = buffer[40+11+1]
	oneStrucPack.ReadWriterStatus = buffer[53]
	oneStrucPack.FirmVersion = string(buffer[54 : 54+3])
	oneStrucPack.SoftVersion = string(buffer[57 : 57+3])
	oneStrucPack.OutsideAntena = buffer[60]
	oneStrucPack.InsideAntena = buffer[61]
	oneStrucPack.ServerIpPort = string(buffer[62 : 62+6])
	oneStrucPack.OtherStatus = string(buffer[68 : 68+5])
	
	
	return 0
	
}
func DealWithBeatHeart(buffer []byte, n int) int {
	CMDchar := buffer[1]
	if CMDchar != 0x01 {
		return 1
	}
	var StrucPack PackageStruct
	StrucPack.CMDchar = buffer[1]
	StrucPack.FirmSerailno = string(buffer[2 : 2+6])
	StrucPack.EquipID = string(buffer[10 : 10+30])
	StrucPack.PhoneNum = string(buffer[40 : 40+11])
	StrucPack.SysEnergy = buffer[40+11]
	StrucPack.ConnStatus = buffer[40+11+1]
	StrucPack.ReadWriterStatus = buffer[53]
	StrucPack.FirmVersion = string(buffer[54 : 54+3])
	StrucPack.SoftVersion = string(buffer[57 : 57+3])
	StrucPack.OutsideAntena = buffer[60]
	StrucPack.InsideAntena = buffer[61]
	StrucPack.ServerIpPort = string(buffer[62 : 62+6])
	StrucPack.OtherStatus = string(buffer[68 : 68+5])

	dbInsertheart(StrucPack)
	return 0
}
type CONNINFO struct {
	Conn net.Conn
	FirmSerialno [6]byte
	ClientIp string
	Clientport int	
	
}

var linesinfos []CONNINFO
func isfoundserialinpool(buffer []byte) (int ,int) {
	for index , value := range linesinfos {
		if value.FirmSerialno == buffer[2:2+6] {
			return (0 , index)
		}
	}
	return (1 , -1)
}
func handleConnection(conn net.Conn) {
	var onelineinfo CONNINFO
	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			Log(conn.RemoteAddr().String(), "连接出错: ", err)
			return
		}
		
		Log(conn.RemoteAddr().String(), "receive data length:", n)
		Log(conn.RemoteAddr().String(), "receive data:", buffer[:n])
		Log(conn.RemoteAddr().String(), "receive data string:", string(buffer[:n]))
		
		ret , index := isfoundserialinpool(buffer)
		if  ret!= 0 {
			onelineinfo.Conn = conn
			onelineinfo.FirmSerialno = buffer[2:2+6]
			append(linesinfos,onelineinfo)
		}

		if IsEqualChecksum(buffer, n) != 0 {
			continue
		}
		DealWithBeatHeart(buffer, n)
		DealWithParmGet(buffer,n)

	}
}

func Log(v ...interface{}) {
	glog.Info(v...)
}

func CheckError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
