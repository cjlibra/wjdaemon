package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	"io"
	"os"

	"runtime"

	"mime/multipart"
	"strconv"

	"net"
	"net/http"

	"strings"
	"time"

	"github.com/golang/glog"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func dbopen(mongohost string) (*mgo.Session, error) {
	var session *mgo.Session
	var err error
	for {
		session, err = mgo.Dial(mongohost)
		if err != nil {
			glog.V(1).Infoln("数据库连接断了，连接的数据库ip：", mongohost)

			time.Sleep(time.Second * 5)
			glog.V(1).Infoln("重新启动连接")
		} else {
			break
		}

	}
	return session, err
}
func dbInsertheart(StrucPack PackageStruct) {

	for {
		err := c.Insert(&StrucPack)
		if err != nil {

			glog.V(1).Infoln("插入数据库有问题，有可能是数据库连接断了")

			session.Close()
			time.Sleep(time.Second * 5)

			session, _ = dbopen(*mongohost)
			c = session.DB("heart").C("info")

		} else {
			glog.V(2).Infoln("成功插入数据库")
			break
		}
	}
}

func SocketServer(sockport string) {

	netListen, err := net.Listen("tcp", ":"+sockport)
	if err != nil {
		glog.V(1).Infoln("Listen出错，可能端口占用：", sockport)
		return
	}

	defer netListen.Close()

	glog.Infoln("后台服务启动于端口 " + sockport)
	for {
		conn, err := netListen.Accept()
		if err != nil {
			continue
		}

		glog.V(2).Infoln(conn.RemoteAddr().String(), "->", "TCP连接成功")
		go handleConnection(conn)
	}

}
func foundserialinpoolbynum(serialnum []byte) int {
	for index, value := range linesinfos {
		if bytes.Equal(value.FirmSerialno[:6+12], serialnum[:6+12]) == true {
			return index
		}
	}
	return -1
}

type UPDATETASK struct {
	FirmSerial     [6 + 12]byte
	Procedure      int
	FirmFileCount  int
	FirmFileBuf    []byte
	AllFramesCount int
	PartPercent    int
	WholeChecksum  byte
	DoTime         time.Time
	ReportChan     chan int
	NumNowPart     int
	flagstop       int
}

var updatefirmtasks []UPDATETASK

