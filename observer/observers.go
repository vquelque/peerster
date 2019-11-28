package observer

import (
	"fmt"
	"strings"
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
	waitingForReply map[*message.SearchRequest]chan *message.SearchReply
	lock            sync.RWMutex
}

type TLCAckObserver struct {
	waitingForAck map[string]chan message.TLCAck
	lock          sync.RWMutex
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
		delete(obs.waitingForAck, sender)
	}
}

func (obs *Observer) GetObserver(sp *vector.StatusPacket, peer string) chan bool {
	obs.lock.RLock()
	defer obs.lock.RUnlock()
	for _, ps := range sp.Want {
		id := fmt.Sprintf("%s : %s : %d", peer, ps.Identifier, ps.NextID-1)
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
	obs := &SearchObserver{make(map[*message.SearchRequest]chan *message.SearchReply), sync.RWMutex{}}
	return obs
}

func (obs *SearchObserver) RegisterSearchObserver(sr *message.SearchRequest) chan *message.SearchReply {
	obs.lock.Lock()
	defer obs.lock.Unlock()
	ch := make(chan *message.SearchReply)
	obs.waitingForReply[sr] = ch
	return ch
}

func (obs *SearchObserver) UnregisterSearchObserver(sr *message.SearchRequest) {
	obs.lock.Lock()
	defer obs.lock.Unlock()
	delete(obs.waitingForReply, sr)
}

func (obs *SearchObserver) SendMatchToSearchObserver(r *message.SearchReply, keyword string) {
	obs.lock.RLock()
	defer obs.lock.RUnlock()
	for sr, m := range obs.waitingForReply {
		keywords := sr.Keywords
		for _, k := range keywords {
			// log.Printf("KEYWORD : %s. REPLY KEYWORD %s", keywords, keyword)
			if strings.Contains(keyword, k) {
				// log.Printf("GOT OBSERVER FOR SR with KEYWORD %s", keyword)
				m <- r
			}
		}
	}
}

func InitTLCAckObserver() *TLCAckObserver {
	return &TLCAckObserver{waitingForAck: make(map[string]chan message.TLCAck)}
}

func (obs *TLCAckObserver) RegisterTLCAckObserver(tlcmsg *message.TLCMessage) chan message.TLCAck {
	obs.lock.Lock()
	defer obs.lock.Unlock()
	ch := make(chan message.TLCAck)
	obs.waitingForAck[TLCAckObserverIdentifier(tlcmsg)] = ch
	return ch
}
func (obs *TLCAckObserver) AddObserverForID(tlcmsg *message.TLCMessage, ch chan message.TLCAck) {
	obs.lock.Lock()
	defer obs.lock.Unlock()
	obs.waitingForAck[TLCAckObserverIdentifier(tlcmsg)] = ch
}

func (obs *TLCAckObserver) UnregisterTLCAckObservers(tlcmsg *message.TLCMessage) {
	obs.lock.Lock()
	defer obs.lock.Unlock()
	id := TLCAckObserverIdentifier(tlcmsg)
	ackChan, found := obs.waitingForAck[id]
	if found && ackChan != nil {
		close(ackChan)
		delete(obs.waitingForAck, id)
	}
}

func (obs *TLCAckObserver) SendTLCToAckObserver(r message.TLCAck) {
	obs.lock.RLock()
	defer obs.lock.RUnlock()
	id := fmt.Sprintf("%s:%d", r.Destination, r.ID)
	ackChan, found := obs.waitingForAck[id]
	if found && ackChan != nil {
		ackChan <- r
	}
}

func TLCAckObserverIdentifier(tlcmsg *message.TLCMessage) string {
	return fmt.Sprintf("%s:%d", tlcmsg.Origin, tlcmsg.ID)
}

func tlcAckObserverIdentifierForID(tlcmsg *message.TLCMessage, id uint32) string {
	return fmt.Sprintf("%s:%d", tlcmsg.Origin, id)
}
