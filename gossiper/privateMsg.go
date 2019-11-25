package gossiper

import (
	"fmt"

	"github.com/vquelque/Peerster/message"
)

func (gsp *Gossiper) processPrivateMessage(msg *message.PrivateMessage) {
	if msg.Destination != gsp.Name {
		if msg.HopLimit == 0 {
			return
		}
		gsp.sendPrivateMessage(msg)
		return
	}
	// this private message is for us
	gsp.PrivateStorage.Store(msg, msg.Origin)
	gsp.UIStorage.StorePrivateMsgAsync(msg, msg.Origin)
	if msg.Text != "" {
		fmt.Println(msg.String())
	}
}

// sendPrivateMessage sends private Message to dest and decrements hop limit
func (gsp *Gossiper) sendPrivateMessage(msg *message.PrivateMessage) {
	msg.HopLimit = msg.HopLimit - 1
	gp := &GossipPacket{Private: msg}
	nextHopAddr := gsp.Routing.GetRoute(msg.Destination)
	// println("sending private message to " + msg.Destination + " via " + nextHopAddr)
	if nextHopAddr != "" {
		if msg.Origin == gsp.Name && msg.Text != "" {
			// we are the origin of this message --> store it to retrieve it in conversation
			// store only if txt is not empty otherwise it is just a TLCAck
			gsp.PrivateStorage.Store(msg, msg.Destination)
			gsp.UIStorage.StorePrivateMsgAsync(msg, msg.Destination)
		}
		if msg.HopLimit > 0 {
			gsp.send(gp, nextHopAddr)
		}
	}
}
