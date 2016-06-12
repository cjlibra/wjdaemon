package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"mime/multipart"
	"strconv"

	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

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
func foundserialinpoolbynum(serialnum []byte) int {
	for index, value := range linesinfos {
		if bytes.Equal(value.FirmSerialno[:6], serialnum[:6]) == true {
			return index
		}
	}
	return -1
}

type UPDATETASK struct {
	FirmSerial     [6]byte
	Procedure      int
	FirmFileCount  int
	FirmFileBuf    []byte
	AllFramesCount int
	PartPercent    int
	WholeChecksum  byte
	DoTime         time.Time
	ReportChan     chan int
}

// 获取文件大小的接口
type Size interface {
	Size() int64
}

var updatefirmtasks []UPDATETASK

func searchtask(FirmSerial []byte) int {
	for index, value := range updatefirmtasks {
		if bytes.Equal(value.FirmSerial[:6], FirmSerial[:6]) == true {
			return index
		}
	}

	return -1
}
func ReadFromStdFile(file multipart.File, firmbuf []byte) (int, error) {
	br := bufio.NewReader(file)
	bb := []byte("00")
	firmbufcount := 0
	for {
		//每次读取一行
		line, err := br.ReadString('\n')
		if len(line) <= 8 {
			glog.V(3).Infoln("读取文件出错,行太短", line)
			return 0, err
		}
		if err == io.EOF {
			break
		} else {
			glog.V(3).Infoln("读取文件出错", line)
			return 0, err
		}
		a0, _ := strconv.Atoi(line[0:1])
		a1, _ := strconv.Atoi(line[1:2])
		aa := a0*16 + a1
		if bytes.Equal([]byte(line[6:6+2]), bb[:2]) != true {
			continue
		}
		copy(firmbuf[firmbufcount:firmbufcount+aa], line[8:8+aa])
		firmbufcount = firmbufcount + aa

	}
	return firmbufcount, nil
}
func updatefirm(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	//r.ParseForm()
	FirmSerial := r.FormValue("FirmSerial")
	if len(r.Form["FirmSerial"]) <= 0 {
		glog.V(3).Infoln("FirmSerial请求参数缺失")
		w.Write([]byte("{status:'1001'}"))
		return
	}
	if len(FirmSerial) != 6 {
		glog.V(3).Infoln("FirmSerial请求参数内容缺失")
		w.Write([]byte("{status:'1002'}"))
		return
	}
	binFirmSerial := []byte(FirmSerial)

	if "POST" != r.Method {
		w.Write([]byte("{status:'1004'}"))
		return
	}
	file, _, err := r.FormFile("firmfile")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer file.Close()
	firmbuf := make([]byte, 1024*1024)
	ReadFromStdFile(file, firmbuf)
	firmnum, err := file.Read(firmbuf)
	if err != nil {
		glog.V(3).Infoln("Firm文件读取失败")
		w.Write([]byte("{status:'1005'}"))
		return
	}
	var oneupdatefirmtask UPDATETASK
	oneupdatefirmtask.FirmFileBuf = firmbuf

	copy(oneupdatefirmtask.FirmSerial[:6], binFirmSerial[:6])
	oneupdatefirmtask.Procedure = 1
	if sizeInterface, ok := file.(Size); ok {
		oneupdatefirmtask.FirmFileCount = int(sizeInterface.Size())
	}
	oneupdatefirmtask.WholeChecksum = CalcChecksum(firmbuf, firmnum+1)
	oneupdatefirmtask.PartPercent = 0
	oneupdatefirmtask.DoTime = time.Now().Local()
	oneupdatefirmtask.AllFramesCount = oneupdatefirmtask.FirmFileCount / CountInPerFrame
	no := searchtask(binFirmSerial)
	if no == -1 {
		updatefirmtasks = append(updatefirmtasks, oneupdatefirmtask)
	} else {
		updatefirmtasks[no].DoTime = time.Now().Local()
		updatefirmtasks[no].FirmFileCount = oneupdatefirmtask.FirmFileCount
		updatefirmtasks[no].PartPercent = oneupdatefirmtask.PartPercent
		updatefirmtasks[no].Procedure = oneupdatefirmtask.Procedure
		updatefirmtasks[no].WholeChecksum = oneupdatefirmtask.WholeChecksum
		updatefirmtasks[no].FirmFileBuf = oneupdatefirmtask.FirmFileBuf
		updatefirmtasks[no].AllFramesCount = oneupdatefirmtask.AllFramesCount
	}
	poolgetnum := foundserialinpoolbynum(binFirmSerial)
	if poolgetnum == -1 {
		glog.V(3).Infoln("客户端未连接上来")
		w.Write([]byte("{status:'1004'}"))
		return
	}
	framenum := 0
	framenum = firmnum / CountInPerFrame
	btmp := make([]byte, 2)
	binary.LittleEndian.PutUint16(btmp, uint16(framenum))

	buffer_updateparm := make([]byte, 256)
	buffer_updateparm[0] = 0xEE
	buffer_updateparm[1] = 0x83
	copy(buffer_updateparm[2:2+6], binFirmSerial[:6])
	buffer_updateparm[8] = 0
	buffer_updateparm[9] = 7
	buffer_updateparm[10] = btmp[1]
	buffer_updateparm[11] = btmp[0]
	buffer_updateparm[12] = CalcChecksum(firmbuf, firmnum+1)
	buffer_updateparm[13] = 0
	buffer_updateparm[14] = 0
	buffer_updateparm[15] = 0
	buffer_updateparm[16] = 0
	buffer_updateparm[17] = 0 //close
	buffer_updateparm[18] = CalcChecksum(buffer_updateparm, 19)

	sendcount, err := linesinfos[poolgetnum].Conn.Write(buffer_updateparm[:19])
	if err != nil {
		glog.V(3).Infoln("无法发送0x83数据包", buffer_updateparm[:19], sendcount)
		w.Write([]byte("{status:'1005'}"))
		return
	}

	go updatefirmstart(poolgetnum, no)
	glog.V(3).Infoln("成功发送0x83数据包", buffer_updateparm[:19], sendcount)
	w.Write([]byte("{status:'0'}"))
	return
}
func updatefirmstart(poolgetnum int, no int) {
	rp := <-updatefirmtasks[no].ReportChan
	if rp < 0 {
		return
	}

	buffer_update := make([]byte, 256)
	FrameCount := updatefirmtasks[no].AllFramesCount + 1

	addfilesize := 0
	var SizeinPerPack uint16 = 0
	var SizeinWholePack uint16 = 0
	btmp := make([]byte, 2)

	for i := 1; i <= FrameCount; i++ {

		addfilesize = i * CountInPerFrame
		if addfilesize >= updatefirmtasks[no].FirmFileCount {
			SizeinPerPack = uint16(updatefirmtasks[no].FirmFileCount - (addfilesize - CountInPerFrame))
		} else {
			SizeinPerPack = uint16(CountInPerFrame)
		}

		SizeinWholePack = SizeinPerPack + 5

		buffer_update[0] = 0xEE
		buffer_update[1] = 0x84
		copy(buffer_update[2:2+6], updatefirmtasks[no].FirmSerial[:6])
		binary.LittleEndian.PutUint16(btmp, SizeinWholePack)
		buffer_update[8] = btmp[1]
		buffer_update[9] = btmp[0]
		binary.LittleEndian.PutUint16(btmp, uint16(i))
		buffer_update[10] = btmp[1]
		buffer_update[11] = btmp[0]
		binary.LittleEndian.PutUint16(btmp, uint16(SizeinPerPack))
		buffer_update[12] = btmp[1]
		buffer_update[13] = btmp[0]
		copy(buffer_update[14:14+SizeinPerPack], updatefirmtasks[no].FirmFileBuf[(i-1)*CountInPerFrame:(i-1)*CountInPerFrame+int(SizeinPerPack)])
		buffer_update[14+SizeinPerPack] = CalcChecksum(buffer_update[10:10+4+SizeinPerPack], 4+int(SizeinPerPack)+1)
		buffer_update[14+SizeinPerPack+1] = 0 //close bit
		buffer_update[14+SizeinPerPack+2] = CalcChecksum(buffer_update[0:], 14+int(SizeinPerPack)+2+1)
		sendcount, err := linesinfos[poolgetnum].Conn.Write(buffer_update[:14+SizeinPerPack+2+1])
		if err != nil {
			glog.V(3).Infoln("无法发送0x84数据包", buffer_update[:14+SizeinPerPack+2+1], sendcount)

			return
		}
		updatefirmtasks[no].Procedure = 3

		rp := <-updatefirmtasks[no].ReportChan
		if rp < 0 {
			return
		}
		if rp == i {
			i = i - 1
			continue
		}

	}

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
	binFirmSerial := []byte(FirmSerial)

	if bytes.Equal([]byte(oneStrucPack.FirmSerailno[:6]), binFirmSerial[:6]) != true {
		glog.V(3).Infoln("找不到Firmserialno")
		w.Write([]byte("{status:'1004'}"))
		return
	}
	b, err := json.Marshal(oneStrucPack)
	if err != nil {
		glog.V(2).Infoln("json编码问题", err)
		w.Write([]byte("{status:'1005'}"))
		return
	}

	glog.V(2).Infoln(string(b))
	w.Write(b)

}
func getparmfromfront(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	if len(r.Form["FirmSerial"]) <= 0 {
		glog.V(3).Infoln("FirmSerial请求参数缺失")
		w.Write([]byte("{status:'1001'}"))
		return
	}
	FirmSerial := r.FormValue("FirmSerial")
	if len(FirmSerial) < 6 {
		glog.V(3).Infoln("FirmSerial请求参数内容缺失")
		w.Write([]byte("{status:'1002'}"))
		return
	}
	binFirmSerial := []byte(FirmSerial)

	glog.V(3).Infoln(binFirmSerial)
	poolgetnum := foundserialinpoolbynum(binFirmSerial)
	if poolgetnum == -1 {
		glog.V(3).Infoln("客户端未连接上来")
		w.Write([]byte("{status:'1004'}"))
		return
	}
	buffer_getparm := make([]byte, 1024)
	buffer_getparm[0] = 0xEE
	buffer_getparm[1] = 0x80
	copy(buffer_getparm[2:2+6], binFirmSerial[:6])
	buffer_getparm[8] = 0
	buffer_getparm[9] = 0
	buffer_getparm[10] = CalcChecksum(buffer_getparm, 11)

	sendcount, err := linesinfos[poolgetnum].Conn.Write(buffer_getparm[:11])
	if err != nil {
		glog.V(3).Infoln("无法发送0x80数据包", buffer_getparm[:11], sendcount)
		w.Write([]byte("{status:'1005'}"))
		return
	}

	glog.V(3).Infoln("成功发送0x80数据包", buffer_getparm[:11])
	w.Write([]byte("{status:'0'}"))
	return
}
func updatefirmafter(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(updatefirmtasks)
	if err != nil {
		glog.V(2).Infoln("json编码问题updatefirmtasks", err)
		w.Write([]byte("{status:'1001'}"))
		return
	}

	glog.V(2).Infoln(string(b))
	w.Write(b)
}
func setparmtofrontafter(w http.ResponseWriter, r *http.Request) {
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
	binFirmSerial := []byte(FirmSerial)

	if bytes.Equal([]byte(secondStrucPack.FirmSerailno[:6]), binFirmSerial[:6]) != true {
		glog.V(3).Infoln("reponse pool找不到Firmserialno")
		w.Write([]byte("{status:'1004'}"))
		return
	}
	b, err := json.Marshal(secondStrucPack)
	if err != nil {
		glog.V(2).Infoln("json编码问题", err)
		w.Write([]byte("{status:'1005'}"))
		return
	}

	glog.V(2).Infoln(string(b))
	w.Write(b)

}

