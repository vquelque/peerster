package main

import (
	"fmt"

	"github.com/vquelque/Peerster/message"
)

func (gsp *Gossiper) processPrivateMessage(msg *message.PrivateMessage) {
	if msg.Destination != gsp.name {
		if msg.HopLimit == 0 {
			return
		}
		gsp.sendPrivateMessage(msg)
		return
	}
	// this private message is for us
	gsp.privateStorage.Store(msg, msg.Origin)
	fmt.Println(msg.String())
}

// sendPrivateMessage sends private Message to dest and decrements hop limit
func (gsp *Gossiper) sendPrivateMessage(msg *message.PrivateMessage) {
	gp := &GossipPacket{Private: msg}
	msg.HopLimit--
	nextHopAddr := gsp.routing.GetRoute(msg.Destination)
	// println("sending private message to " + msg.Destination + " via " + nextHopAddr)
	if nextHopAddr != "" {
		if msg.Origin == gsp.name {
			// we are the origin of this message --> store it to retrieve it in conversation
			gsp.privateStorage.Store(msg, msg.Destination)
		}
		gsp.send(gp, nextHopAddr)
	}
}
