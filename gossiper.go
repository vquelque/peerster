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
	"github.com/vquelque/Peerster/socket"
	"github.com/vquelque/Peerster/storage"
	"github.com/vquelque/Peerster/vector"
)

const channelSize = 4
const ackTimeout = 10 //in seconds

// Gossiper structure
type Gossiper struct {
	name          string
	peers         *peers.Peers
	simple        bool
	peersSocket   socket.Socket
	uiSocket      socket.Socket
	vectorClock   *vector.Vector
	rumors        *storage.Storage
	active        *sync.WaitGroup
	waitingForAck *observer.Observer
}

// GossipPacket is the only type of packet sent to other peers.
type GossipPacket struct {
	Simple       *message.SimpleMessage
	RumorMessage *message.RumorMessage
	StatusPacket *vector.StatusPacket
}

//encapsulate received messages from peers/client to put in the queue
type receivedPackets struct {
	data   []byte
	sender string
}

type packetToSend struct {
	data     []byte
	adddress string
}

// NewGossiper creates and returns a new gossiper running at given address, port with given name.
func newGossiper(address string, name string, uiPort int, peersList string, simple bool) *Gossiper {
	peersSocket := socket.NewUDPSocket(address)
	uiSocket := socket.NewUDPSocket(fmt.Sprintf("127.0.0.1:%d", uiPort))

	peersSet := peers.NewPeersSet(peersList)
	vectorClock := vector.NewVector()
	storage := storage.NewStorage()
	waitingForAck := observer.Init()
	return &Gossiper{
		name:          name,
		peers:         peersSet,
		simple:        simple,
		peersSocket:   peersSocket,
		uiSocket:      uiSocket,
		vectorClock:   vectorClock,
		rumors:        storage,
		waitingForAck: waitingForAck,
		active:        &sync.WaitGroup{},
	}
}

////////////////////////////
// Packets, GossipPacket //
////////////////////////////

//serialize with protobuf and send the gossipPacket to the provided UDP addr using the provided gossiper
func (gsp *Gossiper) send(gossipPacket *GossipPacket, addr string) {
	pkt, err := protobuf.Encode(gossipPacket)
	if err != nil {
		log.Print(err)
	}
	gsp.peersSocket.Send(pkt, addr)
}

