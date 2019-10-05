package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/dedis/protobuf"
	"github.com/vquelque/Peerster/message"
)

// Gossiper structure
type Gossiper struct {
	address *net.UDPAddr
	conn    *net.UDPConn
	name    string
	peers   []*net.UDPAddr
}

// GossipPacket is the only type of packet sent to other peers.
type GossipPacket struct {
	Simple *message.SimpleMessage
}

// NewGossiper creates and returns a new gossiper running at given address, port with given name.
func newGossiper(address, name string) *Gossiper {
	udpAddr, err := net.ResolveUDPAddr("udp4", address)
	if err != nil {
		fmt.Printf("Error while resolving the UDP address")
		os.Exit(0)
	}
	udpConn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		fmt.Printf("Error while listening for UDP packets")
		os.Exit(0)
	}
	return &Gossiper{
		address: udpAddr,
		conn:    udpConn,
		name:    name,
	}
}

//Serialize with protobuf and send the gossipPacket to the provided UDP addr
func send(gossipPacket GossipPacket, conn net.UDPConn, addr *net.UDPAddr) {
	pkt, err := protobuf.Encode(&gossipPacket)
	if err != nil {
		log.Print(err)
	}
	_, err = conn.WriteToUDP(pkt, addr)
	if err != nil {
		log.Print(err)
	}
}

func main() {
	uiPort := flag.Int("UIPort", 8080, "Port for the UI client (default 8080)")
	gossipAddr := flag.String("gossipAddr", "", "ip:port for the gossiper")
	name := flag.String("name", "", "name of the gossiper")
	peers := flag.String("peers", "", "comma separated list of peers of the form ip:port")
	simple := flag.String("simple", "", "run gossiper in simple broadcast mode")

	gossiper := newGossiper(*gossipAddr, *name)
}