func searchtask(FirmSerial []byte) int {
	for index, value := range updatefirmtasks {
		if bytes.Equal(value.FirmSerial[:6+12], FirmSerial[:6+12]) == true {
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
		if err == io.EOF {
			break
		} else {
			if err != nil {
				glog.V(1).Infoln("读取文件出错", line)
				return -2, err
			}
		}
		if len(line) <= 8 {
			glog.V(1).Infoln("读取文件出错,行太短", line)
			return -1, err
		}

		aa1, err := strconv.ParseInt(string(line[1:3]), 16, 0)
		aa := int(aa1)
		if err != nil {
			return -4, err
		}
		if bytes.Equal([]byte(line[7:7+2]), bb[:2]) != true {
			continue
		}
		b, err := hex.DecodeString(line[9 : 9+aa*2])
		if err != nil {
			glog.V(1).Infoln("无法hex.DecodeString")
			return -3, err
		}
		copy(firmbuf[firmbufcount:firmbufcount+aa], b)
		firmbufcount = firmbufcount + aa

	}
	return firmbufcount, nil
}
func updatefirming(firmbuf []byte, firmnum int, binFirmSerial []byte) {

	var oneupdatefirmtask UPDATETASK
	oneupdatefirmtask.FirmFileBuf = firmbuf

	copy(oneupdatefirmtask.FirmSerial[:6+12], binFirmSerial[:6+12])
	oneupdatefirmtask.Procedure = 1

	oneupdatefirmtask.FirmFileCount = firmnum

	oneupdatefirmtask.WholeChecksum = CalcChecksum(firmbuf, firmnum+1)
	oneupdatefirmtask.PartPercent = 0
	oneupdatefirmtask.DoTime = time.Now().Local()
	oneupdatefirmtask.AllFramesCount = oneupdatefirmtask.FirmFileCount / CountInPerFrame
	if oneupdatefirmtask.FirmFileCount%CountInPerFrame > 0 {
		oneupdatefirmtask.AllFramesCount = oneupdatefirmtask.AllFramesCount + 1
	}
	no := searchtask(binFirmSerial)
	if no == -1 {
		updatefirmtasks = append(updatefirmtasks, oneupdatefirmtask)
		no = len(updatefirmtasks) - 1
		updatefirmtasks[no].ReportChan = make(chan int)
	} else {
		if updatefirmtasks[no].Procedure != 0 {
			glog.V(1).Infoln("客户端正在升级中，不要重复升级")
			//w.Write([]byte("{status:'1004'}"))
			return
		}
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
		glog.V(1).Infoln("客户端未连接上来")
		//w.Write([]byte("{status:'1005'}"))
		return
	}
	framenum := firmnum / CountInPerFrame
	if firmnum%CountInPerFrame > 0 {
		framenum = framenum + 1
	}
	btmp := make([]byte, 2)
	binary.LittleEndian.PutUint16(btmp, uint16(framenum))

	buffer_updateparm := make([]byte, 256)
	buffer_updateparm[0] = 0xEE
	buffer_updateparm[1] = 0x83
	copy(buffer_updateparm[2:2+6+12], binFirmSerial[:6+12])
	buffer_updateparm[8+12] = 0
	buffer_updateparm[9+12] = 7
	buffer_updateparm[10+12] = btmp[1]
	buffer_updateparm[11+12] = btmp[0]
	buffer_updateparm[12+12] = CalcChecksum(firmbuf, firmnum+1)
	buffer_updateparm[13+12] = 0
	buffer_updateparm[14+12] = 0
	buffer_updateparm[15+12] = 0
	buffer_updateparm[16+12] = 0
	buffer_updateparm[17+12] = 0 //close
	buffer_updateparm[18+12] = CalcChecksum(buffer_updateparm, 19+12)

	sendcount, err := linesinfos[poolgetnum].Conn.Write(buffer_updateparm[:19+12])
	if err != nil {
		glog.V(3).Infoln("无法发送0x83数据包", hex.EncodeToString(buffer_updateparm[:19+12]), sendcount)
		//	w.Write([]byte("{status:'1005'}"))
		updatefirmtasks[no].Procedure = 0
		updatefirmtasks[no].FirmFileCount = 0
		updatefirmtasks[no].PartPercent = 0

		updatefirmtasks[no].WholeChecksum = 0
		updatefirmtasks[no].FirmFileBuf = []byte{}
		updatefirmtasks[no].AllFramesCount = 0
		return
	}

	go updatefirmstart(poolgetnum, no)
	glog.V(4).Infoln("成功发送0x83数据包", hex.EncodeToString(buffer_updateparm[:19+12]), sendcount)
	//w.Write([]byte("{status:'0'}"))
	return

}
func updatefirm(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	w.Header().Add("Access-Control-Allow-Origin", "*") //保证跨域的ajax
	//r.ParseForm()
	FirmSerial := r.FormValue("FirmSerial")
	if len(r.Form["FirmSerial"]) <= 0 {
		glog.V(1).Infoln("FirmSerial请求参数缺失")
		w.Write([]byte("{status:'1001'}"))
		return
	}
	if len(FirmSerial) != 6+12 {
		glog.V(1).Infoln("FirmSerial请求参数内容缺失")
		w.Write([]byte("{status:'1002'}"))
		return
	}
	binFirmSerial := []byte(FirmSerial)

	if "POST" != r.Method {
		glog.V(1).Infoln("请求模式：", r.Method)
		w.Write([]byte("{status:'1004'}"))
		return
	}
	file, _, err := r.FormFile("firmfile")
	if err != nil {
		glog.V(1).Infoln("文件上传失败")
		w.Write([]byte("{status:'1006'}"))
		return
	}
	defer file.Close()
	firmbuf := make([]byte, 1024*1024)

	firmnum, err := ReadFromStdFile(file, firmbuf)

	if firmnum <= 0 {
		glog.V(1).Infoln("Firm文件读取失败", firmnum)
		w.Write([]byte("{status:'1005'}"))
		return
	}
	//glog.V(6).Infoln("文件内容", firmnum, hex.EncodeToString(firmbuf[:firmnum]))
	//ioutil.WriteFile("sss.bin", firmbuf[:firmnum], 0744)

	var oneupdatefirmtask UPDATETASK
	oneupdatefirmtask.FirmFileBuf = firmbuf

	copy(oneupdatefirmtask.FirmSerial[:6+12], binFirmSerial[:6+12])
	oneupdatefirmtask.Procedure = 1

	oneupdatefirmtask.FirmFileCount = firmnum

	oneupdatefirmtask.WholeChecksum = CalcChecksum(firmbuf, firmnum+1)
	oneupdatefirmtask.PartPercent = 0
	oneupdatefirmtask.DoTime = time.Now().Local()
	oneupdatefirmtask.AllFramesCount = oneupdatefirmtask.FirmFileCount / CountInPerFrame
	if oneupdatefirmtask.FirmFileCount%CountInPerFrame > 0 {
		oneupdatefirmtask.AllFramesCount = oneupdatefirmtask.AllFramesCount + 1
	}
	no := searchtask(binFirmSerial)
	if no == -1 {
		updatefirmtasks = append(updatefirmtasks, oneupdatefirmtask)
		no = len(updatefirmtasks) - 1
		updatefirmtasks[no].ReportChan = make(chan int)
	} else {
		if updatefirmtasks[no].Procedure != 0 {
			glog.V(1).Infoln("客户端正在升级中，不要重复升级")
			w.Write([]byte("{status:'1004'}"))
			return
		}
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
		glog.V(1).Infoln("客户端未连接上来")
		w.Write([]byte("{status:'1005'}"))
		return
	}
	framenum := firmnum / CountInPerFrame
	if firmnum%CountInPerFrame > 0 {
		framenum = framenum + 1
	}
	btmp := make([]byte, 2)
	binary.LittleEndian.PutUint16(btmp, uint16(framenum))

	buffer_updateparm := make([]byte, 256)
	buffer_updateparm[0] = 0xEE
	buffer_updateparm[1] = 0x83
	copy(buffer_updateparm[2:2+6+12], binFirmSerial[:6+12])
	buffer_updateparm[8+12] = 0
	buffer_updateparm[9+12] = 7
	buffer_updateparm[10+12] = btmp[1]
	buffer_updateparm[11+12] = btmp[0]
	buffer_updateparm[12+12] = CalcChecksum(firmbuf, firmnum+1)
	buffer_updateparm[13+12] = 0
	buffer_updateparm[14+12] = 0
	buffer_updateparm[15+12] = 0
	buffer_updateparm[16+12] = 0
	buffer_updateparm[17+12] = 0 //close
	buffer_updateparm[18+12] = CalcChecksum(buffer_updateparm, 19+12)

	sendcount, err := linesinfos[poolgetnum].Conn.Write(buffer_updateparm[:19+12])
	if err != nil {
		glog.V(3).Infoln("无法发送0x83数据包", hex.EncodeToString(buffer_updateparm[:19+12]), sendcount)
		w.Write([]byte("{status:'1005'}"))
		updatefirmtasks[no].Procedure = 0
		updatefirmtasks[no].FirmFileCount = 0
		updatefirmtasks[no].PartPercent = 0

		updatefirmtasks[no].WholeChecksum = 0
		updatefirmtasks[no].FirmFileBuf = []byte{}
		updatefirmtasks[no].AllFramesCount = 0
		return
	}

	go updatefirmstart(poolgetnum, no)
	glog.V(4).Infoln("成功发送0x83数据包", hex.EncodeToString(buffer_updateparm[:19+12]), sendcount)
	w.Write([]byte("{status:'0'}"))
	return
}
func updatefirmstart(poolgetnum int, no int) {
	defer func() {
		glog.V(2).Infoln("升级进程退出")
		updatefirmtasks[no].flagstop = 0
		updatefirmtasks[no].Procedure = 0
		updatefirmtasks[no].NumNowPart = 0
		updatefirmtasks[no].AllFramesCount = 0
		updatefirmtasks[no].PartPercent = 0
		updatefirmtasks[no].DoTime = time.Now().Local()
		updatefirmtasks[no].WholeChecksum = 0
		updatefirmtasks[no].FirmFileCount = 0
	}()

	if updatefirmtasks[no].flagstop == 1 {
		glog.V(2).Infoln("升级进程开头退出")
		return
	}

	var rp int
	select {
	case rp = <-updatefirmtasks[no].ReportChan:
		if rp < 0 {
			glog.V(1).Infoln("出错0x03数据包，rp:", rp)
			return
		}

	case <-time.After(time.Second * 5):
		glog.V(1).Infoln("超时0x03数据包")
		return

	}

	buffer_update := make([]byte, 1024)
	FrameCount := updatefirmtasks[no].AllFramesCount

	addfilesize := 0
	var SizeinPerPack uint16 = 0
	var SizeinWholePack uint16 = 0
	btmp := make([]byte, 2)
	i := 0
	for i = 0; i < FrameCount; i++ {
		if updatefirmtasks[no].flagstop == 1 {
			glog.V(2).Infoln("升级进程循环中退出")
			return
		}
		addfilesize = (i + 1) * CountInPerFrame
		if addfilesize >= updatefirmtasks[no].FirmFileCount {
			SizeinPerPack = uint16(updatefirmtasks[no].FirmFileCount - (addfilesize - CountInPerFrame))
		} else {
			SizeinPerPack = uint16(CountInPerFrame)
		}

		SizeinWholePack = SizeinPerPack + 5

		buffer_update[0] = 0xEE
		buffer_update[1] = 0x84
		copy(buffer_update[2:2+6+12], updatefirmtasks[no].FirmSerial[:6+12])
		binary.LittleEndian.PutUint16(btmp, SizeinWholePack)
		buffer_update[8+12] = btmp[1]
		buffer_update[9+12] = btmp[0]
		binary.LittleEndian.PutUint16(btmp, uint16(i))
		buffer_update[10+12] = btmp[1]
		buffer_update[11+12] = btmp[0]
		binary.LittleEndian.PutUint16(btmp, uint16(SizeinPerPack))
		buffer_update[12+12] = btmp[1]
		buffer_update[13+12] = btmp[0]
		copy(buffer_update[14+12:14+SizeinPerPack+12], updatefirmtasks[no].FirmFileBuf[(i)*CountInPerFrame:(i)*CountInPerFrame+int(SizeinPerPack)])
		buffer_update[14+12+SizeinPerPack] = CalcChecksum(buffer_update[14+12:14+12+SizeinPerPack], int(SizeinPerPack)+1)
		buffer_update[14+12+SizeinPerPack+1] = 0 //close bit
		buffer_update[14+12+SizeinPerPack+2] = CalcChecksum(buffer_update[0:], 14+12+int(SizeinPerPack)+2+1)
		sendcount, err := linesinfos[poolgetnum].Conn.Write(buffer_update[:14+12+SizeinPerPack+2+1])
		if err != nil {
			glog.V(3).Infoln("无法发送0x84数据包", hex.EncodeToString(buffer_update[:14+12+SizeinPerPack+2+1]), sendcount)
			time.Sleep(time.Second * 20)

			i = i - 1
			continue
		}
		glog.V(4).Infoln("成功发送0x84数据包", hex.EncodeToString(buffer_update[:14+12+SizeinPerPack+2+1]), sendcount)
		updatefirmtasks[no].Procedure = 3

		glog.V(5).Infoln("i:", i)

		rp = -1
		select {
		case rp = <-updatefirmtasks[no].ReportChan:
		case <-time.After(10 * time.Second):
			glog.V(1).Infoln("等待接收升级反馈数据包超时10秒钟", string(updatefirmtasks[no].FirmSerial[:18]))
			rp = i
		}
		glog.V(5).Infoln("rp is", rp, "i:", i)
		if rp < 0 {
			return
		}

		i = rp - 1

	}
	if i == FrameCount {
		glog.V(2).Infoln("升级成功,设备码：", string(updatefirmtasks[no].FirmSerial[:18]))
		updatefirmtasks[no].Procedure = 0
	} else {
		glog.V(2).Infoln("升级失败,设备码：", string(updatefirmtasks[no].FirmSerial[:18]))
		updatefirmtasks[no].Procedure = 0
	}

}
func stopupdateprocedure(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	w.Header().Add("Access-Control-Allow-Origin", "*") //保证跨域的ajax
	FirmSerial := r.FormValue("FirmSerial")
	if len(r.Form["FirmSerial"]) <= 0 {
		glog.V(1).Infoln("FirmSerial请求参数缺失")
		w.Write([]byte("{status:'1001'}"))
		return
	}
	if len(FirmSerial) != 18 {
		glog.V(1).Infoln("FirmSerial请求参数内容缺失")
		w.Write([]byte("{status:'1002'}"))
		return
	}
	binFirmSerial := []byte(FirmSerial)
	for idindex, value := range updatefirmtasks {
		if bytes.Equal(value.FirmSerial[:18], binFirmSerial[:18]) == true {
			updatefirmtasks[idindex].flagstop = 1
			updatefirmtasks[idindex].Procedure = 0

			//updatefirmtasks[idindex].ReportChan = -3
			//updatefirmtasks[idindex].FirmFileBuf = []byte{}

			break
		}
	}
	w.Write([]byte("{status:'0'}"))
	return
}
func getparmfromfrontafter(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	w.Header().Add("Access-Control-Allow-Origin", "*") //保证跨域的ajax
	FirmSerial := r.FormValue("FirmSerial")
	if len(r.Form["FirmSerial"]) <= 0 {
		glog.V(1).Infoln("FirmSerial请求参数缺失")
		w.Write([]byte("{status:'1001'}"))
		return
	}
	if len(FirmSerial) <= 0 {
		glog.V(1).Infoln("FirmSerial请求参数内容缺失")
		w.Write([]byte("{status:'1002'}"))
		return
	}
	binFirmSerial := []byte(FirmSerial)

	if oneStrucPack.Refeshflag != 1 {
		glog.V(1).Infoln("信息没有更新")
		w.Write([]byte("{status:'1003'}"))
		return
	}

	if bytes.Equal([]byte(oneStrucPack.FirmSerailno[:6+12]), binFirmSerial[:6+12]) != true {
		glog.V(1).Infoln("找不到Firmserialno")
		w.Write([]byte("{status:'1004'}"))
		return
	}
	b, err := json.Marshal(oneStrucPack)
	if err != nil {
		glog.V(1).Infoln("json编码问题", err)
		w.Write([]byte("{status:'1005'}"))
		return
	}

	glog.V(5).Infoln(string(b))
	w.Write(b)
	oneStrucPack.Refeshflag = 0

}
func getparmfromfront(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	w.Header().Add("Access-Control-Allow-Origin", "*") //保证跨域的ajax

	if len(r.Form["FirmSerial"]) <= 0 {
		glog.V(1).Infoln("FirmSerial请求参数缺失")
		w.Write([]byte("{status:'1001'}"))
		return
	}
	FirmSerial := r.FormValue("FirmSerial")
	if len(FirmSerial) != 6+12 {
		glog.V(1).Infoln("FirmSerial请求参数内容缺失")
		w.Write([]byte("{status:'1002'}"))
		return
	}
	binFirmSerial := []byte(FirmSerial)

	glog.V(5).Infoln(hex.EncodeToString(binFirmSerial))
	poolgetnum := foundserialinpoolbynum(binFirmSerial)
	if poolgetnum == -1 {
		glog.V(1).Infoln("客户端未连接上来")
		w.Write([]byte("{status:'1004'}"))
		return
	}
	buffer_getparm := make([]byte, 1024)
	buffer_getparm[0] = 0xEE
	buffer_getparm[1] = 0x80
	copy(buffer_getparm[2:2+6+12], binFirmSerial[:6+12])
	buffer_getparm[8+12] = 0
	buffer_getparm[9+12] = 0
	buffer_getparm[10+12] = CalcChecksum(buffer_getparm, 11+12)

	sendcount, err := linesinfos[poolgetnum].Conn.Write(buffer_getparm[:11+12])
	if err != nil {
		glog.V(3).Infoln("无法发送0x80数据包", hex.EncodeToString(buffer_getparm[:11+12]), sendcount)
		w.Write([]byte("{status:'1005'}"))
		return
	}

	glog.V(4).Infoln("成功发送0x80数据包", hex.EncodeToString(buffer_getparm[:11+12]))
	w.Write([]byte("{status:'0'}"))
	return
}
func updatefirmafter(w http.ResponseWriter, r *http.Request) {
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
	var stroutupdates []OUTUPDATE
	w.Header().Add("Access-Control-Allow-Origin", "*") //保证跨域的ajax
	for _, value := range updatefirmtasks {
		copy(stroutupdate.FirmSerial[:18], value.FirmSerial[:6+12])
		stroutupdate.Procedure = value.Procedure
		stroutupdate.FirmFileCount = value.FirmFileCount
		stroutupdate.AllFramesCount = value.AllFramesCount
		stroutupdate.PartPercent = value.PartPercent
		stroutupdate.WholeChecksum = value.WholeChecksum
		stroutupdate.DoTime = value.DoTime.Local()

		stroutupdates = append(stroutupdates, stroutupdate)
	}

	b, err := json.Marshal(stroutupdates)
	if err != nil {
		glog.V(1).Infoln("json编码问题alllinestrs", err)
		w.Write([]byte("{status:'1001'}"))
		return
	}

	glog.V(5).Infoln(string(b))
	w.Write(b)
}
func setparmtofrontafter(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	w.Header().Add("Access-Control-Allow-Origin", "*") //保证跨域的ajax
	FirmSerial := r.FormValue("FirmSerial")
	if len(r.Form["FirmSerial"]) <= 0 {
		glog.V(1).Infoln("FirmSerial请求参数缺失")
		w.Write([]byte("{status:'1001'}"))
		return
	}
	if len(FirmSerial) != 6+12 {
		glog.V(1).Infoln("FirmSerial请求参数内容缺失")
		w.Write([]byte("{status:'1002'}"))
		return
	}
	binFirmSerial := []byte(FirmSerial)

	if bytes.Equal([]byte(secondStrucPack.FirmSerailno[:6+12]), binFirmSerial[:6+12]) != true {
		glog.V(1).Infoln("reponse pool找不到Firmserialno")
		w.Write([]byte("{status:'1004'}"))
		return
	}
	b, err := json.Marshal(secondStrucPack)
	if err != nil {
		glog.V(1).Infoln("json编码问题", err)
		w.Write([]byte("{status:'1005'}"))
		return
	}

	glog.V(5).Infoln(string(b))
	w.Write(b)

}

func setparmtofront(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	w.Header().Add("Access-Control-Allow-Origin", "*") //保证跨域的ajax
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

		glog.V(1).Infoln("setparmtofront请求参数缺失")
		w.Write([]byte("{status:'1001'}"))
		return
	}
	if len(FirmSerial) != 6+12 ||
		len(Maskset) != 2 ||
		len(Outantenaset) != 2 ||
		len(Inantenaset) != 2 ||
		len(Monswitchset) != 2 ||
		len(Sysresetset) != 2 ||
		len(Defaultbackset) != 2 ||
		len(Otherset) != 6 {

		glog.V(1).Infoln("setparmtofront请求参数内容不准确")
		w.Write([]byte("{status:'1002'}"))
		return
	}

	binFirmSerial := []byte(FirmSerial)

	poolgetnum := foundserialinpoolbynum(binFirmSerial)
	if poolgetnum == -1 {
		glog.V(1).Infoln("客户端未连接上来")
		w.Write([]byte("{status:'1004'}"))
		return
	}

	binMaskset, err := hex.DecodeString(Maskset)
	if err != nil {
		glog.V(1).Infoln("Maskset DecodeString出错")
		w.Write([]byte("{status:'1003'}"))
		return
	}
	binOutantenaset, err := hex.DecodeString(Outantenaset)
	if err != nil {
		glog.V(1).Infoln("Outantenaset DecodeString出错")
		w.Write([]byte("{status:'1003'}"))
		return
	}
	binInantenaset, err := hex.DecodeString(Inantenaset)
	if err != nil {
		glog.V(1).Infoln("Inantenaset DecodeString出错")
		w.Write([]byte("{status:'1003'}"))
		return
	}
	binMonswitchset, err := hex.DecodeString(Monswitchset)
	if err != nil {
		glog.V(1).Infoln("Monswitchset DecodeString出错")
		w.Write([]byte("{status:'1003'}"))
		return
	}
	binSysresetset, err := hex.DecodeString(Sysresetset)
	if err != nil {
		glog.V(1).Infoln("Sysresetset DecodeString出错")
		w.Write([]byte("{status:'1003'}"))
		return
	}
	binDefaultbackset, err := hex.DecodeString(Defaultbackset)
	if err != nil {
		glog.V(1).Infoln("Defaultbackset DecodeString出错")
		w.Write([]byte("{status:'1003'}"))
		return
	}
	binOtherset, err := hex.DecodeString(Otherset)
	if err != nil {
		glog.V(1).Infoln("Otherset DecodeString出错")
		w.Write([]byte("{status:'1003'}"))
		return
	}
	buffer_setparm := make([]byte, 1024)
	buffer_setparm[0] = 0xEE
	buffer_setparm[1] = 0x82
	copy(buffer_setparm[2:2+6+12], binFirmSerial[:6+12])
	buffer_setparm[8+12] = 0
	buffer_setparm[9+12] = 9
	buffer_setparm[10+12] = binMaskset[0]
	buffer_setparm[11+12] = binOutantenaset[0]
	buffer_setparm[12+12] = binInantenaset[0]
	buffer_setparm[13+12] = binMonswitchset[0]
	buffer_setparm[14+12] = binSysresetset[0]
	buffer_setparm[15+12] = binDefaultbackset[0]
	copy(buffer_setparm[16+12:16+3+12], binOtherset[:3])
	buffer_setparm[19+12] = 0
	buffer_setparm[20+12] = CalcChecksum(buffer_setparm, 21+12)

	sendcount, err := linesinfos[poolgetnum].Conn.Write(buffer_setparm[:21+12])
	if err != nil {
		glog.V(3).Infoln("无法发送0x82数据包", hex.EncodeToString(buffer_setparm[:21+12]), sendcount)
		w.Write([]byte("{status:'1005'}"))
		return
	}

	glog.V(4).Infoln("成功发送0x82数据包", hex.EncodeToString(buffer_setparm[:21+12]))
	w.Write([]byte("{status:'0'}"))
	return

}