func (gsp *Gossiper) broadcastPacket(pkt *GossipPacket, sender string) {
	for peer := range gsp.peers.Iterator().C {
		if peer != sender {
			gsp.send(pkt, peer.(string))
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
// RumorMessage //
////////////////////////////
func (gsp *Gossiper) processRumorMessage(msg *message.RumorMessage, sender string) {
	//if sender is nil then it is a client message
	if sender != "" {
		fmt.Println(msg.PrintRumor(sender))
		gsp.peers.Add(sender)
		// acknowledge the packet
		defer gsp.sendStatusPacket(sender)
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
}

// handle the rumormongering process and callback of eventual ack or timeout
func (gsp *Gossiper) rumormonger(rumor *message.RumorMessage, addr string) {
	go gsp.listenForAck(rumor, addr)
	gsp.sendRumorMessage(rumor, addr)
	fmt.Printf("MONGERING with %s \n", addr)
}

func (gsp *Gossiper) listenForAck(rumor *message.RumorMessage, addr string) {
	gsp.active.Add(1)
	defer gsp.active.Done()
	// register this channel inside the map of channels waiting for an ack (observer).
	channel := gsp.waitingForAck.Register(addr)
	defer gsp.waitingForAck.Unregister(addr)
	timer := time.NewTicker(ackTimeout * time.Second)
	defer timer.Stop()

	//keep running while channel open with for loop assignment
	for {
		select {
		case <-timer.C:
			gsp.coinFlip(rumor, addr)
			return
		case ack := <-channel:
			same, _, _ := gsp.vectorClock.CompareWithStatusPacket(ack)
			if same {
				gsp.coinFlip(rumor, addr)
				return
			}
			//update peers state
			gsp.vectorClock.UpdateVectorClock(ack)
			return
		}
	}
}

// send rumorMessage
func (gsp *Gossiper) sendRumorMessage(msg *message.RumorMessage, peerAddr string) {
	gp := GossipPacket{nil, msg, nil}
	gsp.send(&gp, peerAddr)
}

// coinFlip tosses a coin. If head, we rumormonger the rumor to a random peer. We exclude the sender
// from the randomly chosen peer.
func (gsp *Gossiper) coinFlip(rumor *message.RumorMessage, sender string) {
	if rand.Intn(2) == 0 {
		// exclude the sender of the rumor from the set where we pick our random peer to prevent a loop.
		peer := gsp.peers.PickRandomPeer(sender)
		if peer != "" {
			fmt.Printf("FLIPPED COIN sending rumor to %s\n", peer)
			gsp.rumormonger(rumor, peer)
		}
	}
}

//TODO MAYBE BUG HERE
// Check if we are in sync with peer. Else, send the missing messages to the peer.
func (gsp *Gossiper) synchronizeWithPeer(toAsk []vector.PeerStatus, toSend []vector.PeerStatus, peerAddr string) {
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
func (gsp *Gossiper) sendStatusPacket(addr string) {
	sp := gsp.vectorClock.StatusPacket()
	gp := &GossipPacket{nil, nil, &sp}
	gsp.send(gp, addr)
}

func (gsp *Gossiper) processStatusPacket(sp *vector.StatusPacket, sender string) {
	fmt.Print(sp.String())
	same, toAsk, toSend := gsp.vectorClock.CompareWithStatusPacket(*sp)

	if same {
		fmt.Printf("IN SYNC WITH %s \n", sender)
		return
	}
	// log.Print("START SYNC")
	observer := gsp.waitingForAck.GetObserver(sender)
	if observer != nil {
		// log.Print("OBSERVER FOUND")
		observer <- *sp
	}
	gsp.synchronizeWithPeer(toAsk, toSend, sender)

}

////////////////////////////
// Network //
////////////////////////////
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

func (gsp *Gossiper) processMessages(peerMsgs <-chan *receivedPackets, clientMsgs <-chan *receivedPackets) {
	// increment go routine counter to keep main program running.
	gsp.active.Add(1)
	// decrement goroutine counter at exit.
	defer gsp.active.Done()
	for {
		select {
		case peerMsg := <-peerMsgs:
			var gp *GossipPacket = &GossipPacket{}
			protobuf.Decode(peerMsg.data, gp)
			switch {
			case gp.Simple != nil:
				// received a simple message
				fmt.Println(gp.Simple.String())
				gsp.processSimpleMessage(gp.Simple)
				fmt.Println(gsp.peers.PrintPeers())
			case gp.RumorMessage != nil:
				// received a rumorMessage
				gsp.processRumorMessage(gp.RumorMessage, peerMsg.sender)
			case gp.StatusPacket != nil:
				gsp.processStatusPacket(gp.StatusPacket, peerMsg.sender)
			default:
				log.Print("Error : more than one message or 3 NIL in GossipPacket")
				log.Printf("%s,%s,%s", gp.Simple, gp.RumorMessage, gp.StatusPacket)
			}
		case cliMsg := <-clientMsgs:
			var msg *message.Message = &message.Message{}
			protobuf.Decode(cliMsg.data, msg)
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
	}
}

////////////////////////////
// Gossiper //
////////////////////////////

func (gsp *Gossiper) killGossiper() {
	gsp.peersSocket.Close()
	gsp.uiSocket.Close()
	gsp.active.Done()
}

func (gsp *Gossiper) start() {
	gsp.active.Add(1)
	peerChan := handleIncomingPackets(gsp.peersSocket)
	clientChan := handleIncomingPackets(gsp.uiSocket)
	go gsp.processMessages(peerChan, clientChan)
}

func main() {
	uiPort := flag.Int("UIPort", 8080, "Port for the UI client (default 8080)")
	gossipAddr := flag.String("gossipAddr", "", "ip:port for the gossiper")
	name := flag.String("name", "", "name of the gossiper")
	peersList := flag.String("peers", "", "comma separated list of peers of the form ip:port")
	simple := flag.Bool("simple", false, "run gossiper in simple broadcast mode")
	flag.Parse()

	gossiper := newGossiper(*gossipAddr, *name, *uiPort, *peersList, *simple)
	gossiper.start()
	gossiper.active.Wait()
}
