package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/dedis/protobuf"
	"github.com/vquelque/Peerster/gossiper"
	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/utils"
)

func main() {

	uiPort := flag.Int("UIPort", 8080, "Port for the UI client (default 8080)")
	text := flag.String("msg", "", "message to be sent; if the -dest flag is present, this is a private message, otherwise it’s a rumor message")
	destination := flag.String("dest", "", "destination for the private message. can be omitted")
	file := flag.String("file", "", "file to be indexed by the gossiper")

	flag.Parse()

	addr := fmt.Sprintf("127.0.0.1:%d", *uiPort) //localhost gossiper address
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	udpAddrCli, err := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
	udpConn, err := net.ListenUDP("udp4", udpAddrCli)
	if err != nil {
		log.Fatalln(err)
	}

	msg := message.Message{Text: *text}
	if *destination != "" {
		if *file != "" {
			log.Fatal("ERROR (Bad argument combination)")
		}
		msg.Destination = *destination
	}

	if *file != "" {
		if *text != "" {
			log.Fatal("ERROR (Bad argument combination)")
		}
		utils.CopyFile(*file, gossiper.FileTempDirectory)
		msg = message.Message{File: *file}
	}

	pkt, err := protobuf.Encode(&msg)

	if err != nil {
		log.Fatalln(err)
	}

	_, err = udpConn.WriteToUDP(pkt, udpAddr)
	if err == nil {
		fmt.Printf("CLIENT MESSAGE sent to %s", udpAddr.String())
	}
}
