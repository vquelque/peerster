package observer

import (
	"fmt"
	"sync"

	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/utils"
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

// SendACKToChannel wrap the ack and the result of the comparison with the currrent vector clock
// and send them to the channel
func SendACKToChannel(channel chan bool, same bool) {
	channel <- same
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

func (obs *Observer) GetObserver(peer string) chan bool {
	obs.lock.Lock()
	defer obs.lock.Unlock()
	ackChan, found := obs.waitingForAck[peer]
	if found {
		return ackChan
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
