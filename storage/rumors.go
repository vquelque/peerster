package storage

import (
	"sync"

	"github.com/vquelque/Peerster/message"
)

type Storage struct {
	// use a map to store the previous rumors. Key corresponds to peer origin.
	rumors map[string][]message.RumorMessage
	lock   sync.RWMutex
}

func NewStorage() *Storage {
	st := &Storage{rumors: make(map[string][]message.RumorMessage)}
	return st
}

func (storage *Storage) StoreRumor(rumor *message.RumorMessage) {
	storage.lock.Lock()
	defer storage.lock.Unlock()
	origin := rumor.Origin
	archive, _ := storage.rumors[origin]
	if rumor.ID == uint32(len(archive)-1) {
		// it is the good message
		archive = append(archive, *rumor)
	}
}

func (storage *Storage) GetRumor(peer string, rumorId uint32) *message.RumorMessage {
	storage.lock.RLock()
	defer storage.lock.RUnlock()
	archive, found := storage.rumors[peer]
	if !found || rumorId > uint32(len(archive)-1) {
		// we did not store this rumor previously => problem.
		return nil
	}
	return &archive[rumorId]
}
