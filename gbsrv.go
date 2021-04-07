package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
)

func newConn() (*net.UDPConn, error) {
	host := flag.String("host", "localhost", "host")
	port := flag.String("port", "5061", "port")
	flag.Parse()
	addr, err := net.ResolveUDPAddr("udp", *host+":"+*port)
	if err != nil {
		log.Println("Can't resolve address: ", err)
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error listening:", err)
		return nil, err
	}
	return conn, nil
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

func handleClient(conn *net.UDPConn) {
	data := make([]byte, 1024)
	n, remoteAddr, err := conn.ReadFromUDP(data)
	if err != nil {
		fmt.Println("failed to read UDP msg because of ", err.Error())
		return
	}
	fmt.Println(n, remoteAddr, string(data))
	conn.WriteToUDP([]byte("ok"), remoteAddr)
}
