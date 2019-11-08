package vector

import (
	"fmt"
	"sync"
)

//PeerStatus gives the next message ID to be received by a particular peer.
type PeerStatus struct {
	Identifier string
	NextID     uint32
}

//StatusPacket is exchanged between peers to exchange their vector clocks.
type StatusPacket struct {
	Want []PeerStatus
}

// Vector clock.
// We use ID for messages starting at 0
type Vector struct {
	nextMessage map[string]uint32
	peersLock   sync.RWMutex
}

// NewVector returns a new empty vector clocl
func NewVector() *Vector {
	vec := &Vector{nextMessage: make(map[string]uint32), peersLock: sync.RWMutex{}}
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

// Increments message ID for the given peer.
func (vec *Vector) IncrementMIDForPeer(peer string) uint32 {
	vec.peersLock.Lock()
	defer vec.peersLock.Unlock()
	_, found := vec.nextMessage[peer]
	if !found {
		vec.nextMessage[peer] = 1
	}
	vec.nextMessage[peer]++
	return vec.nextMessage[peer]
}

// StatusPacket returns the status packet for a given peer.
func (vec *Vector) StatusPacket() StatusPacket {
	sp := StatusPacket{}
	sp.Want = make([]PeerStatus, 0)
	vec.peersLock.RLock()
	defer vec.peersLock.RUnlock()
	for peer, mID := range vec.nextMessage {
		ps := PeerStatus{peer, mID}
		sp.Want = append(sp.Want, ps)
	}
	return sp
}

// UpdateVectorClock updates the whole vector clock with the given status packet.
func (vec *Vector) UpdateVectorClock(sp StatusPacket) {
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
func (vec *Vector) CompareWithStatusPacket(otherPeerStatus StatusPacket) (same bool, toAsk []PeerStatus, toSend []PeerStatus) {
	toSend = make([]PeerStatus, 0)
	toAsk = make([]PeerStatus, 0)

	vec.peersLock.RLock()
	defer vec.peersLock.RUnlock()
	// we use a map to compute the difference between arrays
	m := make(map[string]bool)

	// first pass : compute difference with peers contained in other peer statusVector
	for _, status := range otherPeerStatus.Want {
		m[status.Identifier] = true
		next, found := vec.nextMessage[status.Identifier]
		ps := PeerStatus{status.Identifier, next}

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
			ps := PeerStatus{peer, 1}
			toSend = append(toSend, ps)
		}
	}
	same = len(toSend) == 0 && len(toAsk) == 0
	return same, toAsk, toSend
}

//Prints a PeerStatus message
func (ps *PeerStatus) String() string {
	return fmt.Sprintf("peer %s nextID %d", ps.Identifier, ps.NextID)
}

//StringStatusWithSender prints a StatusMessage with its sender
func (msg *StatusPacket) StringStatusWithSender(sender string) string {
	str := fmt.Sprintf("STATUS from %s \n", sender)
	for _, element := range msg.Want {
		str += fmt.Sprintf("%s \n", element.String())
	}
	return str
}
