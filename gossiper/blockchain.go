package gossiper

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/vquelque/Peerster/blockchain"
	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/storage"
)

func (gsp *Gossiper) PublishName(file *storage.File) {
	fmt.Printf("PUBLISHING NAME %s ON BLOCKCHAIN\n", file.Name)
	bp := blockchain.NewBlockPublish(file.Name, file.Size, file.MetafileHash)
	gsp.Blockchain.AddPendingBlock(bp)
}

func (gsp *Gossiper) StartBlockPublishHandler() {
	go func() {
		for {
			select {
			case bp := <-gsp.Blockchain.PendingBlocks.PendingBlocks:
				gsp.HandleBlockPublish(bp)
			}
		}
	}()
}

func (gsp *Gossiper) HandleBlockPublish(bp *message.BlockPublish) {
	nextID := gsp.VectorClock.NextMessageForPeer(gsp.Name)
	TLC := message.NewTLCMessage(gsp.Name, nextID, bp, false)
	validTx := gsp.Blockchain.AddPendingTLCIfValid(TLC)
	if !validTx {
		log.Printf("NON VALID BLOCK. NAME ALREADY PUBLISHED\n")
		return
	}
	channel := gsp.WaitingForTLCAck.RegisterTLCAckObserver(TLC)
	var timer *time.Ticker
	if gsp.AdditionalFlags.StubbornTimeout > 0 {
		stubbornTimeoutDuration := time.Duration(gsp.AdditionalFlags.StubbornTimeout) * time.Second
		timer = time.NewTicker(stubbornTimeoutDuration)
	}
	majority := gsp.AdditionalFlags.PeersNumber / 2
	acknowledged := []string{gsp.Name}
	gsp.mongerTLC(TLC, "")
	defer func() {
		timer.Stop()
		gsp.WaitingForTLCAck.UnregisterTLCAckObservers(TLC)
	}()
	//keep running while channel open with for loop assignment
	for {
		select {
		case <-timer.C:
			//RUMORMONGER AGAIN
			fmt.Printf("MONGERING AGAIN TLC. STUBBORDN TIMEOUT EXCEEDED. \n")
			// gsp.WaitingForTLCAck.UnregisterTLCAckObservers(TLC)
			// nextID := gsp.VectorClock.NextTLCForPeer(gsp.Name)
			// acknowledged = []string{gsp.Name} //reset ack for this TLC
			// TLC = message.NewTLCMessage(gsp.Name, nextID, bp, false)
			// channel = gsp.WaitingForTLCAck.RegisterTLCAckObserver(TLC)
			gsp.mongerTLC(TLC, "")
		case ack := <-channel:
			//received ack
			acknowledged = append(acknowledged, ack.Origin)
			if uint64(len(acknowledged)) > majority {
				//TODO RUMORMONGER TLC WITH CONFIRMED = TRUE for now BROADCAST
				confirmedTLC := message.NewTLCMessage(gsp.Name, TLC.ID, &TLC.TxBlock, true)
				fmt.Printf("RE-BROADCAST ID %d WITNESSES %s\n", TLC.ID, strings.Join(acknowledged, ","))
				gsp.processTLCMessage(confirmedTLC, "")
				gsp.Blockchain.AdvanceToNextRound()
				return
			}
		}
	}
}

func (gsp *Gossiper) processTLCMessage(tlcmsg *message.TLCMessage, sender string) {
	rp := &message.RumorPacket{TLCMessage: tlcmsg}
	gsp.processRumorPacket(rp, sender)
	if tlcmsg.Confirmed {
		gsp.Blockchain.Accept(tlcmsg)
		return
	}
	valid := gsp.Blockchain.AddPendingTLCIfValid(tlcmsg)
	//send ACK to origin
	if valid && tlcmsg.Origin != gsp.Name {
		ack := blockchain.NewTLCAck(gsp.Name, tlcmsg.Origin, tlcmsg.ID, gsp.HopLimit)
		fmt.Printf("SENDING ACK origin %s ID %d \n", gsp.Name, tlcmsg.ID)
		gsp.sendTLACK(ack)
	}
}

func (gsp *Gossiper) mongerTLC(tlcmsg *message.TLCMessage, sender string) {
	gp := &message.RumorPacket{TLCMessage: tlcmsg}
	gsp.processRumorPacket(gp, sender)
}

func (gsp *Gossiper) processTLCAck(tlcack message.TLCAck) {
	if tlcack.Destination != gsp.Name {
		gsp.sendTLACK(tlcack)
		return
	}
	gsp.WaitingForTLCAck.SendTLCToAckObserver(tlcack)
}

func (gsp *Gossiper) sendTLACK(ack message.TLCAck) {
	ack.HopLimit = ack.HopLimit - 1
	gp := &GossipPacket{Ack: &ack}
	nextHopAddr := gsp.Routing.GetRoute(ack.Destination)
	if nextHopAddr != "" {
		if ack.HopLimit > 0 {
			gsp.send(gp, nextHopAddr)
		}
	}
}
