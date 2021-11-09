package main

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jart/gosip/sip"
	"github.com/jart/gosip/util"
	"golang.org/x/net/html/charset"
)

type SipManager struct {
	remoteAddr    *net.UDPAddr
	conn          *net.UDPConn
	cmds          map[string]handler
	msgCount      int
	lastcmd       string
	lastMsg       *sip.Msg
	cseq          int
	gbid          string
	chid          string
	host          string
	port          string
	catalogCallid string
	to            *sip.Addr
	srvSipId      string
	rawMsg        string
	verbose       bool
	dumpMsg       bool
}

type Item struct {
	ChId         string `xml:"DeviceID"`
	Name         string `xml:"Name"`
	Manufacturer string `xml:"Manufacturer"`
	Model        string `xml:"Model"`
}

type DeviceList struct {
	Num   string `xml:"Num,attr"`
	Items []Item `xml:"Item"`
}

type XmlMsg struct {
	CmdType    string     `xml:"CmdType"`
	SN         string     `xml:"SN"`
	DeviceId   string     `xml:"DeviceID"`
	SumNum     int        `xml:"SumNum"`
	DeviceList DeviceList `xml:"DeviceList,omitempty"`
}

func parseConsole() (string, string, bool) {
	_host := flag.String("host", "localhost", "host")
	_port := flag.String("port", "5061", "port")
	_verbose := flag.Bool("show-detail", true, "verbose")
	flag.Parse()
	return *_host, *_port, *_verbose
}

func newConn(host, port string) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", host+":"+port)
	fmt.Println("listen on udp", host, ":", port)
	if err != nil {
		log.Println("Can't resolve address: ", err)
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Println("Error listening:", err)
		return nil, err
	}
	return conn, nil
}

func (self *SipManager) parseXml(raw string) (*XmlMsg, error) {
	xmlMsg := &XmlMsg{}
	decoder := xml.NewDecoder(bytes.NewReader([]byte(raw)))
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(xmlMsg); err != nil {
		log.Println(err)
		return xmlMsg, err
	}
	return xmlMsg, nil
}

func (self *SipManager) gen200Via(msg *sip.Msg) *sip.Via {
	branch := ""
	if msg != nil && msg.Via != nil && msg.Via.Param != nil {
		param := msg.Via.Param.Get("branch")
		if param != nil {
			branch = param.Value
		}
	}
	//log.Println(msg.Via.Port)
	via := &sip.Via{
		Host: msg.Via.Host,
		Port: msg.Via.Port,
		Param: &sip.Param{
			Name:  "branch",
			Value: branch,
			Next: &sip.Param{
				Name:  "received",
				Value: msg.Via.Host,
				Next: &sip.Param{
					Name:  "rport",
					Value: strconv.Itoa(int(msg.Via.Port)),
				},
			},
		},
	}
	return via
}

func (self *SipManager) send200(msg *sip.Msg) {
	resp := &sip.Msg{
		Status:     200,
		Phrase:     "OK",
		From:       msg.From,
		To:         msg.To,
		CallID:     msg.CallID,
		CSeq:       msg.CSeq,
		CSeqMethod: msg.Method,
		UserAgent:  "QVS",
		Expires:    3600,
		Via:        self.gen200Via(msg),
	}
	//log.Println("send response\n" + resp.String())
	self.conn.WriteToUDP([]byte(resp.String()), self.remoteAddr)
}

func (self *SipManager) handleMessage(msg *sip.Msg) {
	//log.Println("got MESSAGE count:", self.msgCount)
	self.msgCount++
	if msg.Payload.ContentType() == "Application/MANSCDP+xml" {
		xmlMsg, err := self.parseXml(string(msg.Payload.Data()))
		if err != nil {
			return
		}
		//log.Println(xmlMsg.CmdType)
		if xmlMsg.CmdType == "Catalog" {
			if len(xmlMsg.DeviceList.Items) > 0 {
				item := xmlMsg.DeviceList.Items[0]
				log.Println("Name:", item.Name)
				log.Println("Chid:", item.ChId)
				log.Println("Model:", item.Model)
				log.Println("Manufacturer:", item.Manufacturer)
				self.chid = item.ChId
			} else {
				log.Println("raw msg:", msg.String())
			}
		}
		if xmlMsg.CmdType != "Alarm" &&
			xmlMsg.CmdType != "Keepalive" &&
			xmlMsg.CmdType != "Catalog" {
			log.Println(self.remoteAddr, msg.String())
		}
		if self.verbose {
			log.Println(xmlMsg.DeviceId, xmlMsg.CmdType)
		}
	}
	self.send200(msg)
}

