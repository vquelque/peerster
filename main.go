package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dedis/protobuf"
	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/observer"
	"github.com/vquelque/Peerster/peers"
	"github.com/vquelque/Peerster/routing"
	"github.com/vquelque/Peerster/socket"
	"github.com/vquelque/Peerster/storage"
	"github.com/vquelque/Peerster/vector"
)

const channelSize = 10
const defaultHopLimit = 10    //for private messages routing (in Hops)
const ackTimeout = 10         //in seconds
const defaultAntiEntropy = 10 //in seconds
const defaultRTimer = 0       //in seconds

// Gossiper main structure
type Gossiper struct {
	name                  string
	peers                 *peers.Peers
	simple                bool
	peersSocket           socket.Socket
	uiSocket              socket.Socket
	vectorClock           *vector.Vector        //current status of this peer.
	rumors                *storage.RumorStorage //store all previously received rumors.
	active                *sync.WaitGroup       //active go routines.
	waitingForAck         *observer.Observer    //registered go routines channels waiting for an ACK.
	antiEntropyTimer      int
	resetAntiEntropyTimer chan bool
	routing               *routing.Routing
	rtimer                int
}

// GossipPacket is the only type of packet sent to other peers.
type GossipPacket struct {
	Simple       *message.SimpleMessage
	RumorMessage *message.RumorMessage
	StatusPacket *vector.StatusPacket
	Private      *message.PrivateMessage
}

// Encapsulate received messages from peers/client to put in the queue
type receivedPackets struct {
	data   []byte
	sender string
}

// NewGossiper creates and returns a new gossiper running at given address, port with given name.
func newGossiper(address string, name string, uiPort int, peersList string, simple bool, antiEntropyTimer int, rtimer int) *Gossiper {
	peersSocket := socket.NewUDPSocket(address)
	uiSocket := socket.NewUDPSocket(fmt.Sprintf("127.0.0.1:%d", uiPort))

	peersSet := peers.NewPeersSet(peersList)
	vectorClock := vector.NewVector()
	storage := storage.NewRumorStorage()
	waitingForAck := observer.Init()
	resetAntiEntropyChan := make(chan (bool))
	routing := routing.NewRoutingTable()

	return &Gossiper{
		name:                  name,
		peers:                 peersSet,
		simple:                simple,
		peersSocket:           peersSocket,
		uiSocket:              uiSocket,
		vectorClock:           vectorClock,
		rumors:                storage,
		waitingForAck:         waitingForAck,
		active:                &sync.WaitGroup{},
		antiEntropyTimer:      antiEntropyTimer,
		resetAntiEntropyTimer: resetAntiEntropyChan,
		routing:               routing,
		rtimer:                rtimer,
	}
}

////////////////////////////
// Packets, GossipPacket //
////////////////////////////

// serialize with protobuf and send the gossipPacket to the provided UDP addr using the provided gossiper
func (gsp *Gossiper) send(gossipPacket *GossipPacket, addr string) {
	pkt, err := protobuf.Encode(gossipPacket)
	if err != nil {
		log.Print(err)
	}
	gsp.peersSocket.Send(pkt, addr)
}

func (gsp *Gossiper) broadcastPacket(pkt *GossipPacket, sender string) {
	for _, peer := range gsp.peers.GetAllPeers() {
		if peer != sender {
			gsp.send(pkt, peer)
		}
	}
}

////////////////////////////
// SimpleMessage //
////////////////////////////
func (gsp *Gossiper) newForwardedMessage(msg *message.SimpleMessage) *message.SimpleMessage {
	msg = message.NewSimpleMessage(msg.Contents, msg.OriginalName, gsp.peersSocket.Address())
	msg.RelayPeerAddr = gsp.peersSocket.Address()
	return msg
}

func (gsp *Gossiper) processSimpleMessage(msg *message.SimpleMessage) {
	gsp.peers.Add(msg.RelayPeerAddr)
	fwdMsg := gsp.newForwardedMessage(msg)
	packet := &GossipPacket{Simple: fwdMsg}
	if gsp.simple { //running in simple mode => broadcast
		gsp.broadcastPacket(packet, msg.RelayPeerAddr)
	}
}

////////////////////////////
// Routing //
////////////////////////////
func (gsp *Gossiper) startRoutingMessageHandler() {
	rTimerDuration := time.Duration(gsp.rtimer) * time.Second
	timer := time.NewTicker(rTimerDuration)
	go func() {
		for {
			select {
			case <-timer.C:
				// timer elapsed : send route rumor packet to randomly chosen peer
				randPeer := gsp.peers.PickRandomPeer("")
				if randPeer != "" {
					gsp.sendRouteRumor(randPeer)
				}
			}
		}
	}()
}

