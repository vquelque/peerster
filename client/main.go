package client

import (
	"flag"
	"fmt"
	"net"
	"os"
)

// SimpleMessage defines a simple message to send to other peers. Contains original sender's name
// peer's address in the form ip:port and text of the message
type SimpleMessage struct {
	OriginalName  string
	RelayPeerAddr string
	Contents      string
}

// GossipPacket is the only type of packet to be sent to other peers.
type GossipPacket struct {
	Simple *SimpleMessage
}

func main() {
	UIPortPtr := flag.Int("UIPort", 8080, "Port for the UI client (default 8080)")
	msgPtr := flag.String("msg", "", "message to be sent")

	flag.Parse()

	udpAddr, err := net.ResolveUDPAddr("udp4", 127.0.0.1:+*UIPortPtr)
	if err != nil {
		fmt.Printf("Error while resolving the UDP address")
		os.Exit(0)
	}
	packetToSend := GossipPacket{Simple: simplemessage}

}