func (self *SipManager) fetchMsg() (*sip.Msg, error) {
	data := make([]byte, 2048)
	n, remoteAddr, err := self.conn.ReadFromUDP(data)
	if err != nil {
		log.Println("failed to read UDP msg because of ", err.Error())
		return nil, err
	}
	if self.dumpMsg && n > 4 {
		log.Println(remoteAddr, n, string(data))
	}
	self.remoteAddr = remoteAddr
	self.rawMsg = string(data[0:n])
	msg, err := sip.ParseMsg(data[0:n])
	if err != nil {
		if n > 4 {
			log.Println(err, "n:", n, "raw:", string(data))
		} else {
			//log.Println("recevied only", n, "bytes, hik ipc sometimes sent `\\r\\n`")
		}
		return nil, err
	}
	self.lastMsg = msg
	self.to = msg.From
	return msg, nil
}

func (self *SipManager) handleClient() {
	msg, err := self.fetchMsg()
	if err != nil {
		return
	}
	if msg.Method == "REGISTER" {
		self.gbid = msg.From.Uri.User
		self.srvSipId = msg.Request.User
		log.Println(self.gbid, "Register", "Expires:", msg.Expires)
		self.send200(msg)
		return
	}

	if msg.Method == "MESSAGE" {
		self.handleMessage(msg)
		return
	}

	if msg.IsResponse() {
		if msg.CSeqMethod == "MESSAGE" &&
			msg.CallID == self.catalogCallid {
			log.Println("Catalog response", msg.Status)
		} else {
			//log.Println(self.rawMsg)
		}
		if msg.CSeqMethod == "INVITE" {
			self.sendAck()
		}

		return
	}
}

type handler func([]string)

func ReadFile(file string) []byte {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Println("read fail", err)
		return []byte("")
	}
	return data
}

func (self *SipManager) handleSipRaw(strs []string) {
	data := ReadFile(strs[1][:len(strs[1])-1])
	self.conn.WriteToUDP(data, self.remoteAddr)
	fmt.Println("send:", string(data))
}

func (self *SipManager) handleHelp(strs []string) {
	fmt.Println(`support cmds: 
	h/help: this help
	last: repeat last command
	catalog: send catalog req
	dump-msg: dump sip message
	invite <audio/video>
	q/quit/exit: exit
	sip-raw <raw-sip-file>`)
}

func (self *SipManager) genSdp() []byte {
	sdp := "v=0\r\n" +
		"o=" + self.srvSipId + " 0 0 IN IP4 " + self.host + "\r\n" +
		"s=Talk\r\n" +
		"c=IN IP4 " + self.host + "\r\n" +
		"t=0 0\r\n" +
		"m=audio 9001 RTP/AVP 8\r\n" +
		"a=sendrecv\r\n" +
		"a=rtpmap:8 PCMA/8000\r\n" +
		"y=0200000001\r\n" // +
		//"f=v/////a/1/8/1\r\n"

	return []byte(sdp)
}

func (self *SipManager) waitRtpOverUdp() {
	addr, err := net.ResolveUDPAddr("udp", self.host+":9001")
	log.Println("listen on", self.host, ":9001")
	if err != nil {
		log.Println("Can't resolve address: ", err)
		return
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Println("Error listening:", err)
		return
	}
	data := make([]byte, 2048)
	_, remoteAddr, err := conn.ReadFromUDP(data)
	if err != nil {
		log.Println("failed to read UDP msg because of ", err.Error())
		return
	}
	log.Println("got rtp data:", remoteAddr, data[:64])
}

func (self *SipManager) inviteAudio() {
	msg := self.newSipReqMsg("INVITE")
	msg.From.Uri.User = "31011500002000000001"
	msg.Request.User = "34020000001370000001" //self.chid
	msg.Request.Host = "100.100.72.253"
	msg.Request.Port = 5060
	msg.To.Uri.User = "34020000001370000001" //self.chid
	payload := &sip.MiscPayload{
		T: "APPLICATION/SDP",
		D: self.genSdp(),
	}
	msg.Payload = payload
	self.conn.WriteToUDP([]byte(msg.String()), self.remoteAddr)
	log.Println(msg.String())
	go self.waitRtpOverUdp()
}

func (self *SipManager) inviteVideo() {

}

func (self *SipManager) handleInvite(strs []string) {
	if strs[1][:len(strs[1])-1] == "audio" {
		self.inviteAudio()
	} else {
		self.inviteVideo()
	}
}

func (self *SipManager) sendAck() {
	msg := self.newSipAckMsg("ACK")
	msg.CallID = self.lastMsg.CallID
	msg.CSeq = self.lastMsg.CSeq
	self.conn.WriteToUDP([]byte(msg.String()), self.remoteAddr)
	//log.Println("send:", msg.String())
}

func (self *SipManager) genCatalogPayload(gbid string) *sip.MiscPayload {
	xml := "<?xml version=\"1.0\" encoding=\"GB2312\"?>\r\n" +
		"<Query>\r\n" +
		"<CmdType>Catalog</CmdType>\r\n" +
		"<SN>419315752</SN>\r\n" +
		"<DeviceID>" + gbid + "</DeviceID>\r\n" +
		"</Query>\r\n"
	payload := &sip.MiscPayload{}
	payload.D = []byte(xml)
	payload.T = "Application/MANSCDP+xml"

	return payload
}

