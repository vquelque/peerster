package peers

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
)

type Peers struct {
	peers map[string]bool
	lock  sync.RWMutex
}

func NewPeersSet(peersStr string) *Peers {
	peersSet := &Peers{peers: make(map[string]bool), lock: sync.RWMutex{}}
	peersSet.lock.Lock()
	defer peersSet.lock.Unlock()
	peers := strings.Split(peersStr, ",")
	for _, peer := range peers {
		_, ok := peersSet.peers[peer]
		if peer != "" && !ok {
			peersSet.peers[peer] = true
		}
	}
	return peersSet
}

func (peersSet *Peers) Add(peer string) {
	peersSet.lock.Lock()
	defer peersSet.lock.Unlock()
	_, ok := peersSet.peers[peer]
	if !ok {
		peersSet.peers[peer] = true
	}
}

func (peersSet *Peers) Delete(peer string) {
	peersSet.lock.Lock()
	defer peersSet.lock.Unlock()
	_, ok := peersSet.peers[peer]
	if ok {
		delete(peersSet.peers, peer)
	}
}

func (peersSet *Peers) CheckPeerPresent(peer string) bool {
	peersSet.lock.RLock()
	defer peersSet.lock.RUnlock()
	_, ok := peersSet.peers[peer]
	return ok
}

func (peersSet *Peers) PrintPeers() string {
	var peersString string
	index := 0
	for _, peer := range peersSet.GetAllPeers() {
		if index > 0 {
			peersString += ","
		}
		peersString += peer
		index++
	}
	return fmt.Sprintf("PEERS : " + peersString)
}

func (peersSet *Peers) PickRandomPeer(sender string) string {
	peersSet.lock.RLock()
	defer peersSet.lock.RUnlock()
	//pick a peer at random in the set except the peer given as argument
	//returns nil if no other peer int the set
	if peersSet.Size() == 0 {
		return ""
	}
	slice := peersSet.GetAllPeers()
	i1 := rand.Intn(len(slice))
	p1 := slice[i1]

	if p1 == sender {
		if len(slice) > 1 {
			i2 := rand.Intn(len(slice))
			p2 := slice[i2]
			return p2
		}
		return "" //no other peers known
	}
	return p1
}

func (peerSet *Peers) GetAllPeers() []string {
	peerSet.lock.RLock()
	defer peerSet.lock.RUnlock()
	peerList := make([]string, 0)
	for peer := range peerSet.peers {
		peerList = append(peerList, peer)
	}
	return peerList
}

func (peerSet *Peers) Size() int {
	peerSet.lock.RLock()
	defer peerSet.lock.RUnlock()
	return len(peerSet.peers)
}
