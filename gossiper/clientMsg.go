package gossiper

import (
	"fmt"

	"github.com/vquelque/Peerster/message"
)

// ProcessClientMessage processes client messages
func (gsp *Gossiper) ProcessClientMessage(msg *message.Message) {
	fmt.Println(msg.String())
	if gsp.Simple {
		gp := &GossipPacket{Simple: message.NewSimpleMessage(msg.Text, gsp.Name, gsp.PeersSocket.Address())}
		//broadcast packet
		gsp.broadcastPacket(gp, gsp.PeersSocket.Address())
	} else {
		if msg.Destination != "" {
			//private message
			m := message.NewPrivateMessage(gsp.Name, msg.Text, msg.Destination, defaultHopLimit)
			gsp.processPrivateMessage(m)
		} else if msg.File != "" {
			go gsp.processFile(msg.File)
		} else {
			//rumor message
			mID := gsp.VectorClock.NextMessageForPeer(gsp.Name)
			m := message.NewRumorMessage(gsp.Name, mID, msg.Text)
			gsp.processRumorMessage(m, "")
		}
	}
}
