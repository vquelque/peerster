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

func (peersSet *Peers) PrintPeers() string {
	peersString := peersSet.peers.String()
	return fmt.Sprintf("PEERS : " + peersString[4:len(peersString)-1])
}

func (peersSet *Peers) PickRandomPeer(sender string) string {
	//pick a peer at random in the set except the peer given as argument
	//returns nil if no other peer int the set
	if peersSet.peers.Cardinality() == 0 {
		return ""
	}
	randPeer := peersSet.peers.Pop()
	if randPeer == sender {
		if peersSet.peers.Cardinality() > 1 {
			randPeer2 := peersSet.peers.Pop()
			peersSet.peers.Add(randPeer2)
			return randPeer2.(string)
		}
		if peersSet.peers.Cardinality() == 1 {
			return "" //no other peers known
		}
		if peersSet.peers.Cardinality() < 0 {
			log.Println("Error. No peers known whereas at least one message received.")
			return ""
		}
	}
	peersSet.peers.Add(randPeer)
	return randPeer.(string)
}
