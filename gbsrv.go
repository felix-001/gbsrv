package main

import (
	"flag"
	"log"
	"net"
	"os"

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
		return
	}
	log.Println(msg)
	conn.WriteToUDP([]byte("ok"), remoteAddr)
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
