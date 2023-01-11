package server

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/jart/gosip/sip"
	"github.com/jart/gosip/util"
	"golang.org/x/net/html/charset"
)

var (
	errInvalidMsg   = errors.New("invalid msg")
	registerResp401 = "SIP/2.0 401 Unauthorized\r\n" +
		"Via: SIP/2.0/UDP 127.0.0.1:56536;rport=56536;received=127.0.0.1;branch=z9hG4bK2024071795\r\n" +
		"From: <sip:31011500991180021627@3101010099>;tag=1205108216\r\n" +
		"To: <sip:31011500991180021627@3101010099>\r\n" +
		"CSeq: 1 REGISTER\r\n" +
		"Call-ID: 707078007@3101010099\r\n" +
		"Contact: <sip:31011500991180021627@127.0.0.1:56536>\r\n" +
		"User-Agent: QVS/2.0.0(Lion)\r\n" +
		"Content-Length: 0\r\n" +
		"WWW-Authenticate: Digest realm=\"3402000000\",qop=\"auth\",nonce=\"2717a8443b332b59955bfec36d4438da\"\r\n\r\n"
	registerResp200 = "SIP/2.0 200 OK\r\n" +
		"Via: SIP/2.0/UDP 127.0.0.1:56536;rport=38976;received=127.0.0.1;branch=z9hG4bK1216505445\r\n" +
		"From: <sip:31011500991180021627@3101010099>;tag=1205108216\r\n" +
		"To: <sip:31011500991180021627@3101010099>\r\n" +
		"CSeq: 2 REGISTER\r\n" +
		"Call-ID: 2000949065@3101010099\r\n" +
		"Contact: <sip:31011500991180021627@127.0.0.1:56536>\r\n" +
		"User-Agent: QVS/2.0.0(Lion)\r\n" +
		"Expires: 3600\r\n" +
		"Content-Length: 0\r\n\r\n"
	msgResp200 = "SIP/2.0 200 OK\r\n" +
		"Via: SIP/2.0/UDP 127.0.0.1:38976;rport=56536;received=127.0.0.1;branch=z9hG4bK541497536\r\n" +
		"From: <sip:31011500991180021627@3101010099>;tag=1601296758\r\n" +
		"To: <sip:31011500002000000001@23.248.173.9:5061>\r\n" +
		"CSeq: 1766 MESSAGE\r\n" +
		"Call-ID: 72112810@3101010099\r\n" +
		"User-Agent: QVS/2.0.0(Lion)\r\n" +
		"Content-Length: 0\r\n\r\n"
)

type Server struct {
	port           string
	conn           *net.UDPConn
	remoteAddr     *net.UDPAddr
	showRemoteAddr bool
	srvGbId        string
	host           string
	branch         string
	catalogCallid  string
	cseq           int
	catalogResp200 bool
	showUA         bool
	isRegistered   bool
	isOnline       bool
	isCatalogResp  bool
	keepAliveCnt   int
	catalogCnt     int
	registerCnt    int
	unRegisterCnt  int
}

func New(port, srvGbId, branch string) *Server {
	return &Server{
		port:           port,
		showRemoteAddr: true,
		srvGbId:        srvGbId,
		branch:         branch,
		cseq:           0,
		catalogResp200: true,
		showUA:         true,
		isRegistered:   false,
		isOnline:       false,
		isCatalogResp:  false,
		host:           getOutboundIP().String(),
	}
}

func (s *Server) newConn() error {
	addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:"+s.port)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	s.conn = conn
	return nil
}

func (s *Server) fetchMsg() (*sip.Msg, error) {
	data := make([]byte, 2048)
	n, remoteAddr, err := s.conn.ReadFromUDP(data)
	if err != nil {
		return nil, err
	}
	if n == 4 {
		// 海康的设备有的时候发送4个字节的无用数据过来("\r\n")
		return nil, errInvalidMsg
	}
	if s.showRemoteAddr {
		log.Println("摄像机地址:", remoteAddr)
		s.showRemoteAddr = false
	}
	s.remoteAddr = remoteAddr
	msg, err := sip.ParseMsg(data[0:n])
	return msg, err
}

func (s *Server) newFrom() *sip.Addr {
	port, _ := strconv.Atoi(s.port)
	uri := &sip.URI{
		User: s.srvGbId,
		Host: s.host,
		Port: uint16(port),
	}
	from := &sip.Addr{
		Uri: uri,
	}
	from.Tag()
	return from
}