func setparmtofront(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	FirmSerial := r.FormValue("FirmSerial")
	Maskset := r.FormValue("Maskset")
	Outantenaset := r.FormValue("Outantenaset")
	Inantenaset := r.FormValue("Inantenaset")
	Monswitchset := r.FormValue("Monswitchset")
	Sysresetset := r.FormValue("Sysresetset")
	Defaultbackset := r.FormValue("Defaultbackset")
	Otherset := r.FormValue("Otherset")

	if len(r.Form["FirmSerial"]) <= 0 ||
		len(r.Form["Maskset"]) <= 0 ||
		len(r.Form["Outantenaset"]) <= 0 ||
		len(r.Form["Inantenaset"]) <= 0 ||
		len(r.Form["Monswitchset"]) <= 0 ||
		len(r.Form["Sysresetset"]) <= 0 ||
		len(r.Form["Defaultbackset"]) <= 0 ||
		len(r.Form["Otherset"]) <= 0 {

		glog.V(3).Infoln("setparmtofront请求参数缺失")
		w.Write([]byte("{status:'1001'}"))
		return
	}
	if len(FirmSerial) != 6 ||
		len(Maskset) != 2 ||
		len(Outantenaset) != 2 ||
		len(Inantenaset) != 2 ||
		len(Monswitchset) != 2 ||
		len(Sysresetset) != 2 ||
		len(Defaultbackset) != 2 ||
		len(Otherset) != 6 {

		glog.V(3).Infoln("setparmtofront请求参数内容不准确")
		w.Write([]byte("{status:'1002'}"))
		return
	}

	binFirmSerial := []byte(FirmSerial)

	poolgetnum := foundserialinpoolbynum(binFirmSerial)
	if poolgetnum == -1 {
		glog.V(3).Infoln("客户端未连接上来")
		w.Write([]byte("{status:'1004'}"))
		return
	}

	binMaskset, err := hex.DecodeString(Maskset)
	if err != nil {
		glog.V(3).Infoln("Maskset DecodeString出错")
		w.Write([]byte("{status:'1003'}"))
		return
	}
	binOutantenaset, err := hex.DecodeString(Outantenaset)
	if err != nil {
		glog.V(3).Infoln("Outantenaset DecodeString出错")
		w.Write([]byte("{status:'1003'}"))
		return
	}
	binInantenaset, err := hex.DecodeString(Inantenaset)
	if err != nil {
		glog.V(3).Infoln("Inantenaset DecodeString出错")
		w.Write([]byte("{status:'1003'}"))
		return
	}
	binMonswitchset, err := hex.DecodeString(Monswitchset)
	if err != nil {
		glog.V(3).Infoln("Monswitchset DecodeString出错")
		w.Write([]byte("{status:'1003'}"))
		return
	}
	binSysresetset, err := hex.DecodeString(Sysresetset)
	if err != nil {
		glog.V(3).Infoln("Sysresetset DecodeString出错")
		w.Write([]byte("{status:'1003'}"))
		return
	}
	binDefaultbackset, err := hex.DecodeString(Defaultbackset)
	if err != nil {
		glog.V(3).Infoln("Defaultbackset DecodeString出错")
		w.Write([]byte("{status:'1003'}"))
		return
	}
	binOtherset, err := hex.DecodeString(Otherset)
	if err != nil {
		glog.V(3).Infoln("Otherset DecodeString出错")
		w.Write([]byte("{status:'1003'}"))
		return
	}
	buffer_setparm := make([]byte, 1024)
	buffer_setparm[0] = 0xEE
	buffer_setparm[1] = 0x82
	copy(buffer_setparm[2:2+6], binFirmSerial[:6])
	buffer_setparm[8] = 0
	buffer_setparm[9] = 9
	buffer_setparm[10] = binMaskset[0]
	buffer_setparm[11] = binOutantenaset[0]
	buffer_setparm[12] = binInantenaset[0]
	buffer_setparm[13] = binMonswitchset[0]
	buffer_setparm[14] = binSysresetset[0]
	buffer_setparm[15] = binDefaultbackset[0]
	copy(buffer_setparm[16:16+3], binOtherset[:3])
	buffer_setparm[19] = 0
	buffer_setparm[20] = CalcChecksum(buffer_setparm, 21)

	sendcount, err := linesinfos[poolgetnum].Conn.Write(buffer_setparm[:21])
	if err != nil {
		glog.V(3).Infoln("无法发送0x82数据包", hex.EncodeToString(buffer_setparm[:21]), sendcount)
		w.Write([]byte("{status:'1005'}"))
		return
	}

	glog.V(3).Infoln("成功发送0x82数据包", hex.EncodeToString(buffer_setparm[:21]))
	w.Write([]byte("{status:'0'}"))
	return

}

