package gossiper

import (
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/vquelque/Peerster/blockchain"
	"github.com/vquelque/Peerster/storage"
)

type Hw3ex2 struct {
	hw3ex2          bool
	PeersNumber     uint64
	StubbornTimeout int
}

func (gsp *Gossiper) PublishName(file *storage.File) {
	fmt.Printf("PUBLISHING NAME %s ON BLOCKCHAIN\n", file.Name)
	bp := blockchain.NewBlockPublish(file.Name, file.Size, file.MetafileHash)
	TLC := blockchain.NewTLCMessage(gsp.Name, atomic.AddUint32(&gsp.Blockchain.ID, 1), bp, false)
	validTx := gsp.Blockchain.AddPendingTLCIfValid(TLC)
	if !validTx {
		log.Printf("NON VALID BLOCK. NAME ALREADY PUBLISHED\n")
		return
	}
	channel := gsp.WaitingForTLCAck.RegisterTLCAckObserver(TLC)
	var timer *time.Ticker
	if gsp.Hw3ex2.StubbornTimeout > 0 {
		stubbornTimeoutDuration := time.Duration(gsp.Hw3ex2.StubbornTimeout) * time.Second
		timer = time.NewTicker(stubbornTimeoutDuration)
	}
	majority := gsp.Hw3ex2.PeersNumber / 2
	acknowledged := []string{gsp.Name}
	//TODO RUMORMONGER MESSAGE FOR NOW BROADCAST
	gp := &GossipPacket{TLCMessage: TLC}
	gsp.broadcastPacket(gp, "")
	defer func() {
		timer.Stop()
		gsp.WaitingForTLCAck.UnregisterTLCAckObservers(TLC)
	}()
	//keep running while channel open with for loop assignment
	for {
		select {
		case <-timer.C:
			//INCREMENT ID AND BROADCAST AGAIN
			//register new observer and remove old => ignore very late messages
			gsp.WaitingForTLCAck.UnregisterTLCAckObservers(TLC)
			TLC.ID = atomic.AddUint32(&gsp.Blockchain.ID, 1)
			channel = gsp.WaitingForTLCAck.RegisterTLCAckObserver(TLC)
			//reset ack slice
			acknowledged = []string{gsp.Name}
			gsp.broadcastTLC(TLC, "")
		case ack := <-channel:
			//received ack
			acknowledged = append(acknowledged, ack.Origin)
			if uint64(len(acknowledged)) > majority {
				tx := gsp.Blockchain.Accept(TLC)
				//TODO RUMORMONGER TLC WITH CONFIRMED = TRUE for now BROADCAST
				confirmedTLC := blockchain.NewTLCMessage(gsp.Name, TLC.ID, tx, true)
				gp := &GossipPacket{TLCMessage: confirmedTLC}
				gsp.broadcastPacket(gp, "")
				fmt.Printf("RE-BROADCAST ID %d WITNESSES %s\n", TLC.ID, strings.Join(acknowledged, ","))
				return
			}
		}
	}
}

func (gsp *Gossiper) processTLCMessage(tlcmsg *blockchain.TLCMessage, prevHop string) {
	fmt.Printf("UNCONFIRMED GOSSIP origin %s ID %d file name %s size %d metahash %v\n",
		tlcmsg.Origin, tlcmsg.ID, tlcmsg.TxBlock.Transaction.Name, tlcmsg.TxBlock.Transaction.Size, tlcmsg.TxBlock.Transaction.MetafileHah)
	if tlcmsg.Confirmed {
		gsp.Blockchain.Accept(tlcmsg)
		fmt.Printf("CONFIRMED GOSSIP origin %s ID %d file name %s size %d metahash %v\n",
			tlcmsg.Origin, tlcmsg.ID, tlcmsg.TxBlock.Transaction.Name, tlcmsg.TxBlock.Transaction.Size, tlcmsg.TxBlock.Transaction.MetafileHah)
		gsp.broadcastTLC(tlcmsg, prevHop)
		return
	}
	gsp.Blockchain.AddPendingTLCIfValid(tlcmsg)
	//send ACK to origin
	ack := blockchain.NewTLCAck(gsp.Name, tlcmsg.Origin, tlcmsg.ID, gsp.HopLimit)
	fmt.Printf("SENDING ACK origin %s ID %d", gsp.Name, tlcmsg.ID)
	gsp.sendTLACK(ack)
	//Broadcast tlc
	gsp.broadcastTLCIfNew(tlcmsg, prevHop)
}

func (gsp *Gossiper) processTLCAck(tlcack blockchain.TLCAck) {
	fmt.Printf("GOT TLC ACK FROM %s\n", tlcack.Origin)
	if tlcack.Destination != gsp.Name {
		gsp.sendTLACK(tlcack)
		return
	}
	gsp.WaitingForTLCAck.SendTLCToAckObserver(tlcack)
}

func (gsp *Gossiper) broadcastTLCIfNew(tlcmsg *blockchain.TLCMessage, prevHop string) {
	lastId := gsp.Blockchain.LastSeenTLCID(tlcmsg.Origin)
	if tlcmsg.ID > lastId {
		gsp.Blockchain.SetLastSeenTLCID(tlcmsg.Origin, tlcmsg.ID)
		//broadcast to other peers
		gsp.broadcastTLC(tlcmsg, prevHop)
	}
}

func (gsp *Gossiper) broadcastTLC(tlcmsg *blockchain.TLCMessage, prevHop string) {
	gp := &GossipPacket{TLCMessage: tlcmsg}
	gsp.broadcastPacket(gp, prevHop)
}

func (gsp *Gossiper) sendTLACK(ack blockchain.TLCAck) {
	ack.HopLimit = ack.HopLimit - 1
	gp := &GossipPacket{Ack: &ack}
	nextHopAddr := gsp.Routing.GetRoute(ack.Destination)
	if nextHopAddr != "" {
		if ack.HopLimit > 0 {
			gsp.send(gp, nextHopAddr)
		}
	}
}
