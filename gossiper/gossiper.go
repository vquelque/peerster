package gossiper

import (
	"fmt"
	"sync"
	"time"

	"github.com/dedis/protobuf"
	"github.com/vquelque/Peerster/blockchain"
	"github.com/vquelque/Peerster/constant"
	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/observer"
	"github.com/vquelque/Peerster/peers"
	"github.com/vquelque/Peerster/routing"
	"github.com/vquelque/Peerster/socket"
	"github.com/vquelque/Peerster/storage"
	"github.com/vquelque/Peerster/utils"
	"github.com/vquelque/Peerster/vector"
)

// Gossiper main structure
type Gossiper struct {
	Name                  string
	Peers                 *peers.Peers
	Simple                bool
	PeersSocket           socket.Socket
	UISocket              socket.Socket
	VectorClock           *vector.Vector        //current status of this peer.
	RumorStorage          *storage.RumorStorage //store all previously received rumors.
	PrivateStorage        *storage.PrivateStorage
	FileStorage           *storage.FileStorage
	Active                sync.WaitGroup         //Active go routines.
	WaitingForAck         *observer.Observer     //registered go routines channels waiting for an ACK.
	WaitingForData        *observer.FileObserver //registered routines waiting for file data
	WaitingForSearchReply *observer.SearchObserver
	AntiEntropyTimer      int
	ResetAntiEntropyTimer chan bool
	Routing               *routing.Routing
	Rtimer                int
	UIStorage             *storage.UIStorage
	PendingSearchRequest  *storage.PendingRequests
	SearchResults         *storage.SearchResults
	ToDownload            *storage.ToDownload
	Blockchain            *blockchain.Blockchain
	AdditionalFlags       *utils.AdditionalFlags
	TLCStorage            *storage.TLCStorage
	WaitingForTLCAck      *observer.TLCAckObserver
	HopLimit              uint32
}

// GossipPacket is the only type of packet sent to other peers.
type GossipPacket struct {
	Simple        *message.SimpleMessage
	RumorMessage  *message.RumorMessage
	StatusPacket  *vector.StatusPacket
	Private       *message.PrivateMessage
	DataRequest   *message.DataRequest
	DataReply     *message.DataReply
	SearchRequest *message.SearchRequest
	SearchReply   *message.SearchReply
	TLCMessage    *message.TLCMessage
	Ack           *message.TLCAck
}

// Encapsulate received messages from peers/client to put in the queue
type receivedPackets struct {
	data   []byte
	sender string
}

// NewGossiper creates and returns a new gossiper running at given address, port with given name.
func NewGossiper(address string, name string, uiPort int, peersList string, simple bool, antiEntropyTimer int, rtimer int, hoplimit uint32, additionalFlags *utils.AdditionalFlags) *Gossiper {
	peersSocket := socket.NewUDPSocket(address)
	uiSocket := socket.NewUDPSocket(fmt.Sprintf("127.0.0.1:%d", uiPort))
	peersSet := peers.NewPeersSet(peersList)
	vectorClock := vector.NewVector()
	rumorStorage := storage.NewRumorStorage()
	privateStorage := storage.NewPrivateStorage()
	fileStorage := storage.NewFileStorage()
	waitingForAck := observer.Init()
	waitingForData := observer.InitFileObserver()
	waitingForSearchReply := observer.InitSearchObserver()
	resetAntiEntropyChan := make(chan (bool))
	routing := routing.NewRoutingTable()
	uiStorage := storage.NewUIStorage()
	searchResults := storage.NewSearchResult()
	toDownload := storage.NewToDownload()
	pendingSearchRequest := storage.NewPendingRequests()
	blockchain := blockchain.InitBlockchain()
	tlcStorage := storage.NewTLCMessageStorage()
	waitingForTLCAck := observer.InitTLCAckObserver()

	return &Gossiper{
		Name:                  name,
		Peers:                 peersSet,
		Simple:                simple,
		PeersSocket:           peersSocket,
		UISocket:              uiSocket,
		VectorClock:           vectorClock,
		RumorStorage:          rumorStorage,
		PrivateStorage:        privateStorage,
		FileStorage:           fileStorage,
		WaitingForAck:         waitingForAck,
		WaitingForData:        waitingForData,
		WaitingForSearchReply: waitingForSearchReply,
		Active:                sync.WaitGroup{},
		AntiEntropyTimer:      antiEntropyTimer,
		ResetAntiEntropyTimer: resetAntiEntropyChan,
		Routing:               routing,
		Rtimer:                rtimer,
		UIStorage:             uiStorage,
		SearchResults:         searchResults,
		ToDownload:            toDownload,
		PendingSearchRequest:  pendingSearchRequest,
		Blockchain:            blockchain,
		AdditionalFlags:       additionalFlags,
		TLCStorage:            tlcStorage,
		WaitingForTLCAck:      waitingForTLCAck,
		HopLimit:              hoplimit,
	}
}

////////////////////////////
// Packets, GossipPacket //
////////////////////////////

