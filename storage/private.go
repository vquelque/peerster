package storage

import (
	"sync"

	"github.com/vquelque/Peerster/message"
)

type PrivateStorage struct {
	messages map[string][]message.PrivateMessage
	lock     sync.RWMutex
}

func NewPrivateStorage() *PrivateStorage {
	st := &PrivateStorage{
		messages: make(map[string][]message.PrivateMessage),
		lock:     sync.RWMutex{},
	}
	return st
}

// Store private message to storage
func (storage *PrivateStorage) Store(message *message.PrivateMessage, peer string) {
	storage.lock.Lock()
	defer storage.lock.Unlock()
	archive := storage.messages[peer]
	archive = append(archive, *message)
	storage.messages[peer] = archive
}

// Get last private message for peer from storage
func (storage *PrivateStorage) Get(peer string) *message.PrivateMessage {
	storage.lock.RLock()
	defer storage.lock.RUnlock()
	archive, found := storage.messages[peer]
	if !found {
		// we don't have private messages for this peer
		return nil
	}
	return &archive[len(archive)-1]
}

// GetAllForPeer returns all the private messages for a peer
func (storage *PrivateStorage) GetAllForPeer(peer string) []message.PrivateMessage {
	storage.lock.RLock()
	defer storage.lock.RUnlock()
	archive, found := storage.messages[peer]
	m := make([]message.PrivateMessage, len(archive))
	if found {
		copy(m, archive)
	}
	return m
}
