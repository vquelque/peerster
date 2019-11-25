package blockchain

import (
	"crypto/sha256"
	"sync"

	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/utils"
	"github.com/vquelque/Peerster/vector"
)

type TxPublish struct {
	Name        string
	Size        int64 // Size in bytes
	MetafileHah []byte
}

type BlockPublish struct {
	PrevHash    [32]byte //0 for ex3
	Transaction TxPublish
}

type Blocks struct {
	Blocks map[utils.SHA256]*BlockPublish
	Lock   sync.RWMutex
}

type PendingTLC struct {
	PendingTLC    map[utils.SHA256]*TLCMessage
	lastSeenTLCID map[string]uint32 //last tlc id for peer
	Lock          sync.RWMutex
}

type TLCMessage struct {
	Origin      string
	ID          uint32
	Confirmed   bool
	TxBlock     BlockPublish
	VectorClock *vector.StatusPacket
	Fitness     float32
}

type TLCAck *message.PrivateMessage

type Blockchain struct {
	Blocks     *Blocks
	PendingTLC *PendingTLC
	ID         uint32 //own rumor id use atomic function to increment/get !
}

func InitBlockchain() *Blockchain {
	blck := &Blocks{Blocks: make(map[utils.SHA256]*BlockPublish)}
	pTLC := &PendingTLC{PendingTLC: make(map[utils.SHA256]*TLCMessage), lastSeenTLCID: make(map[string]uint32)}
	return &Blockchain{Blocks: blck, PendingTLC: pTLC}
}

func NewBlockPublish(filename string, filesize int64, metahash utils.SHA256) *BlockPublish {
	tx := TxPublish{
		Name:        filename,
		Size:        filesize,
		MetafileHah: metahash[:],
	}
	return &BlockPublish{Transaction: tx}
}

func NewTLCMessage(origin string, id uint32, txBlock *BlockPublish, confirmed bool) *TLCMessage {
	return &TLCMessage{
		Origin:    origin,
		ID:        id,
		Confirmed: confirmed,
		TxBlock:   *txBlock,
	}
}

func NewTLCAck(origin string, destination string, id uint32, hoplimit uint32) TLCAck {
	return &message.PrivateMessage{
		Origin:      origin,
		Destination: destination,
		HopLimit:    hoplimit,
		ID:          id,
	}
}

func (b *Blockchain) AddPendingTLCIfValid(tlc *TLCMessage) bool {
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

func (b *Blockchain) Accept(tlc *TLCMessage) *BlockPublish {
	b.Blocks.Lock.Lock()
	defer b.Blocks.Lock.Unlock()
	b.PendingTLC.Lock.Lock()
	defer b.PendingTLC.Lock.Unlock()
	hash := tlc.TxBlock.Transaction.Hash()
	b.Blocks.Blocks[hash] = &tlc.TxBlock
	if _, pending := b.PendingTLC.PendingTLC[hash]; pending {
		delete(b.PendingTLC.PendingTLC, hash)
	}
	return b.Blocks.Blocks[hash]
}

func (tx *TxPublish) Hash() utils.SHA256 {
	hash := sha256.Sum256([]byte(tx.Name))
	return hash
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