// serialize with protobuf and send the gossipPacket to the provided UDP addr using the provided gossiper
func (gsp *Gossiper) send(gossipPacket *GossipPacket, addr string) {
	pkt, err := protobuf.Encode(gossipPacket)
	if err != nil {
		//log.Print(err)
	}
	gsp.PeersSocket.Send(pkt, addr)
}

func (gsp *Gossiper) broadcastPacket(pkt *GossipPacket, sender string) {
	for _, peer := range gsp.Peers.GetAllPeers() {
		if peer != sender {
			gsp.send(pkt, peer)
		}
	}
}

////////////////////////////
// SimpleMessage //
////////////////////////////
func (gsp *Gossiper) newForwardedMessage(msg *message.SimpleMessage) *message.SimpleMessage {
	msg = message.NewSimpleMessage(msg.Contents, msg.OriginalName, gsp.PeersSocket.Address())
	return msg
}

func (gsp *Gossiper) processSimpleMessage(msg *message.SimpleMessage) {
	fmt.Println(msg.String())
	fmt.Println(gsp.Peers.PrintPeers())
	fwdMsg := gsp.newForwardedMessage(msg)
	packet := &GossipPacket{Simple: fwdMsg}
	gsp.broadcastPacket(packet, msg.RelayPeerAddr)
}

////////////////////////////
// Routing //
////////////////////////////
func (gsp *Gossiper) startRoutingMessageHandler() {
	rTimerDuration := time.Duration(gsp.Rtimer) * time.Second
	timer := time.NewTicker(rTimerDuration)
	//send initial routing message to all neighbors
	for _, peer := range gsp.Peers.GetAllPeers() {
		gsp.sendRouteRumor(peer)
	}
	go func() {
		for {
			select {
			case <-timer.C:
				// timer elapsed : send route rumor packet to randomly chosen peer
				randPeer := gsp.Peers.PickRandomPeer("")
				if randPeer != "" {
					gsp.sendRouteRumor(randPeer)
				}
			}
		}
	}()
}

func (gsp *Gossiper) sendRouteRumor(peer string) {
	rID := gsp.VectorClock.NextMessageForPeer(gsp.Name)
	r := message.NewRouteRumorMessage(gsp.Name, rID)
	gsp.processRumorMessage(r, "")
}

////////////////////////////
// Network //
////////////////////////////
// Handles the incoming packets.
func handleIncomingPackets(socket socket.Socket) <-chan *receivedPackets {
	out := make(chan *receivedPackets, constant.ChannelSize)
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
			if peerMsg.sender != "" {
				gsp.Peers.Add(peerMsg.sender)
			}
			if err != nil {
				// log.Print(err)
			}
			switch {
			case gp.Simple != nil:
				// received a simple message
				go gsp.processSimpleMessage(gp.Simple)
			case gp.RumorMessage != nil:
				// received a rumorMessage
				go gsp.processRumorMessage(gp.RumorMessage, peerMsg.sender)
			case gp.StatusPacket != nil:
				go gsp.processStatusPacket(gp.StatusPacket, peerMsg.sender)
			case gp.Private != nil:
				go gsp.processPrivateMessage(gp.Private)
			case gp.DataRequest != nil:
				go gsp.processDataRequest(gp.DataRequest)
			case gp.DataReply != nil:
				go gsp.processDataReply(gp.DataReply)
			case gp.SearchRequest != nil:
				go gsp.processSearchRequest(gp.SearchRequest)
			case gp.SearchReply != nil:
				go gsp.processSearchReply(gp.SearchReply)
			case gp.TLCMessage != nil:
				go gsp.processTLCMessage(gp.TLCMessage, peerMsg.sender)
			case gp.Ack != nil:
				go gsp.processTLCAck(*gp.Ack)
			}
		case cliMsg := <-clientMsgs:
			msg := &message.Message{}
			err := protobuf.Decode(cliMsg.data, msg)
			if err != nil {
				//	log.Print(err)
			}
			go gsp.ProcessClientMessage(msg)
		}
	}
}

////////////////////////////
// Gossiper //
////////////////////////////
// Kills the gossiper
func (gsp *Gossiper) KillGossiper() {
	gsp.PeersSocket.Close()
	gsp.UISocket.Close()
	gsp.Active.Done()
	gsp = nil
}

// Starts the gossiper
func (gsp *Gossiper) Start() {
	gsp.Active.Add(1)
	peerChan := handleIncomingPackets(gsp.PeersSocket)
	clientChan := handleIncomingPackets(gsp.UISocket)
	go gsp.processMessages(peerChan, clientChan)
	if !gsp.Simple && gsp.AntiEntropyTimer > 0 {
		gsp.startAntiEntropyHandler()
	}
	if gsp.Rtimer > 0 {
		gsp.startRoutingMessageHandler()
	}
	if gsp.AdditionalFlags.HW3ex2 || gsp.AdditionalFlags.HW3ex3 {
		gsp.StartBlockPublishHandler()
	}
}
