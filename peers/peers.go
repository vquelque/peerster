package peers

import (
	"fmt"
	"log"
	"strings"

	. "github.com/deckarep/golang-set" //for peers
)

type Peers struct {
	peers Set
}

func NewPeersSet(peersStr string) *Peers {
	peersSet := &Peers{peers: NewSet()}
	peers := strings.Split(peersStr, ",")
	for _, peer := range peers {
		if peer != "" {
			peersSet.peers.Add(peer)
		}
	}
	return peersSet
}

func (peersSet *Peers) Iterator() *Iterator {
	return peersSet.peers.Iterator()
}

func (peersSet *Peers) Add(peer string) {
	peersSet.peers.Add(peer)
}

func (peersSet *Peers) Delete(peer string) {
	if peersSet.CheckPeerPresent(peer) {
		peersSet.peers.Remove(peer)
	}
}

func (peersSet *Peers) CheckPeerPresent(peer string) bool {
	return peersSet.peers.Contains(peer)
}

func (peersSet *Peers) PrintPeers() string {
	var peersString string
	index := 0
	for peer := range peersSet.peers.Iterator().C {
		if index > 0 {
			peersString += ","
		}
		peersString += peer.(string)
		index++
	}
	return fmt.Sprintf("PEERS : " + peersString)
}

func (peersSet *Peers) PickRandomPeer(sender string) string {
	//pick a peer at random in the set except the peer given as argument
	//returns nil if no other peer int the set
	if peersSet.peers.Cardinality() == 0 {
		return ""
	}
	randPeer := peersSet.peers.Pop()
	defer peersSet.peers.Add(randPeer)
	if randPeer == sender {
		if peersSet.peers.Cardinality() > 0 {
			randPeer2 := peersSet.peers.Pop()
			peersSet.peers.Add(randPeer2)
			log.Println(randPeer2)
			return randPeer2.(string)
		}
		if peersSet.peers.Cardinality() == 0 {
			return randPeer.(string) //no other peers known
		}
		if peersSet.peers.Cardinality() < 0 {
			log.Println("Error. No peers known whereas at least one message received.")
			return ""
		}
	}
	return randPeer.(string)
}

func (peerSet *Peers) GetAllPeers() []string {
	peerList := make([]string, 0)
	for peer := range peerSet.Iterator().C {
		peerList = append(peerList, peer.(string))
	}
	return peerList
}