func (s *Server) newVia() *sip.Via {
	//port, _ := strconv.Atoi(s.port)
	via := &sip.Via{
		Host: "127.0.0.1", //s.host,
		Port: 1123,        //uint16(port),
		Param: &sip.Param{
			Name:  "branch",
			Value: s.branch,
			Next: &sip.Param{
				Name: "rport",
			},
		},
	}
	return via
}

func (s *Server) newSipMsg(method string, callId string, cseq int, to *sip.Addr) *sip.Msg {
	from := s.newFrom()
	request := *from.Uri
	msg := &sip.Msg{
		Method:      method,
		Request:     &request,
		From:        from,
		To:          to,
		Via:         s.newVia(),
		CSeqMethod:  method,
		MaxForwards: 70,
		UserAgent:   "QVS",
		CallID:      callId,
		CSeq:        cseq,
	}
	return msg
}

func (s *Server) sendAck(msg *sip.Msg) error {
	newMsg := s.newSipMsg("ACK", msg.CallID, msg.CSeq, msg.From)
	if _, err := s.conn.WriteToUDP([]byte(newMsg.String()), s.remoteAddr); err != nil {
		return err
	}
	s.cseq++
	return nil
}

func (s *Server) handleRemoteResp(msg *sip.Msg) error {
	if msg.CSeqMethod != "MESSAGE" {
		log.Println("未处理的响应方法:", msg.CSeqMethod)
	}
	log.Println("[C->S] 摄像机国标ID:", msg.From.Uri.User, "Catalog响应:", msg.Status)
	if msg.Status != 200 {
		log.Println("raw msg:")
		fmt.Println(msg.String())
		s.catalogResp200 = false
	}
	return s.sendAck(msg)
}

func (s *Server) new200Via(msg *sip.Msg) *sip.Via {
	branch := ""
	if msg != nil && msg.Via != nil && msg.Via.Param != nil {
		param := msg.Via.Param.Get("branch")
		if param != nil {
			branch = param.Value
		}
	}
	via := &sip.Via{
		Host: "192.168.1.7", //msg.Via.Host,
		Port: 48301,         //msg.Via.Port,
		Param: &sip.Param{
			Name:  "branch",
			Value: branch,
			Next: &sip.Param{
				Name: "received",
				//Value: msg.Via.Host,
				Value: "10.20.21.35",
				Next: &sip.Param{
					Name:  "rport",
					Value: "50762", //Value: strconv.Itoa(int(msg.Via.Port)),
				},
			},
		},
	}
	return via
}

func (s *Server) sendResp(msg *sip.Msg) error {
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
		Via:        s.new200Via(msg),
	}
	if _, err := s.conn.WriteToUDP([]byte(resp.String()), s.remoteAddr); err != nil {
		return err
	}
	return nil
}

func (s *Server) sendMessageResp(msg *sip.Msg) error {
	/*
		resp := &sip.Msg{
			Status:     408,
			Phrase:     "Request Timeout",
			From:       msg.From,
			To:         msg.To,
			CallID:     msg.CallID,
			CSeq:       msg.CSeq,
			CSeqMethod: msg.Method,
			UserAgent:  "QVS",
			Expires:    3600,
			Via:        s.new200Via(msg),
		}
	*/
	resp, err := sip.ParseMsg([]byte(msgResp200))
	if err != nil {
		log.Println(err)
		return err
	}
	resp.CallID = msg.CallID
	resp.CSeq = msg.CSeq
	resp.Via = s.new200Via(msg)
	if _, err := s.conn.WriteToUDP([]byte(resp.String()), s.remoteAddr); err != nil {
		return err
	}
	return nil
}

