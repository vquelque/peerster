package observer

import (
	"sync"

	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/utils"
	"github.com/vquelque/Peerster/vector"
)

// Observer structure used for callback to routine
type Observer struct {
	waitingForAck map[string]chan ACK
	lock          *sync.RWMutex
}

// ACK used for sending result of comaprison with current vector clock to ack listener
type ACK struct {
	StatusPacket vector.StatusPacket
	Same         bool
}

type FileObserver struct {
	waitingForData map[utils.SHA256]chan *message.DataReply
	lock           *sync.RWMutex
}

// SendACKToChannel wrap the ack and the result of the comparison with the currrent vector clock
// and send them to the channel
func SendACKToChannel(channel chan ACK, sp *vector.StatusPacket, same bool) {
	toChannel := ACK{StatusPacket: *sp, Same: same}
	channel <- toChannel
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
		delete(obs.waitingForAck, sender)
	}
}

func (obs *Observer) GetObserver(peer string) chan ACK {
	obs.lock.Lock()
	defer obs.lock.Unlock()
	ackChan, found := obs.waitingForAck[peer]
	if found {
		return ackChan
	}
	return nil
}

func InitFileObserver() *FileObserver {
	lock := &sync.RWMutex{}
	obs := &FileObserver{make(map[utils.SHA256]chan *message.DataReply), lock}
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
