package main

import (
	"flag"

	"github.com/vquelque/Peerster/constant"
	"github.com/vquelque/Peerster/gossiper"
	"github.com/vquelque/Peerster/server"
	"github.com/vquelque/Peerster/utils"
)

const defaultAntiEntropy = 10 //in seconds
const defaultRTimer = 0       //in seconds

func main() {
	uiPort := flag.Int("UIPort", 8080, "Port for the UI client (default 8080)")
	gossipAddr := flag.String("gossipAddr", "", "ip:port for the gossiper")
	name := flag.String("name", "", "Name of the gossiper")
	peersList := flag.String("peers", "", "Comma separated list of peers of the form ip:port")
	simple := flag.Bool("simple", false, "Run gossiper in simple broadcast mode")
	antiEntropy := flag.Int("antiEntropy", 10, "Anti entropy timer value in seconds (default to 10sec)")
	startUIServer := flag.Bool("uisrv", false, "set to true to start the UI server on the UI port")
	rtimerFlag := flag.Int("rtimer", 0, "time between sending two route rumor messages")
	hw3ex2Flag := flag.Bool("hw3ex2", false, "HW3 EX2 Blockchain flag")
	hw3ex3Flag := flag.Bool("hw3ex3", false, "HW3 EX3 Blockchain flag")
	ackAfllFlag := flag.Bool("ackAll", false, "ACKALL Flag")
	peersNumber := flag.Uint64("N", 0, "Number of peers in the Network")
	stubbornTimeoutFlag := flag.Int("stubbornTimeout", 0, "stubbornTimeout")
	hoplimitFlag := flag.Int("hoplimit", -1, "TLC hoplimit")

	flag.Parse()
	antiEntropyTimer := *antiEntropy
	if antiEntropyTimer < 0 {
		antiEntropyTimer = defaultAntiEntropy
	}
	rtimer := *rtimerFlag
	if rtimer < 0 {
		rtimer = defaultRTimer
	}
	stubbornTimeout := *stubbornTimeoutFlag
	if stubbornTimeout <= 0 {
		stubbornTimeout = constant.DefaultStubbornTimeout
	}
	var hoplimit uint32
	if *hoplimitFlag < 0 {
		hoplimit = constant.DefaultHopLimit
	} else {
		hoplimit = uint32(*hoplimitFlag)
	}
	additionalFlags := &utils.AdditionalFlags{HW3ex2: *hw3ex2Flag, PeersNumber: *peersNumber, StubbornTimeout: stubbornTimeout, HW3ex3: *hw3ex3Flag, AckAll: *ackAfllFlag}
	gossiper := gossiper.NewGossiper(*gossipAddr, *name, *uiPort, *peersList, *simple, antiEntropyTimer, rtimer, hoplimit, additionalFlags)
	//starts UI server if flag is set
	if *startUIServer {
		uiServer := server.StartUIServer(*uiPort, gossiper)
		defer uiServer.Shutdown(nil)
	}

	gossiper.Start()
	defer gossiper.KillGossiper()
	gossiper.Active.Wait()

}
