package main

import (
	"fmt"

	"github.com/vquelque/Peerster/message"
)

// ProcessClientMessage processes client messages
func (gsp *Gossiper) ProcessClientMessage(msg *message.Message) {
	fmt.Println(msg.String())
	if gsp.simple {
		gp := &GossipPacket{Simple: message.NewSimpleMessage(msg.Text, gsp.name, gsp.peersSocket.Address())}
		//broadcast packet
		gsp.broadcastPacket(gp, gsp.peersSocket.Address())
	} else {
		if msg.Destination != "" {
			//private message
			m := message.NewPrivateMessage(gsp.name, msg.Text, msg.Destination, defaultHopLimit)
			gsp.processPrivateMessage(m)
		} else {
			//rumor message
			mID := gsp.vectorClock.NextMessageForPeer(gsp.name)
			m := message.NewRumorMessage(gsp.name, mID, msg.Text)
			gsp.processRumorMessage(m, "")
		}
	}
}
