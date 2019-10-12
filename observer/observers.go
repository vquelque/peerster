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
	obs := &Observer{make(map[string]chan *vector.StatusPacket)}
	return obs
}

func (obs *Observer) WaitingForAck(sender string) {
	obs.lock.Lock()
	defer obs.lock.Unlock()
	ackChan := make(chan *vector.StatusPacket)
	obs.waitingForAck[sender] = ackChan
}

func (obs *Observer) Unregister(sender string) {
	obs.lock.Lock()
	defer obs.lock.Unlock()
	ackChan, found := obs.waitingForAck[sender]
	if found {
		close(ackChan)
		obs.waitingForAck[sender] = nil
	}
}
