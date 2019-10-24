package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
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
const ackTimeout = 10         //in seconds
const defaultAntiEntropy = 10 //in seconds

// Gossiper main structure
type Gossiper struct {
	name                  string
	peers                 *peers.Peers
	simple                bool
	peersSocket           socket.Socket
	uiSocket              socket.Socket
	vectorClock           *vector.Vector     //current status of this peer.
	rumors                *storage.Storage   //store all previously received rumors.
	active                *sync.WaitGroup    //active go routines.
	waitingForAck         *observer.Observer //registered go routines channels waiting for an ACK.
	antiEntropyTimer      int
	resetAntiEntropyTimer chan bool
	routing               *routing.RoutingTable
}

// GossipPacket is the only type of packet sent to other peers.
type GossipPacket struct {
	Simple       *message.SimpleMessage
	RumorMessage *message.RumorMessage
	StatusPacket *vector.StatusPacket
}

// Encapsulate received messages from peers/client to put in the queue
type receivedPackets struct {
	data   []byte
	sender string
}

// NewGossiper creates and returns a new gossiper running at given address, port with given name.
func newGossiper(address string, name string, uiPort int, peersList string, simple bool, antiEntropyTimer int) *Gossiper {
	peersSocket := socket.NewUDPSocket(address)
	uiSocket := socket.NewUDPSocket(fmt.Sprintf("127.0.0.1:%d", uiPort))

	peersSet := peers.NewPeersSet(peersList)
	vectorClock := vector.NewVector()
	storage := storage.NewStorage()
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
// ClientMessage //
////////////////////////////
func (gsp *Gossiper) ProcessClientMessage(msg *message.Message) {
	fmt.Println(msg.String())
	if gsp.simple {
		gp := &GossipPacket{message.NewSimpleMessage(msg.Msg, gsp.name, gsp.peersSocket.Address()), nil, nil}
		//broadcast packet
		gsp.broadcastPacket(gp, gsp.peersSocket.Address())
	} else {
		mID := gsp.vectorClock.NextMessageForPeer(gsp.name)
		rumorMsg := message.NewRumorMessage(gsp.name, mID, msg.Msg)
		//send to random peer
		gsp.processRumorMessage(rumorMsg, "")
	}
}

////////////////////////////
// RumorMessage //
////////////////////////////
// Procecces incoming rumor message.
func (gsp *Gossiper) processRumorMessage(msg *message.RumorMessage, sender string) {
	//if sender is nil then it is a client message
	if sender != "" {
		fmt.Println(msg.PrintRumor(sender))
		gsp.peers.Add(sender)
	}

	if gsp.vectorClock.NextMessageForPeer(msg.Origin) == msg.ID {
		// we were waiting for this message
		// increase mID for peer and store message
		gsp.vectorClock.IncrementMIDForPeer(msg.Origin)
		gsp.rumors.StoreRumor(msg)
		//pick random peer and rumormonger
		randPeer := gsp.peers.PickRandomPeer(sender)
		if randPeer != "" {
			gsp.rumormonger(msg, randPeer)
		} else {
			log.Print("No other peers to forward rumor message")
		}
	}

	// acknowledge the packet if not sent by client
	if sender != "" {
		gsp.sendStatusPacket(sender)
	}

	//update routing table
	gsp.routing.UpdateRoute(msg, sender)
}

// Handle the rumormongering process and launch go routine that listens for ack or timeout.
func (gsp *Gossiper) rumormonger(rumor *message.RumorMessage, peerAddr string) {
	go gsp.listenForAck(rumor, peerAddr)
	gsp.sendRumorMessage(rumor, peerAddr)
	fmt.Printf("MONGERING with %s \n", peerAddr)
}

// Listen and handle ack or timeout.
func (gsp *Gossiper) listenForAck(rumor *message.RumorMessage, peerAddr string) {
	// register this channel inside the map of channels waiting for an ack (observer).
	id := peerAddr + rumor.Origin + string(rumor.ID)
	channel := gsp.waitingForAck.Register(id)
	timer := time.NewTicker(ackTimeout * time.Second)
	defer func() {
		timer.Stop()
		gsp.waitingForAck.Unregister(id)
	}()

	//keep running while channel open with for loop assignment
	for {
		select {
		case <-timer.C:
			gsp.coinFlip(rumor, peerAddr)
			return
		case ack := <-channel:
			if ack.Same {
				gsp.coinFlip(rumor, peerAddr)
			}
			return
		}
	}
}

// Send rumor to peerAddr.
func (gsp *Gossiper) sendRumorMessage(msg *message.RumorMessage, peerAddr string) {
	gp := GossipPacket{nil, msg, nil}
	gsp.send(&gp, peerAddr)
}

// CoinFlip tosses a coin. If head, we rumormonger the rumor to a random peer. We exclude the sender
// from the randomly chosen peer.
func (gsp *Gossiper) coinFlip(rumor *message.RumorMessage, sender string) {
	head := rand.Int() % 2
	if head == 0 {
		// exclude the sender of the rumor from the set where we pick our random peer to prevent a loop.
		peer := gsp.peers.PickRandomPeer(sender)
		if peer != "" {
			fmt.Printf("FLIPPED COIN sending rumor to %s\n", peer)
			gsp.rumormonger(rumor, peer)
		}
	}
}

// Check if we are in sync with peer. Else, send the missing messages to the peer.
func (gsp *Gossiper) synchronizeWithPeer(same bool, toAsk []vector.PeerStatus, toSend []vector.PeerStatus, peerAddr string) {
	if same {
		fmt.Printf("IN SYNC WITH %s \n", peerAddr)
		return
	}
	if len(toSend) > 0 {
		// we have new messages to send to the peer : start mongering
		//get the rumor we need to send from storage
		rumorMsg := gsp.rumors.GetRumor(toSend[0].Identifier, toSend[0].NextID)
		gsp.rumormonger(rumorMsg, peerAddr)
	} else if len(toAsk) > 0 {
		// send status for triggering peer mongering
		gsp.sendStatusPacket(peerAddr)
	}
}

////////////////////////////
// status packet //
////////////////////////////
// Sends a status packet to the given address.
func (gsp *Gossiper) sendStatusPacket(addr string) {
	sp := gsp.vectorClock.StatusPacket()
	gp := &GossipPacket{nil, nil, &sp}
	gsp.send(gp, addr)
}

// Processes incoming status packets.
func (gsp *Gossiper) processStatusPacket(sp *vector.StatusPacket, sender string) {
	fmt.Print(sp.StringStatusWithSender(sender))

	//reset anti entropy timer
	gsp.resetAntiEntropyTimer <- true

	same, toAsk, toSend := gsp.vectorClock.CompareWithStatusPacket(*sp)

	observerChan := gsp.waitingForAck.GetObserver(sender)
	if observerChan != nil {
		// A registered routine was expecting a status packet.
		// Forward the result of the comparison to the routine to potentially
		// trigger the coin toss.
		// log.Print("OBSERVER FOUND")
		observer.SendACKToChannel(observerChan, sp, same)
	}
	// if no registered channel, it is an anti-entropy status packet.
	// in both cases synchronize with the peer
	gsp.synchronizeWithPeer(same, toAsk, toSend, sender)

}

// Handles the anti entropy timer
func (gsp *Gossiper) startAntiEntropyHandler() {
	antiEntropyDuration := time.Duration(gsp.antiEntropyTimer) * time.Second
	timer := time.NewTicker(antiEntropyDuration)
	go func() {
		for {
			select {
			case <-timer.C:
				// timer elapsed : send status packet to randomly chosen peer
				// log.Println("No STATUS received : sending random STATUS")
				randPeer := gsp.peers.PickRandomPeer("")
				gsp.sendStatusPacket(randPeer)
			case <-gsp.resetAntiEntropyTimer:
				// timer reset : we received a status packet
				// log.Println("Received STATUS : Resetting anti entropy timer")
				timer = time.NewTicker(antiEntropyDuration)
			}

		}
	}()
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
}

func main() {
	uiPort := flag.Int("UIPort", 8080, "Port for the UI client (default 8080)")
	gossipAddr := flag.String("gossipAddr", "", "ip:port for the gossiper")
	name := flag.String("name", "", "Name of the gossiper")
	peersList := flag.String("peers", "", "Comma separated list of peers of the form ip:port")
	simple := flag.Bool("simple", false, "Run gossiper in simple broadcast mode")
	antiEntropy := flag.Int("antiEntropy", 10, "Anti entropy timer value in seconds (default to 10sec)")
	startUIServer := flag.Bool("uisrv", false, "set to true to start the UI server on the UI port")
	flag.Parse()

	antiEntropyTimer := *antiEntropy
	if antiEntropyTimer < 0 {
		antiEntropyTimer = defaultAntiEntropy
	}

	gossiper := newGossiper(*gossipAddr, *name, *uiPort, *peersList, *simple, antiEntropyTimer)

	//starts UI server if flag is set
	if *startUIServer {
		uiServer := StartUIServer(*uiPort, gossiper)
		defer uiServer.Shutdown(nil)
	}

	gossiper.start()
	defer gossiper.killGossiper()
	gossiper.active.Wait()

}
