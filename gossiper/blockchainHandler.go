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
	bp := blockchain.NewBlockPublish(file.Name, file.Size, file.MetafileHash, gsp.Blockchain.GetPreviousHash())
	if gsp.AdditionalFlags.HW3ex2 == true {
		// ex2 -> don't care about TLC rounds
		gsp.HandleBlockPublish(bp, 0)
		return
	}
	fmt.Printf("ALLOWED : %v \n", gsp.Blockchain.CheckAllowedToPublish())
	if gsp.Blockchain.CheckAllowedToPublish() {
		gsp.HandleBlockPublish(bp, 0)
		return
	}
	gsp.Blockchain.AddPendingBlock(bp)
}

func (gsp *Gossiper) StartTLCRoundHandler() {
	go func() {
		TLCProofsForRound := make([]*message.TLCMessage, 0)
		for {
			select {
			case confirmedTLC := <-gsp.Blockchain.PendingBlocks.ConfirmedTLC:
				TLCProofsForRound = append(TLCProofsForRound, confirmedTLC)
				if uint64(len(TLCProofsForRound)) > gsp.AdditionalFlags.PeersNumber/2 {
					gsp.Blockchain.ResetAllowedForRound()
					gsp.Blockchain.AdvanceRoundForPeer(gsp.Name)
					fmt.Printf("ADVANCING TO round ​%d BASED ON CONFIRMED MESSAGES %s\n", gsp.Blockchain.GetRoundForPeer(gsp.Name), ProofsForRound(TLCProofsForRound))
					gsp.UIStorage.AppendProofsForRoundAsync(TLCProofsForRound, gsp.Blockchain.GetRoundForPeer(gsp.Name))
					TLCProofsForRound = make([]*message.TLCMessage, 0)
					select {
					//non blocking
					case gsp.Blockchain.NextRound <- true:
					default:
					}

				}
				if gsp.Blockchain.CheckAllowedToPublish() && gsp.Blockchain.HasPendingBlocks() {
					go gsp.HandleBlockPublish(gsp.Blockchain.ShiftPendingBlock(), 0)
				}
			}
		}
	}()
}

func (gsp *Gossiper) HandleBlockPublish(bp *message.BlockPublish, fitness float32) {
	nextID := gsp.VectorClock.NextMessageForPeer(gsp.Name)
	TLCStatusPkt := gsp.Blockchain.TLCRoundStatus()
	TLC := message.NewTLCMessage(gsp.Name, nextID, bp, -1, TLCStatusPkt, fitness)
	validTx := gsp.Blockchain.AddPendingTLCIfValid(TLC)
	if !validTx {
		log.Printf("NON VALID BLOCK. NAME ALREADY PUBLISHED\n")
		return
	}
	gsp.Blockchain.Published()
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
			gsp.mongerTLC(TLC, "")
			//TODO MONGER AGAIN
		case ack := <-channel:
			//received ack
			acknowledged = append(acknowledged, ack.Origin)
			if uint64(len(acknowledged)) > majority {
				// Broadcast confirmed TLC message
				nextID := gsp.VectorClock.NextMessageForPeer(gsp.Name)
				TLCStatusPkt := gsp.Blockchain.TLCRoundStatus()
				confirmedTLC := message.NewTLCMessage(gsp.Name, nextID, &TLC.TxBlock, int(TLC.ID), TLCStatusPkt, TLC.Fitness)
				fmt.Printf("RE-BROADCAST ID %d WITNESSES %s\n", TLC.ID, strings.Join(acknowledged, ","))
				gsp.processTLCMessage(confirmedTLC, "")
				return
			}
		case <-gsp.Blockchain.NextRound:
			fmt.Printf("NEXT ROUND. FORGETTING BLOCK")
			gsp.Blockchain.RemovePendingTLC(TLC)
			return
		}
	}
}

func (gsp *Gossiper) processTLCMessage(tlcmsg *message.TLCMessage, sender string) {
	rp := &message.RumorPacket{TLCMessage: tlcmsg}
	gsp.processRumorPacket(rp, sender)
	fmt.Printf("%v \n", tlcmsg.VectorClock.Want)
	if tlcmsg.Confirmed > 0 {
		//mothing more to do. we treated the message in the rumorMnonger handler
		return
	}
	valid := gsp.Blockchain.AddPendingTLCIfValid(tlcmsg)
	//send ACK to origin
	fmt.Printf("FORWARD : %v \n", gsp.Blockchain.IsFowardRumor(tlcmsg.VectorClock))
	fmt.Printf("THIS ROUND CLOCK %v \n", gsp.Blockchain.TLCRoundVector)
	fmt.Printf("OTER ROUND CLOCK %v \n", tlcmsg.VectorClock)
	if valid && tlcmsg.Origin != gsp.Name {
		if gsp.AdditionalFlags.HW3ex2 || gsp.Blockchain.IsFowardRumor(tlcmsg.VectorClock) || gsp.AdditionalFlags.AckAll {
			ack := blockchain.NewTLCAck(gsp.Name, tlcmsg.Origin, tlcmsg.ID, gsp.HopLimit)
			// fmt.Printf("SENDING ACK origin %s ID %d \n", gsp.Name, tlcmsg.ID)
			gsp.sendTLACK(ack)
		}
	}
}

func (gsp *Gossiper) mongerTLC(tlcmsg *message.TLCMessage, sender string) {
	log.Println("SENDING TLC")
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
	fmt.Printf("SENDING TLC ACK TO %s \n", ack.Destination)
	nextHopAddr := gsp.Routing.GetRoute(ack.Destination)
	if nextHopAddr != "" {
		if ack.HopLimit > 0 {
			gsp.send(gp, nextHopAddr)
		}
	}
}

func ProofsForRound(proofs []*message.TLCMessage) string {
	var str string
	for index, proof := range proofs {
		if index > 0 {
			str += ", "
		}
		str += fmt.Sprintf("origin%d %s ID%d %d", index+1, proof.Origin, index+1, proof.ID)
	}
	return str
}
