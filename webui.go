package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/vquelque/Peerster/message"
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

func StartUIServer(UIPort int, gsp *Gossiper) {
	UIPortStr := ":" + strconv.Itoa(UIPort)
	router := mux.NewRouter()
	router.Handle("/", http.FileServer(http.Dir("server/template")))
	router.HandleFunc("/peers", gsp.peersListHandler)
	router.HandleFunc("/messages", gsp.msgHandler)
	router.HandleFunc("/id", gsp.idHandler)
	go http.ListenAndServe(UIPortStr, router)
}
