package storage

import (
	"sync"

	"github.com/vquelque/Peerster/message"
)

// Stores the previously received rumors.
type Storage struct {
	// use a map to store the previous rumors. Key corresponds to peer origin.
	// value is a slice with all rumors for a given peer with IDs starting at 0.
	rumors map[string][]message.RumorMessage
	lock   sync.RWMutex
}

// NewStorage creates a new storage to store rumors
func NewStorage() *Storage {
	st := &Storage{rumors: make(map[string][]message.RumorMessage)}
	return st
}

// StoreRumor stores a rumor message
func (storage *Storage) StoreRumor(rumor *message.RumorMessage) {
	storage.lock.Lock()
	defer storage.lock.Unlock()
	origin := rumor.Origin
	archive, _ := storage.rumors[origin]
	if rumor.ID == uint32(len(archive)) {
		// it is the good message
		archive = append(archive, *rumor)
		storage.rumors[origin] = archive
	}
}

// GetRumor gets the rumor from storage
func (storage *Storage) GetRumor(peer string, rumorId uint32) *message.RumorMessage {
	storage.lock.RLock()
	defer storage.lock.RUnlock()
	archive, found := storage.rumors[peer]
	if !found || rumorId > uint32(len(archive)) {
		// we did not store this rumor previously => problem.
		return nil
	}
	return &archive[rumorId]
}
