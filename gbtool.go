package main

import (
	"encoding/json"
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

type sipInfo struct {
	SrvGbid string `json:"srvGbid"`
	SrvIp   string `json:"srvIp"`
	SrvPort string `json:"srvPort"`
}

func handleMessages(_ *astilectron.Window, m bootstrap.MessageIn) (payload interface{}, err error) {
	switch m.Name {
	case "start":
		log.Println("got start msg from js")
		go srv.Run()
		info := &sipInfo{
			SrvGbid: SrvGbId,
			SrvIp:   srv.GetHost(),
			SrvPort: SipSrvPort,
		}
		jsonbody, err := json.Marshal(info)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		log.Println("jsonbody:", string(jsonbody))
		if err := bootstrap.SendMessage(w, "msg", string(jsonbody), func(m *bootstrap.MessageIn) {}); err != nil {
			log.Println(fmt.Errorf("sending about event failed: %w", err))
		}
	case "end":
	default:
	}
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
	data := fmt.Sprintf("%d", count)
	err := bootstrap.SendMessage(w, "keepalive", data, func(m *bootstrap.MessageIn) {})
	if err != nil {
		log.Println(fmt.Errorf("sending keepalive event failed: %w", err))
	}
}

func onRegister(count int) {
	data := fmt.Sprintf("%d", count)
	err := bootstrap.SendMessage(w, "register", data, func(m *bootstrap.MessageIn) {})
	if err != nil {
		log.Println(fmt.Errorf("sending keepalive event failed: %w", err))
	}
}

func onUnRegister(count int) {
	data := fmt.Sprintf("%d", count)
	err := bootstrap.SendMessage(w, "unregister", data, func(m *bootstrap.MessageIn) {})
	if err != nil {
		log.Println(fmt.Errorf("sending keepalive event failed: %w", err))
	}
}

type Catalog struct {
	Count        int    `json:"count"`
	Name         string `json:"name"`
	Chid         string `json:"chid"`
	Model        string `json:"model"`
	Manufacturer string `json:"manufacturer"`
}

func onCatalog(count int, name, chid, model, manufacturer string) {
	catalog := Catalog{
		Count:        count,
		Name:         name,
		Chid:         chid,
		Model:        model,
		Manufacturer: manufacturer,
	}
	jsonbody, err := json.Marshal(&catalog)
	if err != nil {
		log.Println(err)
		return
	}
	err = bootstrap.SendMessage(w, "catalog", string(jsonbody), func(m *bootstrap.MessageIn) {})
	if err != nil {
		log.Println(fmt.Errorf("sending keepalive event failed: %w", err))
	}
}

func onDevGbId(gbId string) {
	err := bootstrap.SendMessage(w, "devGbId", gbId, func(m *bootstrap.MessageIn) {})
	if err != nil {
		log.Println(fmt.Errorf("sending keepalive event failed: %w", err))
	}
}

func onWait(_ *astilectron.Astilectron, ws []*astilectron.Window, _ *astilectron.Menu, _ *astilectron.Tray, _ *astilectron.Menu) error {
	w = ws[0]
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
			Height:          astikit.IntPtr(500),
			Width:           astikit.IntPtr(430),
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
		Label: astikit.StrPtr("GB28181调试工具"),
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
