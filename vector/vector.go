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

//StatusMessage is the type of message sent by a particular peer when receiving a message used
// to acknowledge all the messages it has received so far.
type StatusPacket struct {
	Want []PeerStatus
}
type Vector struct {
	nextMessage map[string]uint32
	peersLock   sync.RWMutex
}

func NewVector() *Vector {
	vec := &Vector{}
	vec.nextMessage = make(map[string]uint32)
	return vec
}

// GetNextMessagen returns next message id for given peer name.
func (vec *Vector) NextMessageForPeer(peer string) uint32 {
	vec.peersLock.RLock()
	defer vec.peersLock.RUnlock()
	return vec.nextMessage[peer]
}

func (vec *Vector) IncrementMIDForPeer(peer string) uint32 {
	vec.peersLock.Lock()
	defer vec.peersLock.Unlock()
	vec.nextMessage[peer]++
	return vec.nextMessage[peer]
}

func (vec *Vector) StatusPacket() *StatusPacket {
	sp := &StatusPacket{}
	sp.Want = make([]PeerStatus, 0)
	vec.peersLock.RLock()
	defer vec.peersLock.RUnlock()
	for peer, mID := range vec.nextMessage {
		ps := PeerStatus{peer, mID}
		sp.Want = append(sp.Want, ps)
	}
	return sp
}

func (vec *Vector) UpdateVectorClock(sp *StatusPacket) {
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

// https://siongui.github.io/2018/03/14/go-set-difference-of-two-arrays/
func (vec *Vector) CompareWithStatusPacket(otherPeerStatus *StatusPacket) (same bool, toAsk []PeerStatus, toSend []PeerStatus) {
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
			ps := PeerStatus{peer, 0}
			toSend = append(toSend, ps)
		}
	}
	same = len(toSend) == 0 && len(toAsk) == 0
	return same, toAsk, toSend
}

//Prints a PeerStatus message
func (ps *PeerStatus) String() string {
	return fmt.Sprintf("Identifier : %s, NextID : %d", ps.Identifier, ps.NextID)
}

//Prints a StatusMessage
func (msg *StatusPacket) String() string {
	str := fmt.Sprintf("STATUS MESSAGE received \n")
	for _, element := range msg.Want {
		str += fmt.Sprintf("%s \n", element.String())
	}
	return str
}
