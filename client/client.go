package client

import (
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/jart/gosip/sip"
	"github.com/jart/gosip/util"
)

func SendMessage(srvId, srvAddr, devId string) error {
	ss := strings.Split(srvAddr, ":")
	host := ss[0]
	port, err := strconv.ParseInt(ss[1], 10, 32)
	if err != nil {
		return err
	}
	reg := &sip.Msg{
		CSeq:       1,
		CallID:     "2052379263",
		Method:     sip.MethodRegister,
		CSeqMethod: sip.MethodRegister,
		UserAgent:  "gbcli v0.01",
		Request: &sip.URI{
			Scheme: "sip",
			User:   srvId,
			Host:   srvAddr,
		},
		Via: &sip.Via{
			Version:  "2.0",
			Protocol: "SIP",
			Host:     host,
			Port:     uint16(port),
			Param:    &sip.Param{Name: "branch", Value: util.GenerateBranch()},
		},
		Contact: &sip.Addr{
			Uri: &sip.URI{
				User: devId,
				Host: host,
				Port: uint16(port),
			},
		},
		From: &sip.Addr{
			Uri: &sip.URI{
				User: devId,
				Host: host,
			}, Param: &sip.Param{Name: "tag", Value: util.GenerateTag()},
		},
		To: &sip.Addr{
			Uri: &sip.URI{
				User: srvId,
				Host: host,
			},
		},
		Expires: 3600,
	}
	rAddr, err := net.ResolveUDPAddr("udp", srvAddr)
	if err != nil {
		return err
	}
	conn, err := net.DialUDP("udp", nil, rAddr)
	if err != nil {
		log.Printf("err = %#v\n", err.Error())
		return err
	}
	log.Println("send msg:\n", reg.String())
	if _, err := conn.Write([]byte(reg.String())); err != nil {
		log.Printf("send msg failed, err = %#v\n", err)
		return err
	}
	buf := make([]byte, 10240)
	conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	n, err := conn.Read(buf)
	if n == 0 || err != nil {
		log.Println("err n:", n, "err:", err)
		return err
	}
	msg, err := sip.ParseMsg(buf[:n])
	if err != nil {
		log.Printf("parse msg failed, err =%+v\n", err)
		return err
	}
	log.Println("recv msg:\n", msg)
	return nil
}