type SDBACK struct {
	PageAll     int
	CurrentPage int
	Status      int
	Data        []CONNINFO
}

func GetSearchDevices(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	FirmSerial := r.FormValue("DeviceNO")
	Page := r.FormValue("Page")
	Callfunc := r.FormValue("Callback")
	if len(r.Form["DeviceNO"]) <= 0 || len(r.Form["Page"]) <= 0 || len(r.Form["Callback"]) <= 0 {
		glog.V(1).Infoln("GetSearchDevices请求参数缺失")
		w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1001}")))
		return
	}
	if len(FirmSerial) <= 0 || len(Page) <= 0 || len(Callfunc) <= 0 {
		glog.V(1).Infoln("GetSearchDevices请求参数内容不准确")
		w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1002}")))
		return
	}
	var sdbackret SDBACK
	var devicestatus CONNINFO
	var devicestatuses []CONNINFO
	var devicestatusespage []CONNINFO
	if len(linesinfos) <= 0 {
		glog.V(1).Infoln("linesinfos为空，表示没有设备连接上来")
		w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1003}")))
		return
	}
	flagnoget := 0
	for _, lineinfo := range linesinfos {
		if strings.Contains(string(lineinfo.FirmSerialno[:18]), FirmSerial) == true {
			devicestatus.ClientIp = lineinfo.ClientIp
			devicestatus.Clientport = lineinfo.Clientport
			devicestatus.Dotime = lineinfo.Dotime
			devicestatus.FirmSerialno = lineinfo.FirmSerialno
			devicestatus.Alive = lineinfo.Alive
			devicestatus.HeartInfo = lineinfo.HeartInfo
			devicestatuses = append(devicestatuses, devicestatus)
			flagnoget = 1
		}

	}
	if flagnoget == 0 {
		glog.V(1).Infoln("没有找到该设备，表示该设备没有连接上来")
		w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1003}")))
		return
	}
	pagesfromds := 0
	countsfromds := len(devicestatuses)
	if countsfromds%200 == 0 {
		pagesfromds = countsfromds / 200
	} else {
		pagesfromds = countsfromds/200 + 1
	}
	sdbackret.PageAll = pagesfromds
	tmpa, err := strconv.Atoi(Page)
	if err != nil {
		glog.V(1).Infoln("Page非数字")
		w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1006}")))
		return
	}
	if tmpa <= 0 {
		glog.V(1).Infoln("Page不能小于等于0")
		w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1000}")))
		return
	}

	if tmpa > pagesfromds {
		sdbackret.CurrentPage = pagesfromds
	} else {
		sdbackret.CurrentPage = tmpa
	}
	sdbackret.Status = 0

	countsonnextpage := 0
	if sdbackret.CurrentPage*200 > countsfromds {
		countsonnextpage = countsfromds
	} else {
		countsonnextpage = sdbackret.CurrentPage * 200
	}
	for i := (sdbackret.CurrentPage - 1) * 200; i < countsonnextpage; i++ {
		devicestatusespage = append(devicestatusespage, devicestatuses[i])
	}
	sdbackret.Data = devicestatusespage

	b, err := json.Marshal(sdbackret)
	if err != nil {
		glog.V(1).Infoln("json编码问题sdbackret", err)
		w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1005}")))
		return
	}

	retstr := fmt.Sprintf("%s(%s);", Callfunc, string(b))
	glog.V(5).Infoln(retstr)
	w.Write([]byte(retstr))
}

