package storage

import (
	"sync"

	"github.com/vquelque/Peerster/message"
)

type Storage interface {
	NewStorage() *Storage
	Store()
	Get(peer string, ID string)
	GetAllForPeer(peer string)
	GetAll()
}

// Stores the previously received rumors.
type RumorStorage struct {
	// use a map to store the previous rumors. Key corresponds to peer origin.
	// value is a slice with all rumors for a given peer with IDs starting at 0.
	rumors      map[string][]message.RumorMessage
	rumor_order []string //append the name of the origin when the rumors arrive.
	// allows the client to retrieve rumors in order
	lock sync.RWMutex
}

// NewStorage creates a new storage to store rumors
func NewRumorStorage() *RumorStorage {
	st := &RumorStorage{
		rumors:      make(map[string][]message.RumorMessage),
		rumor_order: make([]string, 0)}
	return st
}

// StoreRumor stores a rumor message
func (storage *RumorStorage) Store(rumor *message.RumorMessage) {
	storage.lock.Lock()
	defer storage.lock.Unlock()
	origin := rumor.Origin
	archive := storage.rumors[origin]
	if rumor.ID == uint32(len(archive)+1) {
		// it is the good message
		archive = append(archive, *rumor)
		storage.rumors[origin] = archive
		storage.rumor_order = append(storage.rumor_order, origin)
	}
}

// GetRumor gets the rumor from storage
func (storage *RumorStorage) Get(peer string, ID uint32) *message.RumorMessage {
	storage.lock.RLock()
	defer storage.lock.RUnlock()
	archive, found := storage.rumors[peer]
	if !found || ID > uint32(len(archive)+1) {
		// we did not store this rumor previously => problem.
		return nil
	}
	return &archive[ID-1]
}
func (storage *RumorStorage) GetAllForPeer(peer string) []message.RumorMessage {
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
func (storage *RumorStorage) GetAll() []message.RumorMessage {
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
