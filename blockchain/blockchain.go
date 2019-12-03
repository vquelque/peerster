package blockchain

import (
	"fmt"
	"sync"

	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/utils"
)

type Blocks struct {
	Blocks   map[utils.SHA256]*message.BlockPublish
	PrevHash utils.SHA256
	Lock     sync.RWMutex
}

type PendingTLC struct {
	PendingTLC         map[utils.SHA256]*message.TLCMessage
	PengindTLCForClock map[uint32]*message.TLCMessage
	Lock               sync.RWMutex
}

type PendingBlocks struct {
	PendingBlocks []*message.BlockPublish
	ConfirmedTLC  chan *message.TLCMessage
	Lock          sync.RWMutex
}

type TLCRoundVector struct {
	TLCRoundForPeer   map[string]uint32
	TLCProofsForRound []*message.TLCMessage
	allowedForRound   bool
	Lock              sync.RWMutex
	myTime            uint32
}

type Blockchain struct {
	Blocks         *Blocks
	PendingTLC     *PendingTLC
	PendingBlocks  *PendingBlocks
	TLCRoundVector *TLCRoundVector
	NextRound      chan bool
	identifier     string
}

func InitBlockchain(identifier string) *Blockchain {
	blck := &Blocks{Blocks: make(map[utils.SHA256]*message.BlockPublish), PrevHash: utils.SHA256Zeros()}
	pTLC := &PendingTLC{PendingTLC: make(map[utils.SHA256]*message.TLCMessage), PengindTLCForClock: make(map[uint32]*message.TLCMessage)}
	pBlocks := &PendingBlocks{PendingBlocks: make([]*message.BlockPublish, 0), ConfirmedTLC: make(chan *message.TLCMessage)}
	tRV := &TLCRoundVector{TLCRoundForPeer: make(map[string]uint32, 0), allowedForRound: true}
	nextRound := make(chan bool)
	return &Blockchain{Blocks: blck, PendingTLC: pTLC, PendingBlocks: pBlocks, TLCRoundVector: tRV, NextRound: nextRound, identifier: identifier}
}

func NewBlockPublish(filename string, filesize int64, metahash utils.SHA256, prevHash utils.SHA256) *message.BlockPublish {
	tx := message.TxPublish{
		Name:        filename,
		Size:        filesize,
		MetafileHah: metahash[:],
	}
	return &message.BlockPublish{Transaction: tx, PrevHash: prevHash}
}

func NewTLCAck(origin string, destination string, id uint32, hoplimit uint32) message.TLCAck {
	return &message.PrivateMessage{
		Origin:      origin,
		Destination: destination,
		HopLimit:    hoplimit,
		ID:          id,
	}
}

func (b *Blockchain) AddPendingTLCIfValid(tlc *message.TLCMessage) bool {
	b.Blocks.Lock.RLock()
	defer b.Blocks.Lock.RUnlock()
	b.PendingTLC.Lock.Lock()
	defer b.PendingTLC.Lock.Unlock()
	hash := tlc.TxBlock.Transaction.Hash()
	valid := b.isValid(hash)
	if !valid {
		return false
	}
	b.PendingTLC.PendingTLC[hash] = tlc
	return true
}

func (b *Blockchain) RemovePendingTLC(tlc *message.TLCMessage) {
	b.PendingTLC.Lock.Lock()
	defer b.PendingTLC.Lock.Unlock()
	hash := tlc.TxBlock.Transaction.Hash()
	delete(b.PendingTLC.PendingTLC, hash)
}

func (b *Blockchain) IsPending(tlc *message.TLCMessage) bool {
	b.PendingTLC.Lock.RLock()
	defer b.PendingTLC.Lock.RUnlock()
	hash := tlc.TxBlock.Transaction.Hash()
	_, found := b.PendingTLC.PendingTLC[hash]
	return found
}

func (b *Blockchain) isValid(hash utils.SHA256) bool {
	_, exists := b.Blocks.Blocks[hash]
	_, pending := b.PendingTLC.PendingTLC[hash]
	return !exists && !pending
}

func (b *Blockchain) IsValid(hash utils.SHA256) bool {
	b.Blocks.Lock.RLock()
	defer b.Blocks.Lock.RUnlock()
	b.PendingTLC.Lock.RLock()
	defer b.PendingTLC.Lock.RUnlock()
	_, exists := b.Blocks.Blocks[hash]
	_, pending := b.PendingTLC.PendingTLC[hash]
	return !exists && !pending
}

func (b *Blockchain) TryAcceptTLC(tlc *message.TLCMessage) {
	b.PendingTLC.Lock.Lock()
	defer b.PendingTLC.Lock.Unlock()
	id := b.IdForTLCVector(tlc.VectorClock)
	b.PendingTLC.PengindTLCForClock[id] = tlc
	// fmt.Printf("TRY ACCEPT THIS VEC : %v, OTHER VEC : %v\n", b.TLCRoundVector.TLCRoundForPeer, tlc.VectorClock)
	b.Accept(tlc)
}

func (b *Blockchain) Accept(tlc *message.TLCMessage) {
	b.Blocks.Lock.Lock()
	defer b.Blocks.Lock.Unlock()
	b.PendingTLC.Lock.Lock()
	defer b.PendingTLC.Lock.Unlock()
	hash := tlc.TxBlock.Transaction.Hash()
	fmt.Printf("PENDING : %s \n", b.PendingTLC.PendingTLC[hash])
	if _, pending := b.PendingTLC.PendingTLC[hash]; pending {
		fmt.Printf("ACCEPTING BLOCK WITH Hash : %x\n", hash)
		select {
		case b.PendingBlocks.ConfirmedTLC <- tlc:
		default:
		}
		b.Blocks.Blocks[hash] = &tlc.TxBlock
		b.Blocks.PrevHash = hash
		delete(b.PendingTLC.PendingTLC, hash)
	}
}