func GetSearchDevicesbyheart(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	FirmSerial := r.FormValue("DeviceNO")
	Page := r.FormValue("Page")
	Callfunc := r.FormValue("Callback")

	SConnStatus := r.FormValue("ConnStatus")
	SEquipID := r.FormValue("EquipID")
	SDotime := r.FormValue("Dotime")
	SFirmVersion := r.FormValue("FirmVersion")
	SSoftVersion := r.FormValue("SoftVersion")
	SInsideAntena0 := r.FormValue("InsideAntena0")
	SInsideAntena1 := r.FormValue("InsideAntena1")
	SOutsideAntena0 := r.FormValue("OutsideAntena0")
	SOutsideAntena1 := r.FormValue("OutsideAntena1")
	SPhoneNum := r.FormValue("PhoneNum")
	SReadWriterStatus := r.FormValue("ReadWriterStatus")
	SSysEnergy := r.FormValue("SysEnergy")
	SServerIpPort := r.FormValue("ServerIpPort")

	if len(r.Form["DeviceNO"]) <= 0 || len(r.Form["Page"]) <= 0 || len(r.Form["Callback"]) <= 0 {
		glog.V(1).Infoln("GetSearchDevices请求参数缺失")
		w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1001}")))
		return
	}
	if len(FirmSerial) <= 0 || len(Page) <= 0 || len(Callfunc) <= 0 {
		glog.V(1).Infoln("GetSearchDevices请求参数内容不准确")
		w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1002}")))
		return
	}
	var sdbackret SDBACK
	var devicestatus CONNINFO
	var devicestatuses []CONNINFO
	var devicestatusespage []CONNINFO
	if len(linesinfos) <= 0 {
		glog.V(1).Infoln("linesinfos为空，表示没有设备连接上来")
		w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1003}")))
		return
	}
	flagnoget := 0
	a1 := true
	a2 := true
	a3 := true
	a4 := true
	a5 := true
	a6 := true
	a7 := true
	a8 := true
	a9 := true
	a10 := true
	a11 := true
	for _, lineinfo := range linesinfos {
		if strings.Contains(string(lineinfo.FirmSerialno[:18]), FirmSerial) == true {

			if len(SConnStatus) == 0 {
				a1 = true
			} else {
				byteb, err := hex.DecodeString(SConnStatus)
				if err != nil {
					glog.V(1).Infoln("ConnStatus输入格式有误")
					w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1004}")))
					return
				}
				a1 = (lineinfo.HeartInfo.ConnStatus == byteb[0])

			}
			if len(SEquipID) == 0 {
				a2 = true
			} else {
				a2 = lineinfo.HeartInfo.EquipID == SEquipID
			}

			if len(SDotime) == 0 {
				a3 = true
			} else {
				the_time, err := time.Parse("2006-01-02 15:04:05", SDotime)
				if err != nil {
					glog.V(1).Infoln("Dotime输入格式有误")
					w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1005}")))
					return
				}
				a3 = lineinfo.HeartInfo.Dotime.Before(the_time)
			}
			if len(SFirmVersion) == 0 {
				a4 = true
			} else {
				byteb, err := hex.DecodeString(SFirmVersion)
				if err != nil {
					glog.V(1).Infoln("FirmVersion输入格式有误")
					w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1006}")))
					return
				}
				a4 = (lineinfo.HeartInfo.FirmVersion == string(byteb))
			}

			if len(SSoftVersion) == 0 {
				a5 = true
			} else {
				byteb, err := hex.DecodeString(SSoftVersion)
				if err != nil {
					glog.V(1).Infoln("SoftVersion输入格式有误")
					w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1007}")))
					return
				}
				a5 = lineinfo.HeartInfo.SoftVersion == string(byteb)
			}

			if len(SInsideAntena0) == 0 || len(SInsideAntena1) == 0 {
				a6 = true
			} else {
				inta, err := strconv.Atoi(SInsideAntena0)
				if err != nil {
					glog.V(1).Infoln("InsideAntena0输入格式有误")
					w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1008}")))
					return
				}
				intb, err := strconv.Atoi(SInsideAntena1)
				if err != nil {
					glog.V(1).Infoln("InsideAntena1输入格式有误")
					w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1009}")))
					return
				}
				a6 = int(lineinfo.HeartInfo.InsideAntena) >= inta && int(lineinfo.HeartInfo.InsideAntena) <= intb
			}

			if len(SOutsideAntena0) == 0 || len(SOutsideAntena1) == 0 {
				a7 = true
			} else {
				inta, err := strconv.Atoi(SOutsideAntena0)
				if err != nil {
					glog.V(1).Infoln("OutsideAntena0输入格式有误")
					w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1010}")))
					return
				}
				intb, err := strconv.Atoi(SOutsideAntena1)
				if err != nil {
					glog.V(1).Infoln("OutsideAntena1输入格式有误")
					w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1011}")))
					return
				}
				a7 = int(lineinfo.HeartInfo.OutsideAntena) >= inta && int(lineinfo.HeartInfo.OutsideAntena) <= intb
			}

			if len(SPhoneNum) == 0 {
				a8 = true
			} else {
				a8 = lineinfo.HeartInfo.PhoneNum == SPhoneNum
			}

			if len(SReadWriterStatus) == 0 {
				a9 = true
			} else {
				byteb, err := hex.DecodeString(SReadWriterStatus)
				if err != nil {
					glog.V(1).Infoln("ReadWriterStatus输入格式有误")
					w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1012}")))
					return
				}
				a9 = lineinfo.HeartInfo.ReadWriterStatus == byteb[0]
			}

			if len(SSysEnergy) == 0 {
				a10 = true
			} else {
				byteb, err := hex.DecodeString(SSysEnergy)
				if err != nil {
					glog.V(1).Infoln("SysEnergy输入格式有误")
					w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1013}")))
					return
				}
				a10 = lineinfo.HeartInfo.SysEnergy == byteb[0]
			}
			if len(SServerIpPort) == 0 {
				a11 = true
			} else {
				byteb, err := hex.DecodeString(SServerIpPort)
				if err != nil {
					glog.V(1).Infoln("ServerIpPort输入格式有误")
					w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1014}")))
					return
				}
				a11 = lineinfo.HeartInfo.ServerIpPort == string(byteb)
			}

			if a1 && a2 && a3 && a4 && a5 && a6 && a7 && a8 && a9 && a10 && a11 != true {
				continue
			}

			devicestatus.ClientIp = lineinfo.ClientIp
			devicestatus.Clientport = lineinfo.Clientport
			devicestatus.Dotime = lineinfo.Dotime
			devicestatus.FirmSerialno = lineinfo.FirmSerialno
			devicestatus.Alive = lineinfo.Alive
			devicestatus.HeartInfo = lineinfo.HeartInfo
			devicestatuses = append(devicestatuses, devicestatus)
			flagnoget = 1
		}

	}
	if flagnoget == 0 {
		glog.V(1).Infoln("没有找到该设备，表示该设备没有连接上来")
		w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1023}")))
		return
	}
	pagesfromds := 0
	countsfromds := len(devicestatuses)
	if countsfromds%200 == 0 {
		pagesfromds = countsfromds / 200
	} else {
		pagesfromds = countsfromds/200 + 1
	}
	sdbackret.PageAll = pagesfromds
	tmpa, err := strconv.Atoi(Page)
	if err != nil {
		glog.V(1).Infoln("Page非数字")
		w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1026}")))
		return
	}
	if tmpa <= 0 {
		glog.V(1).Infoln("Page不能小于等于0")
		w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1020}")))
		return
	}

	if tmpa > pagesfromds {
		sdbackret.CurrentPage = pagesfromds
	} else {
		sdbackret.CurrentPage = tmpa
	}
	sdbackret.Status = 0

	countsonnextpage := 0
	if sdbackret.CurrentPage*200 > countsfromds {
		countsonnextpage = countsfromds
	} else {
		countsonnextpage = sdbackret.CurrentPage * 200
	}
	for i := (sdbackret.CurrentPage - 1) * 200; i < countsonnextpage; i++ {
		devicestatusespage = append(devicestatusespage, devicestatuses[i])
	}
	sdbackret.Data = devicestatusespage

	b, err := json.Marshal(sdbackret)
	if err != nil {
		glog.V(1).Infoln("json编码问题sdbackret", err)
		w.Write([]byte(fmt.Sprintf("%s(%s);", Callfunc, "{status:1025}")))
		return
	}

	retstr := fmt.Sprintf("%s(%s);", Callfunc, string(b))
	glog.V(5).Infoln(retstr)
	w.Write([]byte(retstr))
}

