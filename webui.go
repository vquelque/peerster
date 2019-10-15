package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/vquelque/Peerster/message"
)

var tpl = template.Must(template.ParseFiles("template/index.html"))

type UIData struct {
	PeerID      string
	Peers       []string
	MessageList []string
}

func decodeJSON(w http.ResponseWriter, r *http.Request, out interface{}) error {
	data, err := ioutil.ReadAll(r.Body)
	if err == nil {
		err := json.Unmarshal(data, out)
		if err == nil {
			return nil
		} else {
			w.WriteHeader(http.StatusBadRequest)
			return err
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		return err
	}
}

func (gsp *Gossiper) peersListHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		peerList := gsp.peers.GetAllPeers()
		peerListJSON, err := json.Marshal(peerList)
		if err != nil {
			log.Print("Error sending peers list as JSON")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(peerListJSON)
	case "POST":
		var peerAddr string
		err := decodeJSON(w, r, peerAddr)
		if err != nil {
			return
		}
		if peerAddr == gsp.peersSocket.Address() {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !gsp.peers.CheckPeerPresent(peerAddr) {
			gsp.peers.Add(peerAddr)
			w.WriteHeader(http.StatusOK)
			return
		}
	}
}
func (gsp *Gossiper) msgHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		peerAddr := r.URL.Query().Get("peer")
		var rumorMessageList []message.RumorMessage
		if (peerAddr) == "" {
			rumorMessageList = gsp.rumors.GetAllRumors()
		} else {
			rumorMessageList = gsp.rumors.GetAllRumorsForPeer(peerAddr)
		}
		mmsgListJSON, err := json.Marshal(rumorMessageList)
		if err != nil {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mmsgListJSON)
	}
}

func (gsp *Gossiper) idHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		peerID := gsp.name
		peerIDJSON, err := json.Marshal(peerID)
		if err != nil {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(peerIDJSON)
	}
}

func StartUIServer(UIPort int, gsp *Gossiper) *http.Server {

	UIPortStr := ":" + strconv.Itoa(UIPort)
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("template/")))
	mux.HandleFunc("/id", gsp.idHandler)
	mux.HandleFunc("/peers", gsp.peersListHandler)
	mux.HandleFunc("/messages", gsp.msgHandler)
	server := &http.Server{Addr: UIPortStr, Handler: mux}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
			return
		}
		fmt.Printf("UI server started at address 127.0.0.1:%s", UIPortStr)
	}()
	return server
}