func (gsp *Gossiper) sendRouteRumor(peer string) {
	rID := gsp.vectorClock.NextMessageForPeer(gsp.name)
	r := message.NewRouteRumorMessage(gsp.name, rID)
	gsp.sendRumorMessage(r, peer)
}

////////////////////////////
// Network //
////////////////////////////
// Handles the incoming packets.
func handleIncomingPackets(socket socket.Socket) <-chan *receivedPackets {
	out := make(chan *receivedPackets, channelSize)
	go func() {
		for {
			data, sender := socket.Receive()
			receivedPackets := &receivedPackets{data: data, sender: sender}
			out <- receivedPackets
		}
	}()
	return out
}

// Processes the incoming messages.
func (gsp *Gossiper) processMessages(peerMsgs <-chan *receivedPackets, clientMsgs <-chan *receivedPackets) {
	for {
		select {
		case peerMsg := <-peerMsgs:
			var gp *GossipPacket = &GossipPacket{}
			err := protobuf.Decode(peerMsg.data, gp)
			if err != nil {
				log.Print(err)
			}
			switch {
			case gp.Simple != nil:
				// received a simple message
				fmt.Println(gp.Simple.String())
				gsp.processSimpleMessage(gp.Simple)
			case gp.RumorMessage != nil:
				// received a rumorMessage
				gsp.processRumorMessage(gp.RumorMessage, peerMsg.sender)
			case gp.StatusPacket != nil:
				gsp.processStatusPacket(gp.StatusPacket, peerMsg.sender)
			case gp.Private != nil:
				gsp.processPrivateMessage(gp.Private)
			default:
				log.Print("Error : more than one message or 3 NIL in GossipPacket")
			}
		case cliMsg := <-clientMsgs:
			msg := &message.Message{}
			err := protobuf.Decode(cliMsg.data, msg)
			if err != nil {
				log.Print(err)
			}
			gsp.ProcessClientMessage(msg)
		}
		fmt.Println(gsp.peers.PrintPeers())
	}
}

////////////////////////////
// Gossiper //
////////////////////////////
// Kills the gossiper
func (gsp *Gossiper) killGossiper() {
	gsp.peersSocket.Close()
	gsp.uiSocket.Close()
	gsp.active.Done()
	gsp = nil
}

// Starts the gossiper
func (gsp *Gossiper) start() {
	gsp.active.Add(1)
	peerChan := handleIncomingPackets(gsp.peersSocket)
	clientChan := handleIncomingPackets(gsp.uiSocket)
	go gsp.processMessages(peerChan, clientChan)
	if !gsp.simple {
		gsp.startAntiEntropyHandler()
	}
	if gsp.rtimer > 0 {
		gsp.startRoutingMessageHandler()
		//send initial routing message
		randPeer := gsp.peers.PickRandomPeer("")
		gsp.sendRouteRumor(randPeer)
	}
}

func main() {
	uiPort := flag.Int("UIPort", 8080, "Port for the UI client (default 8080)")
	gossipAddr := flag.String("gossipAddr", "", "ip:port for the gossiper")
	name := flag.String("name", "", "Name of the gossiper")
	peersList := flag.String("peers", "", "Comma separated list of peers of the form ip:port")
	simple := flag.Bool("simple", false, "Run gossiper in simple broadcast mode")
	antiEntropy := flag.Int("antiEntropy", 10, "Anti entropy timer value in seconds (default to 10sec)")
	startUIServer := flag.Bool("uisrv", false, "set to true to start the UI server on the UI port")
	rtimerFlag := flag.Int("rtimer", 0, "time between sending two route rumor messages")
	flag.Parse()

	antiEntropyTimer := *antiEntropy
	if antiEntropyTimer < 0 {
		antiEntropyTimer = defaultAntiEntropy
	}
	rtimer := *rtimerFlag
	if rtimer < 0 {
		rtimer = defaultRTimer
	}

	gossiper := newGossiper(*gossipAddr, *name, *uiPort, *peersList, *simple, antiEntropyTimer, rtimer)

	//starts UI server if flag is set
	if *startUIServer {
		uiServer := StartUIServer(*uiPort, gossiper)
		defer uiServer.Shutdown(nil)
	}

	gossiper.start()
	defer gossiper.killGossiper()
	gossiper.active.Wait()

}
