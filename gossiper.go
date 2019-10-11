package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/dedis/protobuf"
	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/peers"
	"github.com/vquelque/Peerster/socket"
	"github.com/vquelque/Peerster/vector"
)

const channelSize = 4

// Gossiper structure
type Gossiper struct {
	name        string
	peers       *peers.Peers
	simple      bool
	peersSocket socket.Socket
	uiSocket    socket.Socket
	vectorClock *vector.Vector
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
	return &Gossiper{
		name:        name,
		peers:       peersSet,
		simple:      simple,
		peersSocket: peersSocket,
		uiSocket:    uiSocket,
		vectorClock: vectorClock,
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
	if gsp.vectorClock.NextMessage[sender] >= msg.ID {
		// forward message to another peer at random because current peer did not have message
		gsp.peers.Add(sender)
		gp := &GossipPacket{RumorMessage: msg}
		randPeer := gsp.peers.PickRandomPeer(sender)
		if randPeer != "" {
			gsp.send(gp, randPeer)
		} else {
			log.Print("No other peers to forward rumor message")
		}
	}
	if gsp.vectorClock.NextMessageForPeer(sender) == msg.ID {
		// we were waiting for this message
		gsp.vectorClock.IncrementMIDForPeer(sender)
	}

	// acknowledge the packet
	gsp.sendStatusPacket(sender)
}

////////////////////////////
// status packet //
////////////////////////////
func (gsp *Gossiper) sendStatusPacket(address string) {
	sp := gsp.vectorClock.StatusPacket()
	gp := &GossipPacket{nil, nil, sp}
	gsp.send(gp, address)
}

func (gsp *Gossiper) processStatusPacket(sp *vector.StatusPacket, sender string) {
	same, want, send := gsp.vectorClock.CompareWithStatusPacket(sp)
	if same && rand.Intn(2) == 0 {
		peer := gsp.peers.PickRandomPeer(sender)
		if peer != "" {

		}
	}

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
				fmt.Println(gp.RumorMessage.String())
				gsp.processRumorMessage(gp.RumorMessage, peerMsg.sender)
			case gp.StatusPacket != nil:
				gsp.processStatusPacket(gp.StatusPacket, sender)
			default:
				log.Print("Error : more than one message in GossipPacket ")
			}
		case cliMsg := <-clientMsgs:
			var msg *message.Message = &message.Message{}
			protobuf.Decode(cliMsg.data, msg)
			fmt.Printf(msg.String())
			gp := &GossipPacket{message.NewSimpleMessage(msg.Msg, gsp.name, gsp.peersSocket.Address()), nil, nil}
			if gsp.simple {
				//broadcast packet
				gsp.broadcastPacket(gp, gsp.peersSocket.Address())
			} else {
				//send to random peer
				randPeer := gsp.peers.PickRandomPeer("")
				if randPeer != "" {
					gsp.send(gp, randPeer)
				} else {
					log.Print("No other peers to forward rumor message")
				}
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
}

func (gsp *Gossiper) Start() {
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
	gossiper.Start()

	exit := make(chan string)
	for {
		select {
		case <-exit:
			os.Exit(0)
		}
	}
}
