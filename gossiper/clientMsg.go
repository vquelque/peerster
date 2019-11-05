package gossiper

import (
	"fmt"

	"github.com/vquelque/Peerster/constant"
	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/utils"
)

// ProcessClientMessage processes client messages
func (gsp *Gossiper) ProcessClientMessage(msg *message.Message) {
	fmt.Println(msg.String())
	if gsp.Simple {
		gp := &GossipPacket{Simple: message.NewSimpleMessage(msg.Text, gsp.Name, gsp.PeersSocket.Address())}
		//broadcast packet
		gsp.broadcastPacket(gp, gsp.PeersSocket.Address())
	} else {
		if msg.Destination != "" && len(msg.Request) == 0 {
			//private message
			m := message.NewPrivateMessage(gsp.Name, msg.Text, msg.Destination, constant.DefaultHopLimit)
			go gsp.processPrivateMessage(m)
		} else if msg.File != "" && len(msg.Request) == 0 {
			go gsp.processFile(msg.File)
		} else if len(msg.Request) != 0 && msg.Destination != "" {
			h := utils.SliceToHash(msg.Request)
			gsp.startFileDownload(h, msg.Destination, msg.File)
		} else {
			//rumor message
			mID := gsp.VectorClock.NextMessageForPeer(gsp.Name)
			m := message.NewRumorMessage(gsp.Name, mID, msg.Text)
			go gsp.processRumorMessage(m, "")
		}
	}
}
