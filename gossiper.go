package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	. "github.com/deckarep/golang-set" //for peers
	"github.com/dedis/protobuf"
	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/socket"
)

const channelSize = 4

// Gossiper structure
type Gossiper struct {
	name        string
	peers       Set
	simple      bool
	peersSocket socket.Socket
	uiSocket    socket.Socket
}

// GossipPacket is the only type of packet sent to other peers.
type GossipPacket struct {
	Simple *message.SimpleMessage
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
func newGossiper(address string, name string, uiPort int, peers []string, simple bool) *Gossiper {
	peersSocket := socket.NewUDPSocket(address)
	uiSocket := socket.NewUDPSocket(fmt.Sprintf("127.0.0.1:%d", uiPort))

	peersSet := NewSet()
	for _, peer := range peers {
		if peer != "" {
			peersSet.Add(peer)
		}
	}

	return &Gossiper{
		name:        name,
		peers:       peersSet,
		simple:      simple,
		peersSocket: peersSocket,
		uiSocket:    uiSocket,
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
// Addresses, Peers //
////////////////////////////
func formatPeersAddress(peers string) []string {
	return strings.Split(peers, ",")
}

////////////////////////////
// SimpleMessage //
////////////////////////////
func (gsp *Gossiper) newForwardedMessage(msg *message.SimpleMessage) *message.SimpleMessage {
	msg = message.NewSimpleMessage(msg.Contents, msg.OriginalName, gsp.peersSocket.Address())
	msg.RelayPeerAddr = gsp.peersSocket.Address()
	return msg
}

func (gsp *Gossiper) processSimpleMessage(msg *message.SimpleMessage, sender string) {
	gsp.peers.Add(msg.RelayPeerAddr)
	fwdMsg := gsp.newForwardedMessage(msg)
	packet := &GossipPacket{Simple: fwdMsg}
	if gsp.simple { //running in simple mode => broadcast
		gsp.broadcastPacket(packet, sender)
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
			fmt.Printf(gp.Simple.String())
			gsp.processSimpleMessage(gp.Simple, gp.Simple.RelayPeerAddr)
			peersString := gsp.peers.String()
			fmt.Println("PEERS : " + peersString[4:len(peersString)-1])
		case cliMsg := <-clientMsgs:
			var msg *message.Message = &message.Message{}
			protobuf.Decode(cliMsg.data, msg)
			fmt.Printf(msg.String())
			gp := &GossipPacket{message.NewSimpleMessage(msg.Msg, gsp.name, gsp.peersSocket.Address())}
			gsp.broadcastPacket(gp, gsp.peersSocket.Address())
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
	peers := flag.String("peers", "", "comma separated list of peers of the form ip:port")
	simple := flag.Bool("simple", false, "run gossiper in simple broadcast mode")
	flag.Parse()

	peersAddr := formatPeersAddress(*peers)
	gossiper := newGossiper(*gossipAddr, *name, *uiPort, peersAddr, *simple)
	gossiper.Start()

	exit := make(chan string)
	for {
		select {
		case <-exit:
			os.Exit(0)
		}
	}
}
