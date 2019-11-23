package storage

import (
	"fmt"
	"sync"

	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/utils"
)

type RumorUIStorage struct {
	//keep different storage to prevent blocking main map
	Rumors []*message.RumorMessage
	Lock   sync.RWMutex
}

type PrivateUIStorage struct {
	PrivateMsg map[string][]message.PrivateMessage
	Lock       sync.RWMutex
}

type DownloadableFiles struct {
	Downloadable map[string]string
	Lock         sync.RWMutex
}

type UIStorage struct {
	RumorUIStorage    *RumorUIStorage
	PrivateUIStorage  *PrivateUIStorage
	DownloadableFiles *DownloadableFiles
}

func NewUIStorage() *UIStorage {
	rumorStorage := &RumorUIStorage{Rumors: make([]*message.RumorMessage, 0), Lock: sync.RWMutex{}}
	privateStorage := &PrivateUIStorage{PrivateMsg: make(map[string][]message.PrivateMessage, 0), Lock: sync.RWMutex{}}
	downloadableFiles := &DownloadableFiles{Downloadable: make(map[string]string, 0), Lock: sync.RWMutex{}}
	return &UIStorage{RumorUIStorage: rumorStorage, PrivateUIStorage: privateStorage, DownloadableFiles: downloadableFiles}
}

func (sto *UIStorage) AppendRumorAsync(rumor *message.RumorMessage) {
	//append rumor if ui not reading => does not block main map
	go func() {
		sto.RumorUIStorage.Lock.Lock()
		defer sto.RumorUIStorage.Lock.Unlock()
		sto.RumorUIStorage.Rumors = append(sto.RumorUIStorage.Rumors, rumor)
	}()
}

func (sto *UIStorage) StorePrivateMsgAsync(msg *message.PrivateMessage, peer string) {
	go func() {
		sto.PrivateUIStorage.Lock.Lock()
		defer sto.PrivateUIStorage.Lock.Unlock()
		archive := sto.PrivateUIStorage.PrivateMsg[peer]
		archive = append(archive, *msg)
		sto.PrivateUIStorage.PrivateMsg[peer] = archive
	}()
}

// GetAllRumors return all the rumors in the order they were added.
func (sto *UIStorage) GetAllRumors() []*message.RumorMessage {
	sto.RumorUIStorage.Lock.RLock()
	defer sto.RumorUIStorage.Lock.RUnlock()
	return sto.RumorUIStorage.Rumors
}

func (sto *UIStorage) GetPrivateMessagesForPeer(peer string) []message.PrivateMessage {
	sto.PrivateUIStorage.Lock.RLock()
	defer sto.PrivateUIStorage.Lock.RUnlock()
	archive, _ := sto.PrivateUIStorage.PrivateMsg[peer]
	return archive
}

func (sto *UIStorage) AddDownloadableFile(filename string, metahash utils.SHA256) {
	sto.DownloadableFiles.Lock.Lock()
	defer sto.DownloadableFiles.Lock.Unlock()
	sto.DownloadableFiles.Downloadable[fmt.Sprint(metahash)] = filename
}

func (sto *UIStorage) RemoveDownloadableFile(metahash utils.SHA256) {
	sto.DownloadableFiles.Lock.Lock()
	defer sto.DownloadableFiles.Lock.Unlock()
	delete(sto.DownloadableFiles.Downloadable, fmt.Sprint(metahash))
}
