package observer

import (
	"sync"

	"github.com/vquelque/Peerster/vector"
)

type Observer struct {
	waitingForAck map[string]chan *vector.StatusPacket
	lock          *sync.RWMutex
}

func Init() *Observer {
	lock := &sync.RWMutex{}
	obs := &Observer{make(map[string]chan *vector.StatusPacket), lock}
	return obs
}

func (obs *Observer) Register(sender string) chan *vector.StatusPacket {
	obs.lock.Lock()
	defer obs.lock.Unlock()
	ackChan := make(chan *vector.StatusPacket)
	obs.waitingForAck[sender] = ackChan
	return ackChan
}

func (obs *Observer) Unregister(sender string) {
	obs.lock.Lock()
	defer obs.lock.Unlock()
	ackChan, found := obs.waitingForAck[sender]
	if found && ackChan != nil {
		close(ackChan)
		obs.waitingForAck[sender] = nil
	}
}

func (obs *Observer) GetObserver(peer string) chan *vector.StatusPacket {
	obs.lock.RLock()
	defer obs.lock.RUnlock()
	ackChan, found := obs.waitingForAck[peer]
	if found {
		return ackChan
	}
	return nil
}