func SetCustomInfo(w http.ResponseWriter, r *http.Request) {
	type CUSTOMINFO struct {
		FirmSerial string
		CustomInfo string
		TheTime    time.Time
	}
	r.ParseForm()
	FirmSerial := r.FormValue("FirmSerial")
	CustomInfo := r.FormValue("CustomInfo")

	if len(r.Form["FirmSerial"]) <= 0 || len(r.Form["CustomInfo"]) <= 0 {
		glog.V(1).Infoln("SetCustomInfo请求参数缺失")
		w.Write([]byte("{status:1001}"))
		return
	}
	if len(FirmSerial) <= 0 {
		glog.V(1).Infoln("SetCustomInfo请求参数内容不准确")
		w.Write([]byte("{status:1002}"))
		return
	}
	cdb := session.DB("custom").C("info")
	var customstring CUSTOMINFO
	var customstrings []CUSTOMINFO
	cdb.Find(bson.M{"firmserial": FirmSerial}).All(&customstrings)
	if len(customstrings) == 0 {
		customstring.FirmSerial = FirmSerial
		customstring.CustomInfo = CustomInfo
		customstring.TheTime = time.Now().Local()

		cdb.Insert(&customstring)
	} else {
		cdb.Update(bson.M{"firmserial": FirmSerial}, bson.M{"$set": bson.M{"custominfo": CustomInfo, "thetime": time.Now().Local()}})
	}

	glog.V(2).Infoln("SetCustomInfo成功")
	w.Write([]byte("{status:0}"))
}

