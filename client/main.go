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
	msg := flag.String("msg", "", "message to be sent")

	flag.Parse()

	addr := fmt.Sprintf("127.0.0.1:%d", *uiPort) //localhost gossiper address
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	udpConn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		log.Fatalln(err)
	}

	message := message.UIMessage{Msg: *msg}

	pkt, err := protobuf.Encode(&message)

	if err != nil {
		log.Fatalln(err)
	}

	udpConn.WriteToUDP(pkt, udpAddr)
}