func RandStringRunes(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func (self *SipManager) genVia() *sip.Via {
	port, _ := strconv.Atoi(self.port)
	via := &sip.Via{
		Host: self.host,
		Port: uint16(port),
		Param: &sip.Param{
			Name:  "branch",
			Value: "z9hG4bK180541459",
			Next: &sip.Param{
				Name: "rport",
			},
		},
	}
	//via.Branch()
	return via
}

func (self *SipManager) newFrom() *sip.Addr {
	port, _ := strconv.Atoi(self.port)
	uri := &sip.URI{
		User: "31011500002000000001",
		Host: self.host,
		Port: uint16(port),
	}
	from := &sip.Addr{
		Uri: uri,
	}
	from.Tag()
	return from
}

func (self *SipManager) newAckFrom() *sip.Addr {
	port, _ := strconv.Atoi(self.port)
	uri := &sip.URI{
		User: self.srvSipId,
		Host: self.host,
		Port: uint16(port),
	}
	from := &sip.Addr{
		Uri: uri,
	}
	from.Tag()
	return from
}

func (self *SipManager) newSipMsg(method string) *sip.Msg {
	from := self.newFrom()
	if self.to == nil {
		log.Println("self.to is nil")
		return nil
	}
	self.to.Param = nil
	request := *from.Uri
	msg := &sip.Msg{
		Method:      method,
		Request:     &request,
		From:        from,
		To:          self.to,
		Via:         self.genVia(),
		CSeqMethod:  method,
		MaxForwards: 70,
		UserAgent:   "QVS",
	}
	self.cseq++
	return msg
}

func (self *SipManager) newSipAckMsg(method string) *sip.Msg {
	contact := &sip.Addr{
		Uri: self.lastMsg.From.Uri,
	}
	msg := &sip.Msg{
		Method:      method,
		Request:     self.lastMsg.To.Uri,
		From:        self.lastMsg.From,
		To:          self.lastMsg.To,
		Via:         self.genVia(),
		CSeqMethod:  method,
		Contact:     contact,
		MaxForwards: 70,
		UserAgent:   "QVS",
	}
	self.cseq++
	return msg
}

func (self *SipManager) newSipReqMsg(method string) *sip.Msg {
	msg := self.newSipMsg(method)
	msg.CallID = util.GenerateCallID()
	msg.CSeq = self.cseq
	return msg
}

func (self *SipManager) handleCatalog(strs []string) {
	msg := self.newSipReqMsg("MESSAGE")
	if self.gbid == "" {
		log.Println("no gbid, not register yet")
		return
	}
	msg.Payload = self.genCatalogPayload(self.gbid)
	self.catalogCallid = msg.CallID
	self.conn.WriteToUDP([]byte(msg.String()), self.remoteAddr)
	//log.Println("send:", msg.String())
}

func (self *SipManager) handleConsole() {
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			continue
		}
		if len(line) == 1 {
			continue
		}
		line = strings.TrimSpace(line)
		if line[:len(line)-1] == "last" {
			line = self.lastcmd
		}
		strs := strings.Split(line, " ")
		cmdstr := strs[0]
		if len(strs) == 1 {
			cmdstr = strs[0][:len(strs[0])-1]
		}
		if _, ok := self.cmds[cmdstr]; ok {
			self.cmds[cmdstr](strs)
			self.lastcmd = line
		} else {
			fmt.Printf("err: unsupported cmd: %s", strs[0])
		}
	}
}

func (self *SipManager) quit(strs []string) {
	os.Exit(0)
}

func (self *SipManager) setDumpMsg(strs []string) {
	log.Println("dump sip raw message on")
	self.dumpMsg = true
}

func NewSipManager(conn *net.UDPConn, host, port string, verbose bool) *SipManager {
	manager := &SipManager{conn: conn, host: host, port: port, verbose: verbose}
	manager.cmds = map[string]handler{
		"sip-raw":  manager.handleSipRaw,
		"help":     manager.handleHelp,
		"h":        manager.handleHelp,
		"invite":   manager.handleInvite,
		"catalog":  manager.handleCatalog,
		"q":        manager.quit,
		"quit":     manager.quit,
		"exit":     manager.quit,
		"dump-msg": manager.setDumpMsg,
	}
	return manager
}

// TODO
// ack request/from/to 分别都是哪个id
// 1. sip raw msg写文件
// 2. gui， 播放视频流
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	host, port, verbose := parseConsole()
	conn, err := newConn(host, port)
	if err != nil {
		os.Exit(1)
	}
	manager := NewSipManager(conn, host, port, verbose)
	defer conn.Close()
	go manager.handleConsole()
	for {
		manager.handleClient()
	}
}