func UploadFiletoServer(w http.ResponseWriter, r *http.Request) {
	type FIRMFILEINFONOID struct {
		Version          string
		FileNameWithPath string
		Comments         string
		hashString       string
		CreateTime       time.Time
	}
	var Firmfileinfo FIRMFILEINFONOID
	cupload := session.DB("upload").C("info")
	r.ParseMultipartForm(32 << 20)
	//r.ParseForm()
	w.Header().Add("Access-Control-Allow-Origin", "*") //保证跨域的ajax

	Version := r.FormValue("Version")
	if len(r.Form["Version"]) <= 0 {
		glog.V(1).Infoln("Version请求参数缺失")
		w.Write([]byte("{status:'1001'}"))
		return
	}
	Comments := r.FormValue("Comments")
	if len(r.Form["Comments"]) <= 0 {
		glog.V(1).Infoln("Comments请求参数缺失")
		w.Write([]byte("{status:'1002'}"))
		return
	}
	if len(Version) < 6 {
		glog.V(1).Infoln("Version请求参数内容缺失")
		w.Write([]byte("{status:'1003'}"))
		return
	}

	if "POST" != r.Method {
		glog.V(1).Infoln("请求模式：", r.Method)
		w.Write([]byte("{status:'1004'}"))
		return
	}
	file, handle, err := r.FormFile("firmfile")
	if err != nil {

		glog.V(1).Infoln("上传文件出现问题")
		w.Write([]byte("{status:'1006'}"))
		return
	}

	FirmFileOnServerDir := "./upload/" + handle.Filename + time.Now().Local().String()
	FirmFileOnServerDir = strings.Replace(FirmFileOnServerDir, ":", "：", -1)
	f, err := os.OpenFile(FirmFileOnServerDir, os.O_WRONLY|os.O_CREATE, 0666)
	io.Copy(f, file)
	if err != nil {
		glog.V(1).Infoln("无法生成文件于服务器上:", FirmFileOnServerDir)
		w.Write([]byte("{status:'1007'}"))
		return
	}
	defer f.Close()
	defer file.Close()

	hs := sha1.New()
	io.Copy(hs, file)
	hashString := hs.Sum(nil)

	Firmfileinfo.Comments = Comments
	Firmfileinfo.Version = Version
	Firmfileinfo.FileNameWithPath = FirmFileOnServerDir
	Firmfileinfo.CreateTime = time.Now().Local()
	Firmfileinfo.hashString = hex.EncodeToString(hashString)
	cupload.Insert(&Firmfileinfo)

	glog.V(2).Infoln("上传文件成功：", FirmFileOnServerDir)
	w.Write([]byte("{status:0}"))

}

