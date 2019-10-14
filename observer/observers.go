package observer

import (
	"sync"

	"github.com/vquelque/Peerster/vector"
)

type Observer struct {
	waitingForAck map[string]chan ACK
	lock          *sync.RWMutex
}

// type for sending ack and result of comaprison with current vector clock to ack listener
type ACK struct {
	StatusPacket vector.StatusPacket
	Same         bool
}

// SendACKToChannel wrap the ack and the result of the comparison with the currrent vector clock
// and send them to the channel
func SendACKToChannel(channel *chan ACK, sp *vector.StatusPacket, same bool) {
	toChannel := ACK{StatusPacket: *sp, Same: same}
	*channel <- toChannel
}

func Init() *Observer {
	lock := &sync.RWMutex{}
	obs := &Observer{make(map[string]chan ACK), lock}
	return obs
}

func (obs *Observer) Register(sender string) chan ACK {
	obs.lock.Lock()
	defer obs.lock.Unlock()
	ackChan := make(chan ACK)
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

func (obs *Observer) GetObserver(peer string) chan ACK {
	obs.lock.RLock()
	defer obs.lock.RUnlock()
	ackChan, found := obs.waitingForAck[peer]
	if found {
		return ackChan
	}
	return nil
}
