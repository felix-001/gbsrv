package main

import (
	"flag"
	"gbsrv/client"
	"gbsrv/server"
	"log"
)

const (
	SipSrvPort = "5061"
)

func main() {
	log.SetFlags(log.Lshortfile)
	mode := flag.String("mode", "srv", "运行模式(srv: 国标服务器模式 cli: 国标客户端)")
	if *mode == "srv" {
		srv := server.New(SipSrvPort)
		srv.Run()
	} else {
		cli := client.New()
		cli.Run()
	}
}
