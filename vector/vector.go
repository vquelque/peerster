package vector

import (
	"sync"

	"github.com/vquelque/Peerster/message"
)

// Vector clock.
// We use ID for messages starting at 0
type Vector struct {
	nextMessage map[string]uint32
	nextRumor   map[string]uint32
	nextTLC     map[string]uint32
	peersLock   sync.RWMutex
}

// NewVector returns a new empty vector clocl
func NewVector() *Vector {
	vec := &Vector{nextMessage: make(map[string]uint32), nextRumor: make(map[string]uint32), nextTLC: make(map[string]uint32)}
	return vec
}

// NextMessageForPeer returns next message id for given peer name.
func (vec *Vector) NextMessageForPeer(peer string) uint32 {
	vec.peersLock.Lock()
	defer vec.peersLock.Unlock()
	_, found := vec.nextMessage[peer]
	if !found {
		vec.nextMessage[peer] = 1
	}
	return vec.nextMessage[peer]
}

func (vec *Vector) nextRumorForPeer(peer string) uint32 {
	vec.peersLock.Lock()
	defer vec.peersLock.Unlock()
	_, found := vec.nextRumor[peer]
	if !found {
		vec.nextRumor[peer] = 1
	}
	return vec.nextRumor[peer]
}

func (vec *Vector) nextTLCForPeer(peer string) uint32 {
	vec.peersLock.Lock()
	defer vec.peersLock.Unlock()
	_, found := vec.nextTLC[peer]
	if !found {
		vec.nextTLC[peer] = 1
	}
	return vec.nextTLC[peer]
}

// Increments message ID for the given peer. returns old value
func (vec *Vector) IncrementMIDForPeer(origin string, rumor bool) uint32 {
	vec.peersLock.Lock()
	defer vec.peersLock.Unlock()
	_, found := vec.nextMessage[origin]
	if !found {
		vec.nextMessage[origin] = 1
	}
	vec.nextMessage[origin] = vec.nextMessage[origin] + 1
	switch rumor {
	case true:
		vec.incrementRumorIDForPeer(origin)
	case false:
		vec.incrementTLCIDForPeer(origin)
	}
	return vec.nextMessage[origin] - 1
}

func (vec *Vector) incrementRumorIDForPeer(peer string) uint32 {
	_, found := vec.nextRumor[peer]
	if !found {
		vec.nextRumor[peer] = 1
	}
	vec.nextRumor[peer]++
	return vec.nextRumor[peer] - 1
}

func (vec *Vector) IncrementRumorIDForPeer(peer string) uint32 {
	vec.peersLock.Lock()
	defer vec.peersLock.Unlock()
	_, found := vec.nextRumor[peer]
	if !found {
		vec.nextRumor[peer] = 1
	}
	vec.nextRumor[peer]++
	return vec.nextRumor[peer] - 1
}

func (vec *Vector) incrementTLCIDForPeer(peer string) uint32 {
	_, found := vec.nextTLC[peer]
	if !found {
		vec.nextTLC[peer] = 1
	}
	vec.nextTLC[peer]++
	return vec.nextTLC[peer] - 1
}

func (vec *Vector) IncrementTLCIDForPeer(peer string) uint32 {
	vec.peersLock.Lock()
	defer vec.peersLock.Unlock()
	_, found := vec.nextTLC[peer]
	if !found {
		vec.nextTLC[peer] = 1
	}
	vec.nextTLC[peer]++
	return vec.nextTLC[peer] - 1
}

// StatusPacket returns the status packet for a given peer.
func (vec *Vector) StatusPacket() *message.StatusPacket {
	sp := &message.StatusPacket{}
	sp.Want = make([]message.PeerStatus, 0)
	vec.peersLock.RLock()
	defer vec.peersLock.RUnlock()
	for peer, mID := range vec.nextMessage {
		ps := message.PeerStatus{peer, mID}
		sp.Want = append(sp.Want, ps)
	}
	return sp
}

// UpdateVectorClock updates the whole vector clock with the given status packet.
func (vec *Vector) UpdateVectorClock(sp message.StatusPacket) {
	vec.peersLock.Lock()
	defer vec.peersLock.Unlock()
	for _, peerStatus := range sp.Want {
		currID := vec.nextMessage[peerStatus.Identifier]
		statusID := peerStatus.NextID
		if statusID > currID {
			vec.nextMessage[peerStatus.Identifier] = statusID
		}
	}
}

// CompareWithStatusPacket compares and returns the difference between
// the current vector clock and the status paket given as arguemnt.
// https://siongui.github.io/2018/03/14/go-set-difference-of-two-arrays/
func (vec *Vector) CompareWithStatusPacket(otherPeerStatus message.StatusPacket) (same bool, toAsk []message.PeerStatus, toSend []message.PeerStatus) {
	toSend = make([]message.PeerStatus, 0)
	toAsk = make([]message.PeerStatus, 0)

	vec.peersLock.RLock()
	defer vec.peersLock.RUnlock()
	// we use a map to compute the difference between arrays
	m := make(map[string]bool)

	// first pass : compute difference with peers contained in other peer statusVector
	for _, status := range otherPeerStatus.Want {
		m[status.Identifier] = true
		next, found := vec.nextMessage[status.Identifier]
		ps := message.PeerStatus{status.Identifier, next}

		if found {
			if next > status.NextID {
				toSend = append(toSend, status)
			} else if next < status.NextID {
				toAsk = append(toAsk, ps)
			}
		} else if status.NextID > 0 {
			toAsk = append(toAsk, ps)
		}
	}
	// second pass : add the peers that are only in this status vector but not in the other one.
	for peer := range vec.nextMessage {
		if !m[peer] {
			ps := message.PeerStatus{peer, 1}
			toSend = append(toSend, ps)
		}
	}
	same = len(toSend) == 0 && len(toAsk) == 0
	return same, toAsk, toSend
}
