package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/utils"
)

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
		http.Redirect(w, r, r.Header.Get("/"), 302)
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}
		peerAddr := r.FormValue("peerAddr")
		if peerAddr == gsp.peersSocket.Address() {
			return
		}
		peerAddrChecked := utils.ToUDPAddr(peerAddr)
		if peerAddrChecked == nil {
			return
		}
		if !gsp.peers.CheckPeerPresent(peerAddr) {
			gsp.peers.Add(peerAddr)
		} else {
			gsp.peers.Delete(peerAddr)
		}
	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}
}
func (gsp *Gossiper) msgHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		var rumorMessageList []message.RumorMessage
		rumorMessageList = gsp.rumors.GetAllRumors()
		mmsgListJSON, err := json.Marshal(rumorMessageList)
		if err != nil {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mmsgListJSON)
	case "POST":
		http.Redirect(w, r, r.Header.Get("/"), 302)
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}
		messageText := r.FormValue("message")
		cliMsg := &message.Message{Text: messageText}
		gsp.ProcessClientMessage(cliMsg)
	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
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

func (gsp *Gossiper) contactsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		contacts := gsp.routing.GetAllRoutes()
		contactsJSON, err := json.Marshal(contacts)
		if err != nil {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(contactsJSON)
	}
}

// StartUIServer starts the UI server
func StartUIServer(UIPort int, gsp *Gossiper) *http.Server {

	UIPortStr := ":" + strconv.Itoa(UIPort)
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("template/")))
	mux.HandleFunc("/id", gsp.idHandler)
	mux.HandleFunc("/peers", gsp.peersListHandler)
	mux.HandleFunc("/message", gsp.msgHandler)
	mux.HandleFunc("/contacts", gsp.contactsHandler)
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
