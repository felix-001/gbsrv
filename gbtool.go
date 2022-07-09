package main

import (
	"flag"
	"gbsrv/client"
	"gbsrv/server"
	"log"
)

const (
	SipSrvPort = "5061"
	SrvGbId    = "31011500002000000001"
	branch     = "z9hG4bK180541459"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime)
	mode := flag.String("mode", "srv", "运行模式,srv: 国标服务器 cli: 国标客户端")
	flag.Parse()
	if *mode == "srv" {
		srv := server.New(SipSrvPort, SrvGbId, branch)
		srv.Run()
	} else {
		cli := client.New()
		cli.Run()
	}
}
