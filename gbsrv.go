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
	host = *flag.String("host", "localhost", "host")
	port = *flag.String("port", "5061", "port")
	flag.Parse()
	return
}

func newConn() (*net.UDPConn, error) {
	host, port := parseConsole()
	addr, err := net.ResolveUDPAddr("udp", host+":"+port)
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
		Via: &sip.Via{
			Host: msg.Via.Host,
			Port: msg.Via.Port,
			Param: &sip.Param{
				Name:  "rport",
				Value: strconv.Itoa(int(msg.Via.Port)),
				Next: &sip.Param{
					Name:  "received",
					Value: msg.Via.Host,
					Next: &sip.Param{
						Name:  "branch",
						Value: branch,
					},
				},
			},
		},
	}
	log.Println("message is " + resp.String())
	conn.WriteToUDP([]byte(resp.String()), remoteAddr)
}

func handleClient(conn *net.UDPConn) {
	data := make([]byte, 1024)
	_, remoteAddr, err := conn.ReadFromUDP(data)
	if err != nil {
		log.Println("failed to read UDP msg because of ", err.Error())
		return
	}
	log.Println(remoteAddr, string(data))
	msg, err := sip.ParseMsg(data)
	if err != nil {
		log.Println(err)
		//return
	}
	log.Println(msg)
	//if msg.Method == "Register" {
	send200(msg, conn, remoteAddr)
	return
	//}
}

func main() {
	conn, err := newConn()
	if err != nil {
		os.Exit(1)
	}
	defer conn.Close()
	for {
		handleClient(conn)
	}
}
