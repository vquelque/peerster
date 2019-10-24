package main

import (
	"fmt"
	"time"

	"github.com/vquelque/Peerster/observer"
	"github.com/vquelque/Peerster/vector"
)

// Sends a status packet to the given address.
func (gsp *Gossiper) sendStatusPacket(addr string) {
	sp := gsp.vectorClock.StatusPacket()
	gp := &GossipPacket{StatusPacket: &sp}
	gsp.send(gp, addr)
}

// Processes incoming status packets.
func (gsp *Gossiper) processStatusPacket(sp *vector.StatusPacket, sender string) {
	fmt.Print(sp.StringStatusWithSender(sender))
	gsp.peers.Add(sender)
	//reset anti entropy timer
	gsp.resetAntiEntropyTimer <- true

	same, toAsk, toSend := gsp.vectorClock.CompareWithStatusPacket(*sp)

	observerChan := gsp.waitingForAck.GetObserver(sender)
	if observerChan != nil {
		// A registered routine was expecting a status packet.
		// Forward the result of the comparison to the routine to potentially
		// trigger the coin toss.
		// log.Print("OBSERVER FOUND")
		observer.SendACKToChannel(observerChan, sp, same)
	}
	// if no registered channel, it is an anti-entropy status packet.
	// in both cases synchronize with the peer
	gsp.synchronizeWithPeer(same, toAsk, toSend, sender)

}

// Handles the anti entropy timer
func (gsp *Gossiper) startAntiEntropyHandler() {
	antiEntropyDuration := time.Duration(gsp.antiEntropyTimer) * time.Second
	timer := time.NewTicker(antiEntropyDuration)
	go func() {
		for {
			select {
			case <-timer.C:
				// timer elapsed : send status packet to randomly chosen peer
				// log.Println("No STATUS received : sending random STATUS")
				randPeer := gsp.peers.PickRandomPeer("")
				if randPeer != "" {
					gsp.sendStatusPacket(randPeer)
				}
			case <-gsp.resetAntiEntropyTimer:
				// timer reset : we received a status packet
				// log.Println("Received STATUS : Resetting anti entropy timer")
				timer = time.NewTicker(antiEntropyDuration)
			}

		}
	}()
}