type FIRMFILEINFO struct {
	Id               bson.ObjectId `bson:"_id"`
	Version          string
	FileNameWithPath string
	Comments         string
	hashString       string
	CreateTime       time.Time
}

func GetUploadFileOnServerInfo(w http.ResponseWriter, r *http.Request) {

	var firmfileInfos []FIRMFILEINFO
	r.ParseForm()
	w.Header().Add("Access-Control-Allow-Origin", "*") //保证跨域的ajax
	cupload := session.DB("upload").C("info")
	cupload.Find(nil).All(&firmfileInfos)

	b, err := json.Marshal(firmfileInfos)
	if err != nil {
		glog.V(1).Infoln("json编码问题firmfileInfos", err)
		w.Write([]byte("{status:1001}"))
		return
	}

	glog.V(5).Infoln(string(b))
	w.Write(b)

}

func DelUploadFileOnServerInfo(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	w.Header().Add("Access-Control-Allow-Origin", "*") //保证跨域的ajax

	id := r.FormValue("id")
	if len(r.Form["id"]) <= 0 {
		glog.V(1).Infoln("id请求参数缺失")
		w.Write([]byte("{status:'1002'}"))
		return
	}
	if len(id) <= 6 {
		glog.V(1).Infoln("id请求参数内容缺失")
		w.Write([]byte("{status:'1003'}"))
		return
	}

	if bson.IsObjectIdHex(id) != true {
		glog.V(1).Infoln("id不是标准格式")
		w.Write([]byte("{status:'1005'}"))
		return
	}
	objid := bson.ObjectIdHex(id)
	cupload := session.DB("upload").C("info")
	_, err := cupload.RemoveAll(bson.M{"_id": objid})
	if err != nil {
		glog.V(1).Infoln("无法删除")
		w.Write([]byte("{status:'1004'}"))
		return
	}

	glog.V(2).Infoln("成功删除")
	w.Write([]byte("{status:'0'}"))

}
func UploadCmdString(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	w.Header().Add("Access-Control-Allow-Origin", "*") //保证跨域的ajax

	cmdstring := r.FormValue("cmdstring")
	if len(r.Form["cmdstring"]) <= 0 {
		glog.V(1).Infoln("cmdstring请求参数缺失")
		w.Write([]byte("{status:'1002'}"))
		return
	}
	if len(cmdstring) <= 6 {
		glog.V(1).Infoln("cmdstring请求参数内容缺失")
		w.Write([]byte("{status:'1003'}"))
		return
	}

	glog.V(2).Infoln(cmdstring)

	type CMDSTRSTRU struct {
		Id          string
		FirmSerials []string
	}
	var cmdstrstrus []CMDSTRSTRU
	err := json.Unmarshal([]byte(cmdstring), &cmdstrstrus)
	if err != nil {
		glog.V(1).Infoln("json无法Unmarshal上传的命令字符串")
		w.Write([]byte("{status:'1004'}"))
		return
	}
	var firmfilefromdb FIRMFILEINFO
	cupload := session.DB("upload").C("info")
	for _, value := range cmdstrstrus {
		err := cupload.Find(bson.M{"_id": value.Id}).One(&firmfilefromdb)
		if err != nil {
			glog.V(1).Infoln("upload数据库内找不到数据by:", value.Id)
			w.Write([]byte("{status:'1005'}"))
			return
		}

		fbuf, err := ioutil.ReadFile(firmfilefromdb.FileNameWithPath)
		if err != nil {
			glog.V(1).Infoln("无法读取文件：", firmfilefromdb.FileNameWithPath)
			//w.Write([]byte("{status:'1006'}"))
			continue
		}
		for _, firmserial := range value.FirmSerials {
			go updatefirming(fbuf, len(fbuf), []byte(firmserial))
			glog.V(2).Infoln("升级固件:", firmserial, firmfilefromdb.FileNameWithPath)
		}

	}

	glog.V(2).Infoln("批量升级命令接收成功")
	w.Write([]byte("{status:'0'}"))
}

var CountInPerFrame int
var session *mgo.Session
var c *mgo.Collection
var mongohost *string

func main() {
	defer func() { // 必须要先声明defer，否则不能捕获到panic异常

		err := recover()
		glog.Info("程序崩溃了，等待30秒后再次启动： ", err)
		time.Sleep(time.Second * 30)

		main()
	}()
	CountInPerFrame = 256

	NCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(NCPU)

	//defer session.Close()
	sockport := flag.Int("p", 48080, "socket server port")
	webport := flag.Int("wp", 58080, "socket server port")
	mongohost = flag.String("h", "127.0.0.1", "ip of mongodb server")
	flag.Parse()

	session, _ = dbopen(*mongohost)
	c = session.DB("heart").C("info")

	go SocketServer(fmt.Sprintf("%d", *sockport))

	http.HandleFunc("/updatefirmafter", updatefirmafter)
	http.HandleFunc("/updatefirm", updatefirm)
	http.HandleFunc("/stopupdateprocedure", stopupdateprocedure)
	http.HandleFunc("/getparmfromfrontafter", getparmfromfrontafter)
	http.HandleFunc("/getparmfromfront", getparmfromfront)

	http.HandleFunc("/setparmtofrontafter", setparmtofrontafter)
	http.HandleFunc("/setparmtofront", setparmtofront)

	http.HandleFunc("/GetSearchDevices", GetSearchDevices)
	http.HandleFunc("/GetSearchDevicesbyheart", GetSearchDevicesbyheart)

	http.HandleFunc("/SetCustomInfo", SetCustomInfo)
	http.HandleFunc("/UploadFiletoServer", UploadFiletoServer)
	http.HandleFunc("/GetUploadFileOnServerInfo", GetUploadFileOnServerInfo)
	http.HandleFunc("/DelUploadFileOnServerInfo", DelUploadFileOnServerInfo)

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
	FirmSerial := make([]byte, 6+12)
	copy(FirmSerial[:6+12], buffer[2:2+6+12])

	/*if buffer[15] != CalcChecksum(buffer[14:14+1], 1) {
		glog.V(3).Infoln("rp ischecksum")
		return 2
	}*/
	num := searchtask(FirmSerial)
	if num == -1 {
		return 3
	}
	xuhao := int(buffer[10+12])*256 + int(buffer[11+12])
	if buffer[14+12] != 0 {
		updatefirmtasks[num].ReportChan <- xuhao
		if xuhao == 0 {
			updatefirmtasks[num].ReportChan <- -2 //反馈不成功，又是0从头开始，直接出错处理
		}
	} else {
		updatefirmtasks[num].ReportChan <- xuhao
		yusu := uint(updatefirmtasks[num].FirmFileCount % CountInPerFrame)
		if yusu > 0 {
			updatefirmtasks[num].PartPercent = xuhao * 100 / (updatefirmtasks[num].FirmFileCount/CountInPerFrame + 1)
		} else {
			updatefirmtasks[num].PartPercent = xuhao * 100 / (updatefirmtasks[num].FirmFileCount/CountInPerFrame + 0)
		}
		updatefirmtasks[num].NumNowPart = xuhao
	}

	return 0
}