var CountInPerFrame int
var session *mgo.Session
var c *mgo.Collection

func main() {
	CountInPerFrame = 50

	session = dbopen()
	c = session.DB("heart").C("info")
	//defer session.Close()
	sockport := flag.Int("p", 48080, "socket server port")
	webport := flag.Int("wp", 58080, "socket server port")
	flag.Parse()

	go SocketServer(fmt.Sprintf("%d", *sockport))

	http.HandleFunc("/updatefirmafter", updatefirmafter)
	http.HandleFunc("/updatefirm", updatefirm)
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
func DealWithUpdateFirm(buffer []byte, n int) int {
	CMDchar := buffer[1]
	if CMDchar != 0x04 {
		return 1
	}
	FirmSerial := make([]byte, 6)
	copy(FirmSerial[:6], buffer[2:2+6])

	if buffer[15] != CalcChecksum(buffer[14:14+6], 6) {
		return 2
	}
	num := searchtask(FirmSerial)
	if num == -1 {
		return 3
	}
	xuhao := int(buffer[10])*256 + int(buffer[11])
	if buffer[14] != 0 {
		updatefirmtasks[num].ReportChan <- xuhao
	} else {
		updatefirmtasks[num].ReportChan <- (xuhao + 1)
		updatefirmtasks[num].PartPercent = xuhao * 100 / (updatefirmtasks[num].FirmFileCount/CountInPerFrame + 1)
	}

	return 0
}

func DealWithPreUpdateFirm(buffer []byte, n int) int {
	CMDchar := buffer[1]
	if CMDchar != 0x03 {
		return 1
	}
	FirmSerial := make([]byte, 6)
	copy(FirmSerial[:6], buffer[2:2+6])
	allchecksum := buffer[12]

	num := searchtask(FirmSerial)
	if num == -1 {
		return 2
	}

	if updatefirmtasks[num].WholeChecksum == allchecksum && updatefirmtasks[num].Procedure == 1 {
		updatefirmtasks[num].Procedure = 2
		updatefirmtasks[num].ReportChan <- 0

	}

	return 0

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
	Dotime           time.Time
	//CloseBit         byte
}

type PackageStructBySetParm struct {
	CMDchar         byte
	FirmSerailno    string
	ParmSetResponse string
}

var secondStrucPack PackageStructBySetParm

func DealWithParmSetReponse(buffer []byte, n int) int {
	CMDchar := buffer[1]
	if CMDchar != 0x02 {
		return 1
	}
	secondStrucPack.CMDchar = buffer[1]
	secondStrucPack.FirmSerailno = string(buffer[2 : 2+6])
	secondStrucPack.ParmSetResponse = hex.EncodeToString(buffer[10 : 10+9])

	return 0
}

var oneStrucPack PackageStruct

func DealWithParmGet(buffer []byte, n int) int {
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
	oneStrucPack.ServerIpPort = Iphex2string(buffer[62:62+6], 6)
	oneStrucPack.OtherStatus = string(buffer[68 : 68+5])
	oneStrucPack.Dotime = time.Now().Local()

	return 0

}
func Iphex2string(buf []byte, n int) string {
	a1 := fmt.Sprintf("%d", uint8(buf[0]))
	a2 := fmt.Sprintf("%d", uint8(buf[1]))
	a3 := fmt.Sprintf("%d", uint8(buf[2]))
	a4 := fmt.Sprintf("%d", uint8(buf[3]))
	a5 := fmt.Sprintf("%d", uint(buf[4])*256+uint(buf[5]))
	ss := fmt.Sprintf("%s.%s.%s.%s:%s", a1, a2, a3, a4, a5)
	return ss
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
	StrucPack.ServerIpPort = Iphex2string(buffer[62:62+6], 6)
	StrucPack.OtherStatus = string(buffer[68 : 68+5])
	StrucPack.Dotime = time.Now().Local()

	dbInsertheart(StrucPack)

	ret := isfoundserialinpool(buffer)
	if ret == -1 {
		glog.V(3).Infoln("客户端没有连接上")

		return 1
	}
	buffer_heartback := make([]byte, 256)
	buffer_heartback[0] = 0xEE
	buffer_heartback[1] = 0x81
	copy(buffer_heartback[2:2+6], buffer[2:2+6])
	buffer_heartback[8] = 0x00
	buffer_heartback[9] = 0x01
	buffer_heartback[10] = 0x00
	buffer_heartback[11] = CalcChecksum(buffer_heartback, 12)
	sendcount, err := linesinfos[ret].Conn.Write(buffer_heartback[:12])
	if err != nil {
		glog.V(3).Infoln("无法发送心跳返回数据包", hex.EncodeToString(buffer_heartback[:12]), sendcount)

		return 2
	}
	glog.V(3).Infoln("成功发送心跳返回数据包", hex.EncodeToString(buffer_heartback[:12]), sendcount)
	return 0
}

type CONNINFO struct {
	Conn         net.Conn
	FirmSerialno [6]byte
	ClientIp     string
	Clientport   string
	Dotime       time.Time
}

var linesinfos []CONNINFO

func isfoundserialinpool(buffer []byte) int {
	for index, value := range linesinfos {
		if bytes.Equal(value.FirmSerialno[:6], buffer[2:2+6]) == true {
			return index
		}
	}
	return -1
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
		Log(conn.RemoteAddr().String(), "receive data:", hex.EncodeToString(buffer[:n]))
		//Log(conn.RemoteAddr().String(), "receive data string:", string(buffer[:n]))

		ret := isfoundserialinpool(buffer)
		if ret == -1 {
			onelineinfo.Conn = conn
			copy(onelineinfo.FirmSerialno[:6], buffer[2:2+6])
			onelineinfo.ClientIp = conn.RemoteAddr().String()
			onelineinfo.Clientport = strings.Split(conn.RemoteAddr().String(), ":")[1]
			onelineinfo.Dotime = time.Now().Local()
			linesinfos = append(linesinfos, onelineinfo)

		} else {
			linesinfos[ret].Dotime = time.Now().Local()
			linesinfos[ret].Conn = conn
			linesinfos[ret].ClientIp = conn.RemoteAddr().String()
			linesinfos[ret].Clientport = strings.Split(conn.RemoteAddr().String(), ":")[1]
		}

		if IsEqualChecksum(buffer, n) != 0 {
			continue
		}
		DealWithBeatHeart(buffer, n)
		DealWithParmGet(buffer, n)
		DealWithParmSetReponse(buffer, n)
		DealWithPreUpdateFirm(buffer, n)
		DealWithUpdateFirm(buffer, n)

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
