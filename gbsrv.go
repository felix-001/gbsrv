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

func parseConsole() (host, port string) {
	_host := flag.String("host", "localhost", "host")
	_port := flag.String("port", "5061", "port")
	flag.Parse()
	return *_host, *_port
}

func newConn(host, port string) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", host+":"+port)
	log.Println("listen on", host, ":", port)
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
		VersionMajor: 2,
		VersionMinor: 0,
		Status:       200,
		Phrase:       "OK",
		From:         msg.From,
		To:           msg.To,
		CallID:       msg.CallID,
		CSeq:         msg.CSeq,
		CSeqMethod:   msg.Method,
		UserAgent:    "QVS",
		Expires:      3600,
		Via:          self.gen200Via(msg),
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
		log.Println(xmlMsg.CmdType)
		if xmlMsg.CmdType == "Catalog" {
			item := xmlMsg.DeviceList.Items[0]
			log.Println("Name:", item.Name)
			log.Println("Chid:", item.ChId)
			log.Println("Model:", item.Model)
			log.Println("Manufacturer:", item.Manufacturer)
			self.chid = item.ChId
		}
		if xmlMsg.CmdType != "Alarm" &&
			xmlMsg.CmdType != "Keepalive" &&
			xmlMsg.CmdType != "Catalog" {
			log.Println(self.remoteAddr, msg.String())
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
	//log.Println(remoteAddr, string(data))
	self.remoteAddr = remoteAddr
	msg, err := sip.ParseMsg(data[0:n])
	if err != nil {
		log.Println(err)
		return nil, err
	}
	self.lastMsg = msg
	self.to = msg.From
	self.to.Param = nil
	return msg, nil
}

func (self *SipManager) handleClient() {
	msg, err := self.fetchMsg()
	if err != nil {
		return
	}
	if msg.Method == "REGISTER" {
		self.gbid = msg.From.Uri.User
		log.Println(self.gbid, "Register")
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
			log.Println(msg)
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
	help 
	last: repeat last command
	catalog: send catalog req
	sip-raw <raw-sip-file>`)
}

func (self *SipManager) inviteAudio() {
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
	from := self.lastMsg.To
	from.Tag()
	to := self.lastMsg.From
	to.Param = nil
	msg := &sip.Msg{
		Method:  "ACK",
		Request: self.lastMsg.From.Uri,
		Via:     self.genVia(),
		From:    from,
		To:      to,
	}
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
			Name: "rport",
		},
	}
	via.Branch()
	return via
}

func (self *SipManager) newFrom() *sip.Addr {
	port, _ := strconv.Atoi(self.port)
	uri := &sip.URI{
		User: self.gbid,
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
	msg := &sip.Msg{
		Method:      method,
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

func (self *SipManager) handleCatalog(strs []string) {
	from := self.lastMsg.To
	from.Tag()
	to := self.lastMsg.From
	to.Param = nil
	callid := util.GenerateCallID()
	self.catalogCallid = callid
	catalog := &sip.Msg{
		Method:      "MESSAGE",
		Request:     self.lastMsg.From.Uri,
		From:        from,
		To:          to,
		CallID:      callid,
		CSeq:        self.cseq,
		CSeqMethod:  "MESSAGE",
		MaxForwards: 70,
		UserAgent:   "QVS",
		Payload:     self.genCatalogPayload(self.gbid),
		Via:         self.genVia(),
	}
	self.conn.WriteToUDP([]byte(catalog.String()), self.remoteAddr)
	self.cseq++
	//log.Println("send:", catalog.String())
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
			fmt.Printf("err: unsupported cmd:%s", strs[0])
		}
	}
}

func NewSipManager(conn *net.UDPConn, host, port string) *SipManager {
	manager := &SipManager{conn: conn, host: host, port: port}
	manager.cmds = map[string]handler{
		"sip-raw": manager.handleSipRaw,
		"help":    manager.handleHelp,
		"invite":  manager.handleInvite,
		"catalog": manager.handleCatalog,
	}
	return manager
}

func main() {
	log.SetFlags(log.Lshortfile)
	host, port := parseConsole()
	conn, err := newConn(host, port)
	if err != nil {
		os.Exit(1)
	}
	manager := NewSipManager(conn, host, port)
	defer conn.Close()
	go manager.handleConsole()
	for {
		manager.handleClient()
	}
}
