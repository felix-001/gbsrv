package server

import (
	"bytes"
	"encoding/xml"
	"errors"
	"log"
	"net"
	"strconv"

	"github.com/jart/gosip/sip"
	"golang.org/x/net/html/charset"
)

// TODO 超时3分钟收不到任何信令报错，退出

var (
	errInvalidMsg = errors.New("invalid msg")
)

type Server struct {
	port           string
	conn           *net.UDPConn
	remoteAddr     *net.UDPAddr
	showRemoteAddr bool
}

func New(port string) *Server {
	return &Server{port: port, showRemoteAddr: true}
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

func (s *Server) handleRemoteResp(msg *sip.Msg) {

}

func (s *Server) newVia(msg *sip.Msg) *sip.Via {
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
		Via:        s.newVia(msg),
	}
	//log.Println("send response\n" + resp.String())
	if _, err := s.conn.WriteToUDP([]byte(resp.String()), s.remoteAddr); err != nil {
		return err
	}
	return nil
}

func (s *Server) handleRegister(msg *sip.Msg) error {
	if msg.Expires == 0 {
		log.Println("摄像机国标ID:", msg.From.Uri.User, "收到注销信令")
	} else {
		log.Println("摄像机国标ID:", msg.From.Uri.User, "收到注册信令")
	}
	return s.sendResp(msg)
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

func (s Server) handleCatalog(xml *XmlMsg) {
	if len(xml.DeviceList.Items) == 0 {
		log.Println("对端响应的CATALOG设备个数为0")
		return
	}
	item := xml.DeviceList.Items[0]
	log.Println("Name:", item.Name)
	log.Println("Chid:", item.ChId)
	log.Println("Model:", item.Model)
	log.Println("Manufacturer:", item.Manufacturer)
}

func (s *Server) handleSipMessage(msg *sip.Msg) error {
	if msg.Payload.ContentType() != "Application/MANSCDP+xml" {
		log.Println("收到消息格式为非xml,暂不处理")
		return nil
	}
	xmlMsg, err := s.parseXml(string(msg.Payload.Data()))
	if err != nil {
		return err
	}
	switch xmlMsg.CmdType {
	case "Catalog":
		log.Println("摄像机国标ID:", msg.From.Uri.User, "收到Catalog信令")
		s.handleCatalog(xmlMsg)
	case "Keepalive":
		log.Println("摄像机国标ID:", msg.From.Uri.User, "收到心跳信令")
	case "Alarm":
		log.Println("摄像机国标ID:", msg.From.Uri.User, "收到心跳告警")
	}
	return s.sendResp(msg)
}

func (s *Server) handleMsg(msg *sip.Msg) error {
	if msg.IsResponse() {
		s.handleRemoteResp(msg)
	}
	switch msg.Method {
	case "REGISTER":
		if err := s.handleRegister(msg); err != nil {
			return err
		}
	case "MESSAGE":
		if err := s.handleSipMessage(msg); err != nil {
			return err
		}
	default:
		log.Println("未处理的方法:", msg.Method)
	}
	return nil
}

func (s *Server) Run() {
	if err := s.newConn(); err != nil {
		log.Fatal("new conn err:", err)
	}
	log.Printf("listen on 0.0.0.0:%s\n", s.port)
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
