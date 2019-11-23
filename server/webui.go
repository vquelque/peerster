package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/vquelque/Peerster/gossiper"
	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/utils"
)

func peersListHandler(gsp *gossiper.Gossiper) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			peerList := gsp.Peers.GetAllPeers()
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
			if peerAddr == gsp.PeersSocket.Address() {
				return
			}
			peerAddrChecked := utils.ToUDPAddr(peerAddr)
			if peerAddrChecked == nil {
				return
			}
			if !gsp.Peers.CheckPeerPresent(peerAddr) {
				gsp.Peers.Add(peerAddr)
			} else {
				gsp.Peers.Delete(peerAddr)
			}
		default:
			fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
		}
	})
}
func msgHandler(gsp *gossiper.Gossiper) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			var rumorMessageList []*message.RumorMessage
			rumorMessageList = gsp.UIStorage.GetAllRumors()
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
				http.Error(w, "Invalid Data", http.StatusBadRequest)
				return
			}
			messageText := r.FormValue("message")
			cliMsg := &message.Message{Text: messageText}
			go gsp.ProcessClientMessage(cliMsg)
		default:
			fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
		}
	})
}

func idHandler(gsp *gossiper.Gossiper) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			peerID := gsp.Name
			peerIDJSON, err := json.Marshal(peerID)
			if err != nil {
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(peerIDJSON)
		}
	})
}

func contactsHandler(gsp *gossiper.Gossiper) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			contacts := gsp.Routing.GetAllRoutes()
			contactsJSON, err := json.Marshal(contacts)
			if err != nil {
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(contactsJSON)
		}
	})
}

func privateMsgHandler(gsp *gossiper.Gossiper) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			peer := r.FormValue("peer")
			m := gsp.UIStorage.GetPrivateMessagesForPeer(peer)
			mJSON, err := json.Marshal(m)
			if err != nil {
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(mJSON)
		case "POST":
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Invalid Data", http.StatusBadRequest)
				return
			}
			peer := r.FormValue("peer")
			messageText := r.FormValue("message")
			cliMsg := &message.Message{Text: messageText, Destination: peer}
			go gsp.ProcessClientMessage(cliMsg)
			http.Redirect(w, r, r.Header.Get("/privateMsg?peer="+peer), 302)
		default:
			fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
		}
	})
}

func fileUploadHandler(gsp *gossiper.Gossiper) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			filename, err := fileUploadHelper(r)
			if err != nil {
				http.Error(w, "Invalid Data", http.StatusBadRequest)
				return
				//checking whether any error occurred retrieving image
			}
			cliMsg := &message.Message{File: filename}
			go gsp.ProcessClientMessage(cliMsg)
			http.Redirect(w, r, r.Header.Get("/"), 302)
		}
	})
}

//this function returns the filename of the saved file or an error if it occurs
func fileUploadHelper(r *http.Request) (string, error) {
	r.ParseMultipartForm(5 << 20)              //limit file size to 5 MB
	file, handler, err := r.FormFile("myFile") //retrieve the file from form data
	if err != nil {
		return "", err
	}
	defer file.Close() //close the file when we finish
	//this is path which  we want to store the file
	f, err := os.OpenFile("_SharedFiles/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return "", err
	}
	defer f.Close()
	io.Copy(f, file)
	return handler.Filename, nil
}

func fileDownloadHandler(gsp *gossiper.Gossiper) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			http.Redirect(w, r, r.Header.Get("/"), 302)
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Invalid Data", http.StatusBadRequest)
				return
			}
			strMetahash := r.FormValue("metahash")
			peer := r.FormValue("peer")
			filename := r.FormValue("filename")
			metahash, err := hex.DecodeString(strMetahash)
			if err != nil || len(metahash) != sha256.Size {
				log.Print("Unable to parse hash")
				http.Redirect(w, r, r.Header.Get("/"), 302)
				return
			}
			cliMsg := &message.Message{File: filename, Destination: peer, Request: metahash}
			go gsp.ProcessClientMessage(cliMsg)
		}
	})
}

func fileSearchHandler(gsp *gossiper.Gossiper) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Invalid Data", http.StatusBadRequest)
				return
			}
			keywordStr := r.FormValue("keywords")
			budget, err := strconv.ParseUint(r.FormValue("budget"), 10, 64)
			if err != nil || len(keywordStr) <= 0 {
				return
			}
			keywords := strings.Split(keywordStr, ",")
			cliMsg := &message.Message{Keywords: keywords, Budget: budget}
			go gsp.ProcessClientMessage(cliMsg)
			http.Redirect(w, r, r.Header.Get("/"), 302)
		}
	})
}

func searchResultsHandler(gsp *gossiper.Gossiper) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			gsp.UIStorage.DownloadableFiles.Lock.RLock()
			dFiles := gsp.UIStorage.DownloadableFiles.Downloadable
			filesJSON, err := json.Marshal(dFiles)
			gsp.UIStorage.DownloadableFiles.Lock.RUnlock()
			if err != nil {
				fmt.Printf("ERROR PARSING FILE MAP")
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(filesJSON)
		}
	})
}

// StartUIServer starts the UI server
func StartUIServer(UIPort int, gsp *gossiper.Gossiper) *http.Server {

	UIPortStr := ":" + strconv.Itoa(UIPort)
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("server/")))
	mux.HandleFunc("/id", idHandler(gsp))
	mux.HandleFunc("/peers", peersListHandler(gsp))
	mux.HandleFunc("/message", msgHandler(gsp))
	mux.HandleFunc("/contacts", contactsHandler(gsp))
	mux.HandleFunc("/privateMsg", privateMsgHandler(gsp))
	mux.HandleFunc("/uploadFile", fileUploadHandler(gsp))
	mux.HandleFunc("/downloadFile", fileDownloadHandler(gsp))
	mux.HandleFunc("/searchFile", fileSearchHandler(gsp))
	mux.HandleFunc("/searchResults", searchResultsHandler(gsp))
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