func (b *Blockchain) GetTime(peer string) uint32 {
	b.TLCRoundVector.Lock.RLock()
	defer b.TLCRoundVector.Lock.RUnlock()
	return b.TLCRoundVector.TLCRoundForPeer[peer]
}

func (b *Blockchain) AddPendingBlock(bp *message.BlockPublish) {
	b.PendingBlocks.Lock.Lock()
	defer b.PendingBlocks.Lock.Unlock()
	b.PendingBlocks.PendingBlocks = append(b.PendingBlocks.PendingBlocks, bp)
}

func (b *Blockchain) HasPendingBlocks() bool {
	b.PendingBlocks.Lock.RLock()
	defer b.PendingBlocks.Lock.RUnlock()
	return len(b.PendingBlocks.PendingBlocks) > 0
}

func (b *Blockchain) ShiftPendingBlock() *message.BlockPublish {
	b.PendingBlocks.Lock.RLock()
	defer b.PendingBlocks.Lock.RUnlock()
	pb := b.PendingBlocks.PendingBlocks[0]
	b.PendingBlocks.PendingBlocks = b.PendingBlocks.PendingBlocks[1:]
	return pb
}

func (b *Blockchain) AdvanceRoundForPeer(peer string) {
	b.TLCRoundVector.Lock.Lock()
	defer b.TLCRoundVector.Lock.Unlock()
	_, p := b.TLCRoundVector.TLCRoundForPeer[peer]
	if !p {
		b.TLCRoundVector.TLCRoundForPeer[peer] = 0
	}
	b.TLCRoundVector.TLCRoundForPeer[peer] = b.TLCRoundVector.TLCRoundForPeer[peer] + 1
}

func (b *Blockchain) ResetAllowedForRound() {
	b.TLCRoundVector.Lock.Lock()
	defer b.TLCRoundVector.Lock.Unlock()
	b.TLCRoundVector.allowedForRound = true
}

func (b *Blockchain) GetRoundForPeer(peer string) uint32 {
	b.TLCRoundVector.Lock.Lock()
	defer b.TLCRoundVector.Lock.Unlock()
	return b.TLCRoundVector.TLCRoundForPeer[peer]
}

func (b *Blockchain) CheckAllowedToPublish() bool {
	b.TLCRoundVector.Lock.RLock()
	defer b.TLCRoundVector.Lock.RUnlock()
	return b.TLCRoundVector.allowedForRound
}

func (b *Blockchain) Published() {
	b.TLCRoundVector.Lock.Lock()
	defer b.TLCRoundVector.Lock.Unlock()
	b.TLCRoundVector.allowedForRound = false
}

func (b *Blockchain) ResetRoundForPeers(currRound uint32) {
	b.TLCRoundVector.Lock.Lock()
	defer b.TLCRoundVector.Lock.Unlock()
	for p, r := range b.TLCRoundVector.TLCRoundForPeer {
		if r < currRound {
			b.TLCRoundVector.TLCRoundForPeer[p] = currRound
		}
	}
}

func (b *Blockchain) GetPreviousHash() utils.SHA256 {
	b.Blocks.Lock.RLock()
	defer b.Blocks.Lock.RUnlock()
	return b.Blocks.PrevHash
}

func (b *Blockchain) TLCRoundStatus() *message.StatusPacket {
	sp := &message.StatusPacket{}
	sp.Want = make([]message.PeerStatus, 0)
	b.TLCRoundVector.Lock.RLock()
	defer b.TLCRoundVector.Lock.RUnlock()
	for peer, rID := range b.TLCRoundVector.TLCRoundForPeer {
		ps := message.PeerStatus{peer, rID}
		sp.Want = append(sp.Want, ps)
	}
	return sp
}

func (b *Blockchain) IsFowardRumor(tlcStatus *message.StatusPacket) bool {
	b.TLCRoundVector.Lock.RLock()
	defer b.TLCRoundVector.Lock.RUnlock()
	if b.TLCRoundVector.TLCRoundForPeer[b.identifier] == 0 {
		return true
	}
	for _, round := range tlcStatus.Want {
		if round.Identifier == b.identifier {
			if round.NextID >= b.TLCRoundVector.TLCRoundForPeer[b.identifier] {
				return true
			}
		}
	}
	return false
}

func (b *Blockchain) IsInSync(tlcStatus *message.StatusPacket) bool {
	b.TLCRoundVector.Lock.RLock()
	defer b.TLCRoundVector.Lock.RUnlock()
	// we use a map to compute the difference between arrays
	m := make(map[string]bool)

	// first pass : compute difference with peers contained in other peer statusVector
	for _, status := range tlcStatus.Want {
		m[status.Identifier] = true
		next, found := b.TLCRoundVector.TLCRoundForPeer[status.Identifier]

		if found {
			if next > status.NextID {
				return false
			} else if next < status.NextID {
				return false
			}
		} else if status.NextID > 0 {
			return false
		}
	}
	// second pass : add the peers that are only in this status vector but not in the other one.
	for peer := range b.TLCRoundVector.TLCRoundForPeer {
		if !m[peer] {
			return false
		}
	}
	return true
}

func (b *Blockchain) IdForTLCVector(sp *message.StatusPacket) uint32 {
	var round uint32 = 0
	for _, p := range sp.Want {
		round += p.NextID
	}
	return round
}
