package gossiper

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/vector"
)

// Procecces incoming rumor message.
func (gsp *Gossiper) processRumorMessage(msg *message.RumorMessage, sender string) {
	//if sender is nil then it is a client message
	if sender != "" && msg.Origin != gsp.Name {
		fmt.Println(msg.PrintRumor(sender))
		gsp.Peers.Add(sender)
	}

	if gsp.VectorClock.NextMessageForPeer(msg.Origin) == msg.ID {
		// we were waiting for this message
		// increase mID for peer and store message
		gsp.VectorClock.IncrementMIDForPeer(msg.Origin)
		gsp.RumorStorage.Store(msg)
		//pick random peer and rumormonger
		randPeer := gsp.Peers.PickRandomPeer(sender)
		if randPeer != "" {
			gsp.rumormonger(msg, randPeer)
		} else {
			log.Print("No other peers to forward rumor message")
		}
		if sender != "" {
			gsp.Routing.UpdateRoute(msg, sender) //update routing table
		}
	}

	// acknowledge the packet if not sent by client
	if sender != "" {
		gsp.sendStatusPacket(sender)
		// println(gsp.routing.String())
		if msg.Text != "" && msg.Origin != gsp.Name {
			// Print DSDV only when not route runor
			fmt.Println(gsp.Routing.PrintUpdate(msg.Origin))
		}
	}
}

// Handle the rumormongering process and launch go routine that listens for ack or timeout.
func (gsp *Gossiper) rumormonger(rumor *message.RumorMessage, peerAddr string) {
	go gsp.listenForAck(rumor, peerAddr)
	gsp.sendRumorMessage(rumor, peerAddr)
	fmt.Printf("MONGERING with %s \n", peerAddr)
}

// Listen and handle ack or timeout.
func (gsp *Gossiper) listenForAck(rumor *message.RumorMessage, peerAddr string) {
	// register this channel inside the map of channels waiting for an ack (observer).
	id := peerAddr + rumor.Origin + string(rumor.ID)
	channel := gsp.WaitingForAck.Register(id)
	timer := time.NewTicker(ackTimeout * time.Second)
	defer func() {
		timer.Stop()
		gsp.WaitingForAck.Unregister(id)
	}()

	//keep running while channel open with for loop assignment
	for {
		select {
		case <-timer.C:
			gsp.coinFlip(rumor, peerAddr)
			return
		case ack := <-channel:
			if ack.Same {
				gsp.coinFlip(rumor, peerAddr)
			}
			return
		}
	}
}

// Send rumor to peerAddr.
func (gsp *Gossiper) sendRumorMessage(msg *message.RumorMessage, peerAddr string) {
	gp := GossipPacket{RumorMessage: msg}
	gsp.send(&gp, peerAddr)
}

// CoinFlip tosses a coin. If head, we rumormonger the rumor to a random peer. We exclude the sender
// from the randomly chosen peer.
func (gsp *Gossiper) coinFlip(rumor *message.RumorMessage, sender string) {
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
		rumorMsg := gsp.RumorStorage.Get(toSend[0].Identifier, toSend[0].NextID)
		gsp.rumormonger(rumorMsg, peerAddr)
	} else if len(toAsk) > 0 {
		// send status for triggering peer mongering
		gsp.sendStatusPacket(peerAddr)
	}
}
