package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strings"

	. "github.com/deckarep/golang-set" //for peers
	"github.com/dedis/protobuf"
	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/utils"
)

// Gossiper structure
type Gossiper struct {
	socket Socket
	name   string
	peers  Set
	simple bool
}

// GossipPacket is the only type of packet sent to other peers.
type GossipPacket struct {
	Simple *message.SimpleMessage
}

// NewGossiper creates and returns a new gossiper running at given address, port with given name.
func newGossiper(address string, name string, uiPort int, peers *[]net.UDPAddr, simple bool) *Gossiper {
	udpAddr := utils.ToUDPAddr(address)
	uiUDPAddr := utils.ToUDPAddr(fmt.Sprintf("127.0.0.1:%d", uiPort))

	udpConn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		log.Fatal("error creating udp connection")
	}

	uiConn, err := net.ListenUDP("udp4", uiUDPAddr)
	if err != nil {
		log.Fatal("error creating ui udp connection")
	}

	peersSet := NewSet()
	for _, peer := range *peers {
		peersSet.Add(peer)
	}

	return &Gossiper{
		address: udpAddr,
		conn:    udpConn,
		uiConn:  uiConn,
		name:    name,
		peers:   peersSet,
		simple:  simple,
	}
}

////////////////////////////
// Packets, GossipPacket //
////////////////////////////

//serialize with protobuf and send the gossipPacket to the provided UDP addr using the provided gossiper
func (gsp *Gossiper) send(gossipPacket *GossipPacket, addr *net.UDPAddr) {
	pkt, err := protobuf.Encode(gossipPacket)
	if err != nil {
		log.Print(err)
	}
	_, err = gsp.conn.WriteToUDP(pkt, addr)
	if err != nil {
		log.Print(err)
	}
}

func (gsp *Gossiper) broadcastPacket(pkt *GossipPacket) {
	for peer := range gsp.peers.Iterator().C {
		gsp.send(pkt, peer.(*net.UDPAddr))
	}
}

////////////////////////////
// Addresses, Peers //
////////////////////////////
func formatPeersAddress(peers string) *[]net.UDPAddr {
	addr := strings.Split(peers, ",")
	return utils.MapToUDP(addr)
}

////////////////////////////
// SimpleMessage //
////////////////////////////
func (gsp *Gossiper) newForwardedMessage(msg *message.SimpleMessage) *message.SimpleMessage {
	msg = message.NewSimpleMessage(msg.Contents, msg.OriginalName)
	msg.RelayPeerAddr = gsp.address.String()
	return msg
}

func (gsp *Gossiper) processSimpleMessage(msg *message.SimpleMessage) {
	gsp.peers.Add(utils.ToUDPAddr(msg.RelayPeerAddr))
	fwdMsg := gsp.newForwardedMessage(msg)
	packet := &GossipPacket{Simple: fwdMsg}
	// Broadcast to everyone but sender
	gsp.broadcastPacket(packet)
}

////////////////////////////
// Network //
////////////////////////////
func (gsp *Gossiper) handleMessage() {
	for {

	}
}

////////////////////////////
// Gossiper //
////////////////////////////
func (gsp *Gossiper) killGossiper() {
	gsp.socket.Close()
}

func main() {
	uiPort := flag.Int("UIPort", 8080, "Port for the UI client (default 8080)")
	gossipAddr := flag.String("gossipAddr", "", "ip:port for the gossiper")
	name := flag.String("name", "", "name of the gossiper")
	peers := flag.String("peers", "", "comma separated list of peers of the form ip:port")
	simple := flag.Bool("simple", false, "run gossiper in simple broadcast mode")

	peersAddr := formatPeersAddress(*peers)
	gossiper := newGossiper(*gossipAddr, *name, *uiPort, peersAddr, *simple)
	defer gossiper.killGossiper()
}
