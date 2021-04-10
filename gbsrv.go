package main

import (
	"flag"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/jart/gosip/sip"
)

func parseConsole() (host, port string) {
    _host := flag.String("host", "localhost", "host")
	_port := flag.String("port", "5061", "port")
	flag.Parse()
	return *_host, *_port
}

func newConn() (*net.UDPConn, error) {
	host, port := parseConsole()
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

func send200(msg *sip.Msg, conn *net.UDPConn, remoteAddr *net.UDPAddr) {
	branch := ""
	if msg != nil && msg.Via != nil && msg.Via.Param != nil {
		param := msg.Via.Param.Get("branch")
		if param != nil {
			branch = param.Value
		}
	}
	resp := &sip.Msg{
		VersionMajor: 2,
		VersionMinor: 0,
		Status:       200,
		Phrase:       "OK",
		From:         msg.From,
		To:           msg.To,
		CallID:       msg.CallID,
		CSeq:         msg.CSeq,
        CSeqMethod:   "REGISTER",
        UserAgent:    "QVS",
        Expires:      3600,
		Via: &sip.Via{
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
		},
	}
	log.Println("send response\n" + resp.String())
	conn.WriteToUDP([]byte(resp.String()), remoteAddr)
}

var count = 0

func handleClient(conn *net.UDPConn) {
	data := make([]byte, 1024)
	n, remoteAddr, err := conn.ReadFromUDP(data)
	if err != nil {
		log.Println("failed to read UDP msg because of ", err.Error())
		return
	}
	log.Println(remoteAddr, string(data))
    msg, err := sip.ParseMsg(data[0:n])
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(msg)
	if msg.Method == "REGISTER" {
		send200(msg, conn, remoteAddr)
		return
	}
    if msg.Method == "MESSAGE" {
        log.Println("got MESSAGE count:", count)
        count++
    }
}

func main() {
	log.SetFlags(log.Lshortfile)
	conn, err := newConn()
	if err != nil {
		os.Exit(1)
	}
	defer conn.Close()
	for {
		handleClient(conn)
	}
}
