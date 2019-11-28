package blockchain

import (
	"sync"
	"sync/atomic"

	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/utils"
)

type Blocks struct {
	Blocks map[utils.SHA256]*message.BlockPublish
	Lock   sync.RWMutex
}

type PendingTLC struct {
	PendingTLC    map[utils.SHA256]*message.TLCMessage
	lastSeenTLCID map[string]uint32 //last tlc id for peer
	Lock          sync.RWMutex
}

type PendingBlocks struct {
	PendingBlocks chan *message.BlockPublish
	Lock          sync.RWMutex
}

type Blockchain struct {
	Blocks        *Blocks
	PendingTLC    *PendingTLC
	PendingBlocks *PendingBlocks
	my_time       uint32
}

func InitBlockchain() *Blockchain {
	blck := &Blocks{Blocks: make(map[utils.SHA256]*message.BlockPublish)}
	pTLC := &PendingTLC{PendingTLC: make(map[utils.SHA256]*message.TLCMessage), lastSeenTLCID: make(map[string]uint32)}
	pBlocks := &PendingBlocks{PendingBlocks: make(chan *message.BlockPublish)}
	return &Blockchain{Blocks: blck, PendingTLC: pTLC, PendingBlocks: pBlocks}
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
	b.Blocks.Blocks[hash] = &tlc.TxBlock
	if tlc, pending := b.PendingTLC.PendingTLC[hash]; pending {
		//for mapping in rumor storage
		tlc.Confirmed = true
		delete(b.PendingTLC.PendingTLC, hash)
	}
	return b.Blocks.Blocks[hash]
}

func (b *Blockchain) LastSeenTLCID(origin string) uint32 {
	b.PendingTLC.Lock.RLock()
	defer b.PendingTLC.Lock.RUnlock()
	return b.PendingTLC.lastSeenTLCID[origin]
}

func (b *Blockchain) SetLastSeenTLCID(origin string, id uint32) uint32 {
	b.PendingTLC.Lock.Lock()
	defer b.PendingTLC.Lock.Unlock()
	b.PendingTLC.lastSeenTLCID[origin] = id
	return b.PendingTLC.lastSeenTLCID[origin]
}

func (b *Blockchain) Mytime() uint32 {
	return atomic.LoadUint32(&b.my_time)
}

func (b *Blockchain) AdvanceToNextRound() uint32 {
	return atomic.AddUint32(&b.my_time, 1)
}

func (b *Blockchain) AddPendingBlock(bp *message.BlockPublish) {
	b.PendingBlocks.Lock.Lock()
	defer b.PendingBlocks.Lock.Unlock()
	b.PendingBlocks.PendingBlocks <- bp
}

func (b *Blockchain) HasPendingBlocks() bool {
	b.PendingBlocks.Lock.RLock()
	defer b.PendingBlocks.Lock.RUnlock()
	return len(b.PendingBlocks.PendingBlocks) > 0
}
