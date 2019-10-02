package main

import (
	"flag"
	"fmt"
	"net"
	"os"
)

// Gossiper structure
type Gossiper struct {
	address *net.UDPAddr
	conn    *net.UDPConn
	name    string
	//	peers   []*net.UDPAddr
}

// NewGossiper creates and returns a new gossiper running at given address, port with given name.
func NewGossiper(address, name string) *Gossiper {
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

func main() {
	UIPortPtr := flag.Int("UIPort", 8080, "Port for the UI client (default 8080)")
	GossipAddrPtr := flag.String("gossipAddr", "", "ip:port for the gossiper")
	namePtr := flag.String("name", "", "name of the gossiper")
	peersPtr := flag.String("peers", "", "comma separated list of peers of the form ip:port")
	simplePtr := flag.String("simple", "", "run gossiper in simple broadcast mode")
}
