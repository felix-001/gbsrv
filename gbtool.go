package main

import (
	"gbsrv/server"
	"log"
)

const (
	SipSrvPort = "5062"
	SrvGbId    = "31011500002000000001"
	branch     = "z9hG4bK180541459"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime)
	srv := server.New(SipSrvPort, SrvGbId, branch)
	srv.Run()
}
