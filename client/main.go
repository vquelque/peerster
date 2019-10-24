package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/dedis/protobuf"
	"github.com/vquelque/Peerster/message"
)

func main() {

	uiPort := flag.Int("UIPort", 8080, "Port for the UI client (default 8080)")
	msg := flag.String("msg", "", "message to be sent; if the -dest flag is present, this is a private message, otherwise it’s a rumor message")
	destination := flag.String("dest", "", "destination for the private message. can be omitted")

	flag.Parse()

	addr := fmt.Sprintf("127.0.0.1:%d", *uiPort) //localhost gossiper address
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	udpAddrCli, err := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
	udpConn, err := net.ListenUDP("udp4", udpAddrCli)
	if err != nil {
		log.Fatalln(err)
	}

	message := message.Message{Text: *msg}
	if *destination != "" {
		message.Destination = *destination
	}

	pkt, err := protobuf.Encode(&message)

	if err != nil {
		log.Fatalln(err)
	}

	_, err = udpConn.WriteToUDP(pkt, udpAddr)
	if err == nil {
		log.Print("Packet sent")
	}
}
