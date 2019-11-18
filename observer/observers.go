package observer

import (
	"fmt"
	"sync"

	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/utils"
	"github.com/vquelque/Peerster/vector"
)

// Observer structure used for callback to routine
type Observer struct {
	waitingForAck map[string]chan bool
	lock          sync.RWMutex
}

type FileObserver struct {
	waitingForData map[utils.SHA256]chan *message.DataReply
	lock           sync.RWMutex
}

type SearchObserver struct {
	waitingForReply map[*message.SearchRequest]chan bool
	lock            sync.RWMutex
}

func Init() *Observer {
	obs := &Observer{make(map[string]chan bool), sync.RWMutex{}}
	return obs
}

func (obs *Observer) Register(sender string) chan bool {
	obs.lock.Lock()
	defer obs.lock.Unlock()
	ackChan := make(chan bool)
	obs.waitingForAck[sender] = ackChan
	return ackChan
}

func (obs *Observer) Unregister(sender string) {
	obs.lock.Lock()
	defer obs.lock.Unlock()
	ackChan, found := obs.waitingForAck[sender]
	if found && ackChan != nil {
		close(ackChan)
		delete(obs.waitingForAck, sender)
	}
}

func (obs *Observer) GetObserver(sp *vector.StatusPacket, peer string) chan bool {
	obs.lock.RLock()
	defer obs.lock.RUnlock()
	for _, ps := range sp.Want {
		id := peer + fmt.Sprintf("%s : %d", ps.Identifier, ps.NextID-1)
		ackChan, found := obs.waitingForAck[id]
		if found {
			return ackChan
		}
	}
	return nil
}

func InitFileObserver() *FileObserver {
	obs := &FileObserver{make(map[utils.SHA256]chan *message.DataReply), sync.RWMutex{}}
	return obs
}

func (obs *FileObserver) RegisterFileObserver(caller utils.SHA256) chan *message.DataReply {
	obs.lock.Lock()
	defer obs.lock.Unlock()
	ch := make(chan *message.DataReply)
	obs.waitingForData[caller] = ch
	return ch
}

func (obs *FileObserver) UnregisterFileObserver(caller utils.SHA256) {
	obs.lock.Lock()
	defer obs.lock.Unlock()
	ackChan, found := obs.waitingForData[caller]
	if found && ackChan != nil {
		close(ackChan)
		delete(obs.waitingForData, caller)
	}
}

func (obs *FileObserver) SendDataToObserver(caller utils.SHA256, chunk *message.DataReply) error {
	obs.lock.RLock()
	defer obs.lock.RUnlock()
	ackChan, found := obs.waitingForData[caller]
	if found && ackChan != nil {
		ackChan <- chunk
		return nil
	}
	return fmt.Errorf("No observer found for this reply")
}

func InitSearchObserver() *SearchObserver {
	obs := &SearchObserver{make(map[*message.SearchRequest]chan bool), sync.RWMutex{}}
	return obs
}

func (obs *SearchObserver) RegisterSearchObserver(sr *message.SearchRequest) chan bool {
	obs.lock.Lock()
	defer obs.lock.Unlock()
	ch := make(chan bool)
	obs.waitingForReply[sr] = ch
	return ch
}

func (obs *SearchObserver) UnregisterSearchObserver(sr *message.SearchRequest) {
	obs.lock.Lock()
	defer obs.lock.Unlock()
	ackChan, found := obs.waitingForReply[sr]
	if found && ackChan != nil {
		close(ackChan)
		delete(obs.waitingForReply, sr)
	}
}

func (obs *SearchObserver) SendMatchToSearchObserver(keyword string) {
	obs.lock.RLock()
	defer obs.lock.RUnlock()
	for sr, m := range obs.waitingForReply {
		keywords := sr.Keywords
		for _, k := range keywords {
			if k == keyword {
				m <- true
			}
		}
	}
}
