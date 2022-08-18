package main

import (
	"encoding/json"
	"fmt"
	"gbsrv/client"
	"gbsrv/server"
	"io"
	"log"
	"os"

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

type SipSrvInfo struct {
	SipSrvAddr  string `json:"sipSrvAddr"`
	SipSrvId    string `json:"sipSrvId"`
	SipId       string `json:"sipId"`
	CboxChecked bool   `json:"cboxChecked"`
}

func handleMessages(_ *astilectron.Window, m bootstrap.MessageIn) (payload interface{}, err error) {
	switch m.Name {
	case "start":
		log.Println("got start msg from js")
		payload := struct {
			CboxChecked bool `json:"cboxChecked"`
		}{}
		err = json.Unmarshal(m.Payload, &payload)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		go srv.Run(payload.CboxChecked)
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
		if srv != nil {
			srv.Quit()
		}
	case "sendMessage":
		log.Println("got message sendMessage", string(m.Payload))
		payload := SipSrvInfo{}
		err = json.Unmarshal(m.Payload, &payload)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		log.Printf("%+v\n", payload)
		if payload.CboxChecked {
			f, err := os.OpenFile("out.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm)
			if err != nil {
				return nil, err
			}
			defer func() {
				f.Close()
			}()

			multiWriter := io.MultiWriter(os.Stdout, f)
			log.SetOutput(multiWriter)

			log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
		}
		if err := client.SendMessage(payload.SipSrvId, payload.SipSrvAddr, payload.SipId, w); err != nil {
			return nil, err
		}
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
		log.Println(fmt.Errorf("sending register event failed: %w", err))
	}
}

func onUnRegister(count int) {
	data := fmt.Sprintf("%d", count)
	err := bootstrap.SendMessage(w, "unregister", data, func(m *bootstrap.MessageIn) {})
	if err != nil {
		log.Println(fmt.Errorf("sending unregister event failed: %w", err))
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
		log.Println(fmt.Errorf("sending catalog event failed: %w", err))
	}
}

func onDevGbId(gbId string) {
	err := bootstrap.SendMessage(w, "devGbId", gbId, func(m *bootstrap.MessageIn) {})
	if err != nil {
		log.Println(fmt.Errorf("sending devGbId event failed: %w", err))
	}
}

func onPeerAddr(peerAddr string) {
	err := bootstrap.SendMessage(w, "peerAddr", peerAddr, func(m *bootstrap.MessageIn) {})
	if err != nil {
		log.Println(fmt.Errorf("sending peerAddr event failed: %w", err))
	}
}

func onWait(_ *astilectron.Astilectron, ws []*astilectron.Window, _ *astilectron.Menu, _ *astilectron.Tray, _ *astilectron.Menu) error {
	w = ws[0]
	return nil
}