func (s *Server) handleRegister(msg *sip.Msg) error {
	resp := registerResp401
	if msg.Expires == 0 {
		s.unRegisterCnt++
		log.Println("[C->S] 摄像机国标ID:", msg.From.Uri.User, "收到注销信令")
		resp = registerResp200
	} else {
		s.registerCnt++
		if s.showUA {
			log.Println("摄像机User-Agent:", msg.UserAgent)
			s.showUA = false
		}
		s.isRegistered = true
		if msg.Authorization != "" {
			log.Println("[C->S] 摄像机国标ID:", msg.From.Uri.User, "收到注册信令,带鉴权")
			resp = registerResp200
		}
		log.Println("[C->S] 摄像机国标ID:", msg.From.Uri.User, "收到注册信令,不带鉴权")
	}
	respMsg, err := sip.ParseMsg([]byte(resp))
	if err != nil {
		log.Println(err)
		return err
	}
	respMsg.CallID = msg.CallID
	respMsg.CSeq = msg.CSeq
	respMsg.Via = s.new200Via(msg)
	if _, err := s.conn.WriteToUDP([]byte(respMsg.String()), s.remoteAddr); err != nil {
		return err
	}
	return nil
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

func (s *Server) parseXml(raw string) (*XmlMsg, error) {
	xmlMsg := &XmlMsg{}
	decoder := xml.NewDecoder(bytes.NewReader([]byte(raw)))
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(xmlMsg); err != nil {
		return xmlMsg, err
	}
	return xmlMsg, nil
}

func (s *Server) handleCatalog(xml *XmlMsg) {
	s.catalogCnt++
	if len(xml.DeviceList.Items) == 0 {
		log.Println("对端响应的CATALOG设备个数为0")
		return
	}
	item := xml.DeviceList.Items[0]
	log.Println("Name:", item.Name)
	log.Println("Chid:", item.ChId)
	log.Println("Model:", item.Model)
	log.Println("Manufacturer:", item.Manufacturer)
	s.isCatalogResp = true
}

func (s *Server) newCatalogPayload(gbid string) *sip.MiscPayload {
	xml := `<?xml version="1.0" encoding="GB2312"?>
<Query>
<CmdType>Catalog</CmdType>
<SN>419315752</SN>
<DeviceID>`
	xml += gbid + "</DeviceID>\r\n</Query>\r\n"
	payload := &sip.MiscPayload{}
	payload.D = []byte(xml)
	payload.T = "Application/MANSCDP+xml"

	return payload
}

func (s *Server) sendCatalogReq(remoteSipAddr *sip.Addr) {
	log.Println("[S->C] 向摄像机发送CATALOG请求")
	msg := s.newSipMsg("MESSAGE", util.GenerateCallID(), s.cseq, remoteSipAddr)
	msg.Payload = s.newCatalogPayload(remoteSipAddr.Uri.User)
	if !s.catalogResp200 {
		log.Println("发送的原始CATALOG消息:")
		fmt.Println(msg.String())
	}
	if _, err := s.conn.WriteToUDP([]byte(msg.String()), s.remoteAddr); err != nil {
		log.Fatal("send catalog err", err)
	}
	s.catalogCallid = msg.CallID
}

func (s *Server) handleSipMessage(msg *sip.Msg) error {
	if msg.Payload.ContentType() != "application/MANSCDP+xml" {
		log.Println("收到消息格式为非xml,暂不处理", msg.String())
		return nil
	}
	xmlMsg, err := s.parseXml(string(msg.Payload.Data()))
	if err != nil {
		return err
	}
	switch xmlMsg.CmdType {
	case "Catalog":
		log.Println("[C->S] 摄像机国标ID:", msg.From.Uri.User, "收到Catalog信令")
		s.handleCatalog(xmlMsg)
	case "Keepalive":
		if s.showUA {
			log.Println("摄像机User-Agent:", msg.UserAgent)
			s.showUA = false
		}
		s.isOnline = true
		s.keepAliveCnt++
		if s.keepAliveCnt%5 == 0 {
			go s.sendCatalogReq(msg.From)
			log.Println("[S->C] 摄像机国标ID:", msg.From.Uri.User, "发送catalog请求")
		}
		log.Println("[C->S] 摄像机国标ID:", msg.From.Uri.User, "收到心跳信令", s.keepAliveCnt, "次")
	case "Alarm":
		log.Println("[C->S] 摄像机国标ID:", msg.From.Uri.User, "收到心跳告警")
	}
	return s.sendMessageResp(msg)
}

func (s *Server) handleMsg(msg *sip.Msg) error {
	if msg.IsResponse() {
		return s.handleRemoteResp(msg)
	}
	switch msg.Method {
	case "REGISTER":
		return s.handleRegister(msg)
	case "MESSAGE":
		return s.handleSipMessage(msg)
	default:
		log.Println("未处理的方法:", msg.Method)
	}
	return nil
}

func getOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func (s *Server) GetHost() string {
	return s.host
}

func (s *Server) GetPort() string {
	return s.port
}

func (s *Server) Run() {
	if err := s.newConn(); err != nil {
		log.Fatal("new conn err:", err)
	}
	for {
		msg, err := s.fetchMsg()
		if err != nil {
			if err == errInvalidMsg {
				continue
			}
			log.Fatal("fetch msg err:", err, msg.String())
		}
		if err := s.handleMsg(msg); err != nil {
			log.Fatal("hand msg err:", err)
		}
	}

}
