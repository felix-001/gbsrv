package main

import (
	"fmt"
	"gbsrv/server"
	"log"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	bootstrap "github.com/asticode/go-astilectron-bootstrap"
)

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
			//{Role: astilectron.MenuItemRoleClose},
			{Role: astilectron.MenuItemRolePaste},
		},
		Role: astilectron.MenuItemRolePaste,
	}
	srv = server.New(SipSrvPort, SrvGbId, branch, onKeepAlive, onRegister, onUnRegister, onCatalog, onDevGbId, onPeerAddr)
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
