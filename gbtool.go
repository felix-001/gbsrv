package main

import (
	"fmt"
	"log"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	bootstrap "github.com/asticode/go-astilectron-bootstrap"
)

const (
	SipSrvPort = "5061"
	SrvGbId    = "31011500002000000001"
	branch     = "z9hG4bK180541459"
	htmlAbout  = `国标调试工具.http://www.qiniu.com`
)

var (
	AppName            string
	BuiltAt            string
	VersionAstilectron string
	VersionElectron    string
	w                  *astilectron.Window
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

func main() {
	log.SetFlags(log.Ldate | log.Ltime)
	win := &bootstrap.Window{
		Homepage:       "index.html",
		MessageHandler: handleMessages,
		Options: &astilectron.WindowOptions{
			BackgroundColor: astikit.StrPtr("#333"),
			Center:          astikit.BoolPtr(true),
			Height:          astikit.IntPtr(700),
			Width:           astikit.IntPtr(700),
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
				Label:   astikit.StrPtr("About"),
				OnClick: showMenu,
			},
			{Role: astilectron.MenuItemRoleClose},
		},
	}
	err := bootstrap.Run(bootstrap.Options{
		Asset:              Asset,
		AssetDir:           AssetDir,
		AstilectronOptions: options,
		Debug:              false,
		MenuOptions:        []*astilectron.MenuItemOptions{menuOptions},
		OnWait: func(_ *astilectron.Astilectron, ws []*astilectron.Window, _ *astilectron.Menu, _ *astilectron.Tray, _ *astilectron.Menu) error {
			w = ws[0]
			return nil
		},
		RestoreAssets: RestoreAssets,
		Windows:       []*bootstrap.Window{win},
	})

	if err != nil {
		log.Fatal(fmt.Errorf("running bootstrap failed: %w", err))
	}
}
