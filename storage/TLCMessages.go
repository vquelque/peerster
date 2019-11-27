package storage

import (
	"sync"

	"github.com/vquelque/Peerster/message"
)

type TLCStorage struct {
	TLCMessages map[string][]*message.TLCMessage
	lock        *sync.RWMutex
}

func NewTLCMessageStorage() *TLCStorage {
	st := &TLCStorage{
		TLCMessages: make(map[string][]*message.TLCMessage)}
	return st
}

func (storage *TLCStorage) Store(tlcmsg *message.TLCMessage) {
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

func (storage *TLCStorage) Get(peer string, ID uint32) *message.TLCMessage {
	storage.lock.RLock()
	defer storage.lock.RUnlock()
	archive, found := storage.TLCMessages[peer]
	if !found || ID > uint32(len(archive)) || ID < 1 {
		return nil
	}
	return archive[ID-1]
}
