package storage

import "github.com/vquelque/Peerster/blockchain"

import "sync"

type TLCStorage struct {
	TLCMessages map[string][]*blockchain.TLCMessage
	lock        *sync.RWMutex
}

func NewTLCMessageStorage() *TLCStorage {
	st := &TLCStorage{
		TLCMessages: make(map[string][]*blockchain.TLCMessage)}
	return st
}

func (storage *TLCStorage) Store(tlcmsg *blockchain.TLCMessage) {
	storage.lock.Lock()
	defer storage.lock.Unlock()
	origin := tlcmsg.Origin
	archive := storage.TLCMessages[origin]
	if tlcmsg.ID == uint32(len(archive)+1) {
		// it is the good message
		archive = append(archive, tlcmsg)
		storage.TLCMessages[origin] = archive
	}
}

func (storage *TLCStorage) Get(peer string, ID uint32) *blockchain.TLCMessage {
	storage.lock.RLock()
	defer storage.lock.RUnlock()
	archive, found := storage.TLCMessages[peer]
	if !found || ID > uint32(len(archive)) || ID < 1 {
		return nil
	}
	return archive[ID-1]
}
