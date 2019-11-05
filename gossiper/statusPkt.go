package gossiper

import (
	"fmt"
	"time"

	"github.com/vquelque/Peerster/observer"
	"github.com/vquelque/Peerster/vector"
)

// Sends a status packet to the given address.
func (gsp *Gossiper) sendStatusPacket(addr string) {
	sp := gsp.VectorClock.StatusPacket()
	gp := &GossipPacket{StatusPacket: &sp}
	gsp.send(gp, addr)
}

// Processes incoming status packets.
func (gsp *Gossiper) processStatusPacket(sp *vector.StatusPacket, sender string) {
	fmt.Print(sp.StringStatusWithSender(sender))
	gsp.Peers.Add(sender)
	fmt.Println(gsp.Peers.PrintPeers())
	//reset anti entropy timer
	gsp.ResetAntiEntropyTimer <- true

	same, toAsk, toSend := gsp.VectorClock.CompareWithStatusPacket(*sp)

	observerChan := gsp.WaitingForAck.GetObserver(sender)
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
	antiEntropyDuration := time.Duration(gsp.AntiEntropyTimer) * time.Second
	timer := time.NewTicker(antiEntropyDuration)
	defer timer.Stop()
	go func() {
		for {
			select {
			case <-timer.C:
				// timer elapsed : send status packet to randomly chosen peer
				// log.Println("No STATUS received : sending random STATUS")
				randPeer := gsp.Peers.PickRandomPeer("")
				if randPeer != "" {
					gsp.sendStatusPacket(randPeer)
				}
			case <-gsp.ResetAntiEntropyTimer:
				// timer reset : we received a status packet
				// log.Println("Received STATUS : Resetting anti entropy timer")
				timer = time.NewTicker(antiEntropyDuration)
			}
		}
	}()
}
