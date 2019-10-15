package storage

import (
	"sync"

	"github.com/vquelque/Peerster/message"
)

// Stores the previously received rumors.
type Storage struct {
	// use a map to store the previous rumors. Key corresponds to peer origin.
	// value is a slice with all rumors for a given peer with IDs starting at 0.
	rumors      map[string][]message.RumorMessage
	rumor_order []string //append the name of the origin when the rumors arrive.
	// allows the client to retrieve rumors in order
	lock sync.RWMutex
}

// NewStorage creates a new storage to store rumors
func NewStorage() *Storage {
	st := &Storage{
		rumors:      make(map[string][]message.RumorMessage),
		rumor_order: make([]string, 0)}
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
		storage.rumor_order = append(storage.rumor_order, origin)
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
func (storage *Storage) GetAllRumorsForPeer(peer string) []message.RumorMessage {
	storage.lock.RLock()
	defer storage.lock.RUnlock()
	rumors := make([]message.RumorMessage, 0)
	archive, found := storage.rumors[peer]
	if found {
		rumors = archive
	}
	return rumors
}

// GetAllRumors return all the rumors in the order they were added.
func (storage *Storage) GetAllRumors() []message.RumorMessage {
	storage.lock.RLock()
	defer storage.lock.RUnlock()
	rID := make(map[string]int, len(storage.rumor_order))
	rumors := make([]message.RumorMessage, 0)
	for _, sender := range storage.rumor_order {
		rumors = append(rumors, storage.rumors[sender][rID[sender]])
		rID[sender]++
	}
	return rumors
}
