package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	bootstrap "github.com/asticode/go-astilectron-bootstrap"
)

const (
	SipSrvPort = "5061"
	SrvGbId    = "31011500002000000001"
	branch     = "z9hG4bK180541459"
	htmlAbout  = `Welcome on <b>Astilectron</b> demo!<br> This is using the bootstrap and the bundler.`
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

func main() {
	log.SetFlags(log.Ldate | log.Ltime)
	err := bootstrap.Run(bootstrap.Options{
		Asset:    Asset,
                AssetDir: AssetDir,
		AstilectronOptions: astilectron.Options{
			AppName:            AppName,
			AppIconDarwinPath:  "resources/icon.icns",
			AppIconDefaultPath: "resources/icon.png",
			SingleInstance:     true,
			VersionAstilectron: VersionAstilectron,
			VersionElectron:    VersionElectron,
		},
		Debug: false,
		MenuOptions: []*astilectron.MenuItemOptions{{
			Label: astikit.StrPtr("File"),
			SubMenu: []*astilectron.MenuItemOptions{
				{
					Label: astikit.StrPtr("About"),
					OnClick: func(e astilectron.Event) (deleteListener bool) {
						if err := bootstrap.SendMessage(w, "about", htmlAbout, func(m *bootstrap.MessageIn) {
							// Unmarshal payload
							var s string
							if err := json.Unmarshal(m.Payload, &s); err != nil {
								log.Println(fmt.Errorf("unmarshaling payload failed: %w", err))
								return
							}
							log.Printf("About modal has been displayed and payload is %s!\n", s)
						}); err != nil {
							log.Println(fmt.Errorf("sending about event failed: %w", err))
						}
						return
					},
				},
				{Role: astilectron.MenuItemRoleClose},
			},
		}},
		OnWait: func(_ *astilectron.Astilectron, ws []*astilectron.Window, _ *astilectron.Menu, _ *astilectron.Tray, _ *astilectron.Menu) error {
			w = ws[0]
			go func() {
				time.Sleep(5 * time.Second)
				if err := bootstrap.SendMessage(w, "check.out.menu", "Don't forget to check out the menu!"); err != nil {
					log.Println(fmt.Errorf("sending check.out.menu event failed: %w", err))
				}
			}()
			return nil
		},
		RestoreAssets: RestoreAssets,
		Windows: []*bootstrap.Window{{
			Homepage:       "index.html",
			MessageHandler: handleMessages,
			Options: &astilectron.WindowOptions{
				BackgroundColor: astikit.StrPtr("#333"),
				Center:          astikit.BoolPtr(true),
				Height:          astikit.IntPtr(700),
				Width:           astikit.IntPtr(700),
			},
		}},
	})

	if err != nil {
		log.Fatal(fmt.Errorf("running bootstrap failed: %w", err))
	}
}
