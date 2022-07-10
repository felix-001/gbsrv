package main

import (
	"fmt"
	"gbsrv/server"
	"log"
	"path"
	"runtime"

	"github.com/webview/webview"
)

const (
	SipSrvPort = "5061"
	SrvGbId    = "31011500002000000001"
	branch     = "z9hG4bK180541459"
)

var w webview.WebView

func pathToStartPage() string {
	_, currentFile, _, _ := runtime.Caller(0)
	dir := path.Dir(currentFile)
	return path.Join(dir, "index.html")
}

func startServer(param string) {
	srv := server.New(SipSrvPort, SrvGbId, branch)
	go srv.Run()

	data := "请按照如下参数配置摄像机:"
	data += fmt.Sprintf("<div>国标服务器编码: %s</div>", SrvGbId)
	data += fmt.Sprintf("<div>国标服务器IP: %s</div>", srv.GetHost())
	data += fmt.Sprintf("<div>国标服务器端口: %s</div>", srv.GetPort())
	w.Eval(fmt.Sprintf("setDivContent('output', '%s')", data))
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime)
	w = webview.New(false)
	defer w.Destroy()
	w.SetTitle("GB28181工具")
	w.SetSize(400, 500, webview.HintNone)
	w.Navigate("file://" + pathToStartPage())
	w.Bind("startServer", startServer)
	w.Run()
}
