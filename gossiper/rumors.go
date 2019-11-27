package gossiper

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/vquelque/Peerster/constant"
	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/vector"
)

// Procecces incoming rumor message/TLC packet.
func (gsp *Gossiper) processRumorMessage(msg *message.RumorMessage, sender string) {
	next := gsp.VectorClock.NextMessageForPeer(msg.Origin)
	if sender != "" && msg.ID >= next && msg.Origin != gsp.Name {
		gsp.Routing.UpdateRoute(msg, sender) //update routing table
		if msg.Text != "" {
			fmt.Println(gsp.Routing.PrintUpdate(msg.Origin))
		}
	}
	rp := &message.RumorPacket{RumorMessage: msg}
	gsp.processRumorPacket(rp, sender)
}

func (gsp *Gossiper) processRumorPacket(pkt *message.RumorPacket, sender string) {
	//if sender is nil then it is a client message

	origin, id, rumor := pkt.GetDetails()
	//store rumor packet
	next := gsp.VectorClock.NextMessageForPeer(origin)

	if next == id {
		if sender != "" {
			fmt.Println(pkt.String(origin))
		}
		// we were waiting for this message
		// increase mID for peer and store message
		gsp.VectorClock.IncrementMIDForPeer(origin)
		gsp.RumorStorage.Store(pkt)
		//pick random peer and rumormonger
		randPeer := gsp.Peers.PickRandomPeer(sender)
		if rumor && pkt.RumorMessage.Text != "" {
			//not route rumor => append to UI
			gsp.UIStorage.AppendRumorAsync(pkt.RumorMessage)
		}
		if randPeer != "" {
			gsp.rumormonger(pkt, randPeer)
		}
	}

	if !rumor && id <= next {
		//TLC Packet
		fmt.Println(gsp.Blockchain.IsPending(pkt.TLCMessage))
		if gsp.Blockchain.IsPending(pkt.TLCMessage) && pkt.TLCMessage.Confirmed {
			fmt.Println(pkt.String(origin))
			randPeer := gsp.Peers.PickRandomPeer(sender)
			if randPeer != "" {
				gsp.rumormonger(pkt, randPeer)
			}
		}
	}

	// acknowledge the packet if not sent by client
	if sender != "" {
		gsp.sendStatusPacket(sender)
	}

}

// Handle the rumormongering process and launch go routine that listens for ack or timeout.
func (gsp *Gossiper) rumormonger(rumorPkt *message.RumorPacket, peerAddr string) {
	go gsp.listenForAck(rumorPkt, peerAddr)
	switch {
	case rumorPkt.RumorMessage != nil:
		gsp.sendRumorMessage(rumorPkt.RumorMessage, peerAddr)
	case rumorPkt.TLCMessage != nil:
		gsp.sendTLCMessage(rumorPkt.TLCMessage, peerAddr)
	}
	fmt.Printf("MONGERING with %s \n", peerAddr)
}

// Listen and handle ack or timeout.
func (gsp *Gossiper) listenForAck(pkt *message.RumorPacket, peerAddr string) {
	// register this channel inside the map of channels waiting for an ack (observer).
	origin, id, _ := pkt.GetDetails()
	cID := fmt.Sprintf("%s : %s : %d", peerAddr, origin, id)
	channel := gsp.WaitingForAck.Register(cID)
	timer := time.NewTicker(constant.AckTimeout * time.Second)
	defer func() {
		timer.Stop()
		gsp.WaitingForAck.Unregister(cID)
	}()

	//keep running while channel open with for loop assignment
	for {
		select {
		case <-timer.C:
			gsp.coinFlip(pkt, peerAddr)
			//	fmt.Printf("TIMEOUT \n")
			return
		case ack := <-channel:
			if ack {
				gsp.coinFlip(pkt, peerAddr)
			}
			//	fmt.Printf("GOT ACK \n")
			return
		}
	}
}

// Send rumor to peerAddr.
func (gsp *Gossiper) sendRumorMessage(msg *message.RumorMessage, peerAddr string) {
	gp := GossipPacket{RumorMessage: msg}
	gsp.send(&gp, peerAddr)
}

func (gsp *Gossiper) sendTLCMessage(msg *message.TLCMessage, peerAddr string) {
	gp := GossipPacket{TLCMessage: msg}
	gsp.send(&gp, peerAddr)
}

// CoinFlip tosses a coin. If head, we rumormonger the rumor to a random peer. We exclude the sender
// from the randomly chosen peer.
func (gsp *Gossiper) coinFlip(rumor *message.RumorPacket, sender string) {
	head := rand.Int() % 2
	if head == 0 {
		// exclude the sender of the rumor from the set where we pick our random peer to prevent a loop.
		peer := gsp.Peers.PickRandomPeer(sender)
		if peer != "" {
			fmt.Printf("FLIPPED COIN sending rumor to %s\n", peer)
			gsp.rumormonger(rumor, peer)
		}
	}
}

// Check if we are in sync with peer. Else, send the missing messages to the peer.
func (gsp *Gossiper) synchronizeWithPeer(same bool, toAsk []vector.PeerStatus, toSend []vector.PeerStatus, peerAddr string) {
	if same {
		fmt.Printf("IN SYNC WITH %s \n", peerAddr)
		return
	}
	if len(toSend) > 0 {
		// we have new messages to send to the peer : start mongering
		//get the rumor we need to send from storage
		rumor := gsp.RumorStorage.Get(toSend[0].Identifier, toSend[0].NextID)
		if rumor != nil {
			gsp.rumormonger(rumor, peerAddr)
		}
	} else if len(toAsk) > 0 {
		// send status for triggering peer mongering
		//fmt.Println(toAsk)
		gsp.sendStatusPacket(peerAddr)
	}
}