func DealWithPreUpdateFirm(buffer []byte, n int) int {
	CMDchar := buffer[1]
	if CMDchar != 0x03 {
		return 1
	}
	FirmSerial := make([]byte, 6+12)
	copy(FirmSerial[:6+12], buffer[2:2+6+12])
	allchecksum := buffer[12+12]

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
	Refeshflag int
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
	secondStrucPack.FirmSerailno = string(buffer[2 : 2+6+12])
	secondStrucPack.ParmSetResponse = hex.EncodeToString(buffer[10+12 : 10+9+12])

	return 0
}

var oneStrucPack PackageStruct

func DealWithParmGet(buffer []byte, n int) int {
	CMDchar := buffer[1]
	if CMDchar != 0x00 {
		return 1
	}

	oneStrucPack.CMDchar = buffer[1]
	oneStrucPack.FirmSerailno = string(buffer[2 : 2+6+12])
	oneStrucPack.EquipID = string(buffer[10+12 : 10+30+12])
	oneStrucPack.PhoneNum = string(buffer[40+12 : 40+11+12])
	oneStrucPack.SysEnergy = buffer[40+11+12]
	oneStrucPack.ConnStatus = buffer[40+11+1+12]
	oneStrucPack.ReadWriterStatus = buffer[53+12]
	oneStrucPack.FirmVersion = string(buffer[54+12 : 54+3+12])
	oneStrucPack.SoftVersion = string(buffer[57+12 : 57+3+12])
	oneStrucPack.OutsideAntena = buffer[60+12]
	oneStrucPack.InsideAntena = buffer[61+12]
	oneStrucPack.ServerIpPort = Iphex2string(buffer[62+12:62+6+12], 6)
	oneStrucPack.OtherStatus = string(buffer[68+12 : 68+5+12])
	oneStrucPack.Dotime = time.Now().Local()
	oneStrucPack.Refeshflag = 1

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
	StrucPack.FirmSerailno = string(buffer[2 : 2+6+12])
	StrucPack.EquipID = string(buffer[10+12 : 10+30+12])
	StrucPack.PhoneNum = string(buffer[40+12 : 40+11+12])
	StrucPack.SysEnergy = buffer[40+11+12]
	StrucPack.ConnStatus = buffer[40+11+1+12]
	StrucPack.ReadWriterStatus = buffer[53+12]
	StrucPack.FirmVersion = string(buffer[54+12 : 54+3+12])
	StrucPack.SoftVersion = string(buffer[57+12 : 57+3+12])
	StrucPack.OutsideAntena = buffer[60+12]
	StrucPack.InsideAntena = buffer[61+12]
	StrucPack.ServerIpPort = Iphex2string(buffer[62+12:62+6+12], 6)
	StrucPack.OtherStatus = string(buffer[68+12 : 68+5+12])
	StrucPack.Dotime = time.Now().Local()

	dbInsertheart(StrucPack)

	ret := isfoundserialinpool(buffer)
	if ret == -1 {
		glog.V(1).Infoln("客户端没有连接上")

		return 1
	}
	linesinfos[ret].HeartInfo = StrucPack

	buffer_heartback := make([]byte, 256)
	buffer_heartback[0] = 0xEE
	buffer_heartback[1] = 0x81
	copy(buffer_heartback[2:2+6+12], buffer[2:2+6+12])
	buffer_heartback[8+12] = 0x00
	buffer_heartback[9+12] = 0x01
	buffer_heartback[10+12] = 0x00
	buffer_heartback[11+12] = CalcChecksum(buffer_heartback, 12+12)
	sendcount, err := linesinfos[ret].Conn.Write(buffer_heartback[:12+12])
	if err != nil {
		glog.V(3).Infoln("无法发送心跳返回数据包", hex.EncodeToString(buffer_heartback[:12+12]), sendcount)

		return 2
	}
	glog.V(4).Infoln("成功发送心跳返回数据包", hex.EncodeToString(buffer_heartback[:12+12]), sendcount)
	return 0
}

type CONNINFO struct {
	Conn         net.Conn
	FirmSerialno [6 + 12]byte
	ClientIp     string
	Clientport   string
	Dotime       time.Time
	Alive        int
	HeartInfo    PackageStruct
}

var linesinfos []CONNINFO

func isfoundserialinpool(buffer []byte) int {
	for index, value := range linesinfos {
		if bytes.Equal(value.FirmSerialno[:6+12], buffer[2:2+6+12]) == true {
			return index
		}
	}
	return -1
}
func handleConnection(conn net.Conn) {
	var onelineinfo CONNINFO
	defer func() {
		for iindex, value := range linesinfos {
			if value.Conn == conn {
				value.Alive = 0
				linesinfos[iindex].Alive = 0

			}
		}
		conn.Close()
	}()
	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			glog.V(1).Infoln(conn.RemoteAddr().String(), "连接出错: ", err)
			return
		}

		glog.V(5).Infoln(conn.RemoteAddr().String(), "->", hex.EncodeToString(buffer[:n]), n)

		ret := isfoundserialinpool(buffer)
		if ret == -1 {
			onelineinfo.Conn = conn
			copy(onelineinfo.FirmSerialno[:6+12], buffer[2:2+6+12])
			onelineinfo.ClientIp = conn.RemoteAddr().String()
			onelineinfo.Clientport = strings.Split(conn.RemoteAddr().String(), ":")[1]
			onelineinfo.Dotime = time.Now().Local()
			onelineinfo.Alive = 1
			linesinfos = append(linesinfos, onelineinfo)

		} else {
			linesinfos[ret].Dotime = time.Now().Local()
			linesinfos[ret].Conn = conn
			linesinfos[ret].ClientIp = conn.RemoteAddr().String()
			linesinfos[ret].Clientport = strings.Split(conn.RemoteAddr().String(), ":")[1]
			linesinfos[ret].Alive = 1
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
