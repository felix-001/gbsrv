package main

import (
	"fmt"
	"gbsrv/server"
	"log"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	bootstrap "github.com/asticode/go-astilectron-bootstrap"
)

const (
	SipSrvPort = "5061"
	SrvGbId    = "31011500002000000001"
	branch     = "z9hG4bK180541459"
	htmlAbout  = `国标调试工具.<br>https://www.qiniu.com`
)

var (
	AppName            string
	BuiltAt            string
	VersionAstilectron string
	VersionElectron    string
	w                  *astilectron.Window
	srv                *server.Server
)

func handleMessages(_ *astilectron.Window, m bootstrap.MessageIn) (payload interface{}, err error) {
	return
}

func showMenu(e astilectron.Event) (deleteListener bool) {
	if err := bootstrap.SendMessage(w, "about", htmlAbout, func(m *bootstrap.MessageIn) {
	}); err != nil {
		log.Println(fmt.Errorf("sending about event failed: %w", err))
	}
	return
}

func onKeepAlive(count int) {
	log.Println("发送心跳消息")
	data := fmt.Sprintf("收到摄像机心跳信令%d次", count)
	err := bootstrap.SendMessage(w, "keepalive", data, func(m *bootstrap.MessageIn) {})
	if err != nil {
		log.Println(fmt.Errorf("sending keepalive event failed: %w", err))
	}
}

func onRegister(count int) {
	data := fmt.Sprintf("收到摄像机注册信令%d次", count)
	err := bootstrap.SendMessage(w, "register", data, func(m *bootstrap.MessageIn) {})
	if err != nil {
		log.Println(fmt.Errorf("sending keepalive event failed: %w", err))
	}
}

func onUnRegister(count int) {
	data := fmt.Sprintf("收到摄像机注销信令%d次", count)
	err := bootstrap.SendMessage(w, "unregister", data, func(m *bootstrap.MessageIn) {})
	if err != nil {
		log.Println(fmt.Errorf("sending keepalive event failed: %w", err))
	}
}

func onCatalog(count int, name, chid, model, manufacturer string) {
	data := fmt.Sprintf("<div>收到摄像机CATALOG信令%d次</div>", count)
	data += fmt.Sprintf("<div>Name: %s</div>", name)
	data += fmt.Sprintf("<div>Chid: %s</div>", chid)
	data += fmt.Sprintf("<div>Model: %s</div>", model)
	data += fmt.Sprintf("<div>Manufacturer: %s</div>", manufacturer)
	err := bootstrap.SendMessage(w, "catalog", data, func(m *bootstrap.MessageIn) {})
	if err != nil {
		log.Println(fmt.Errorf("sending keepalive event failed: %w", err))
	}
}

func onDevGbId(gbId string) {
	data := fmt.Sprintf("摄像机国标ID: %s", gbId)
	err := bootstrap.SendMessage(w, "devGbId", data, func(m *bootstrap.MessageIn) {})
	if err != nil {
		log.Println(fmt.Errorf("sending keepalive event failed: %w", err))
	}
}

func onWait(_ *astilectron.Astilectron, ws []*astilectron.Window, _ *astilectron.Menu, _ *astilectron.Tray, _ *astilectron.Menu) error {
	w = ws[0]
	go srv.Run()
	data := "<div>国标服务器编码: " + SrvGbId + "</div>"
	data += "<div>国标服务器IP: " + srv.GetHost() + "</div>"
	data += "<div>国标服务器端口: " + SipSrvPort + "</div>"
	err := bootstrap.SendMessage(w, "msg", data, func(m *bootstrap.MessageIn) {})
	if err != nil {
		log.Println(fmt.Errorf("sending about event failed: %w", err))
	}
	return nil
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime)
	win := &bootstrap.Window{
		Homepage:       "index.html",
		MessageHandler: handleMessages,
		Options: &astilectron.WindowOptions{
			BackgroundColor: astikit.StrPtr("#333"),
			Center:          astikit.BoolPtr(true),
			Height:          astikit.IntPtr(330),
			Width:           astikit.IntPtr(370),
		},
	}
	options := astilectron.Options{
		AppName:            AppName,
		AppIconDarwinPath:  "resources/icon.icns",
		AppIconDefaultPath: "resources/icon.png",
		SingleInstance:     true,
		VersionAstilectron: VersionAstilectron,
		VersionElectron:    VersionElectron,
	}
	menuOptions := &astilectron.MenuItemOptions{
		Label: astikit.StrPtr("File"),
		SubMenu: []*astilectron.MenuItemOptions{
			{
				Label:   astikit.StrPtr("关于"),
				OnClick: showMenu,
			},
			{Role: astilectron.MenuItemRoleClose},
		},
	}
	srv = server.New(SipSrvPort, SrvGbId, branch, onKeepAlive, onRegister, onUnRegister, onCatalog, onDevGbId)
	err := bootstrap.Run(bootstrap.Options{
		Asset:              Asset,
		AssetDir:           AssetDir,
		AstilectronOptions: options,
		Debug:              true,
		MenuOptions:        []*astilectron.MenuItemOptions{menuOptions},
		OnWait:             onWait,
		RestoreAssets:      RestoreAssets,
		Windows:            []*bootstrap.Window{win},
	})

	if err != nil {
		log.Fatal(fmt.Errorf("running bootstrap failed: %w", err))
	}
}
