package blockchain

import (
	"fmt"
	"sync"

	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/utils"
)

type Blocks struct {
	Blocks map[utils.SHA256]*message.BlockPublish
	Lock   sync.RWMutex
}

type PendingTLC struct {
	PendingTLC map[utils.SHA256]*message.TLCMessage
	Lock       sync.RWMutex
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
}

func InitBlockchain() *Blockchain {
	blck := &Blocks{Blocks: make(map[utils.SHA256]*message.BlockPublish)}
	pTLC := &PendingTLC{PendingTLC: make(map[utils.SHA256]*message.TLCMessage)}
	pBlocks := &PendingBlocks{PendingBlocks: make([]*message.BlockPublish, 0), ConfirmedTLC: make(chan *message.TLCMessage)}
	tRV := &TLCRoundVector{TLCRoundForPeer: make(map[string]uint32, 0), allowedForRound: true}
	nextRound := make(chan bool)
	return &Blockchain{Blocks: blck, PendingTLC: pTLC, PendingBlocks: pBlocks, TLCRoundVector: tRV, NextRound: nextRound}
}

func NewBlockPublish(filename string, filesize int64, metahash utils.SHA256) *message.BlockPublish {
	tx := message.TxPublish{
		Name:        filename,
		Size:        filesize,
		MetafileHah: metahash[:],
	}
	return &message.BlockPublish{Transaction: tx}
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

func (b *Blockchain) Accept(tlc *message.TLCMessage) *message.BlockPublish {
	b.Blocks.Lock.Lock()
	defer b.Blocks.Lock.Unlock()
	b.PendingTLC.Lock.Lock()
	defer b.PendingTLC.Lock.Unlock()
	hash := tlc.TxBlock.Transaction.Hash()
	if _, pending := b.PendingTLC.PendingTLC[hash]; pending {
		//for mapping in rumor storages
		// tlc.Confirmed = true
		fmt.Printf("PENDING TRANSACTION FOUND FOR TLC WITH HASH %x. ACCEPTING BLOCK\n", hash)
		b.Blocks.Blocks[hash] = &tlc.TxBlock
		delete(b.PendingTLC.PendingTLC, hash)
	}
	return b.Blocks.Blocks[hash]
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

func (b *Blockchain) AdvanceRoundForPeer(peer string, reset bool) {
	b.TLCRoundVector.Lock.Lock()
	defer b.TLCRoundVector.Lock.Unlock()
	_, p := b.TLCRoundVector.TLCRoundForPeer[peer]
	if !p {
		b.TLCRoundVector.TLCRoundForPeer[peer] = 0
	}
	if reset {
		b.TLCRoundVector.allowedForRound = true
	}
	b.TLCRoundVector.TLCRoundForPeer[peer] = b.TLCRoundVector.TLCRoundForPeer[peer] + 1

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
