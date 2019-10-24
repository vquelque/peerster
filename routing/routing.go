package routing

import (
	"fmt"
	"sync"

	"github.com/vquelque/Peerster/message"
)

// Peer RoutingTable
type RoutingTable struct {
	routes map[string]string //key -> Origin, value -> address:portnumber
	lock   sync.RWMutex
}

// RoutingTable returns a new routing table
func NewRoutingTable() *RoutingTable {
	rt := &RoutingTable{
		routes: make(map[string]string),
		lock:   sync.RWMutex{},
	}
	return rt
}

// AddRoute add a route to the routing table
// In fact, nextHopAddr is the next hop for origin and is also the peer from which
// we received the rumor message.
func (rt *RoutingTable) AddRoute(origin string, nextHopAddr string) {
	rt.lock.Lock()
	rt.routes[origin] = nextHopAddr
	rt.lock.Unlock()
}

// DeleteRoute deletes a route from the routing table
func (rt *RoutingTable) DeleteRoute(origin string) {
	rt.lock.Lock()
	delete(rt.routes, origin)
	rt.lock.Unlock()
}

func (rt *RoutingTable) PrintUpdate(origin string) {
	rt.lock.RLock()
	defer rt.lock.RUnlock()
	fmt.Printf("DSDV %s %s", origin, rt.routes[origin])
}

func (rt *RoutingTable) Contains(origin string) bool {
	rt.lock.RLock()
	defer rt.lock.RUnlock()
	_, ok := rt.routes[origin]
	return ok
}

func (rt *RoutingTable) UpdateRoute(msg *message.RumorMessage, sender string) {
	if msg.ID == 1 {
		// initial message
		rt.AddRoute(msg.Origin, sender)
		return
	}
	rt.lock.Lock()
	defer rt.lock.Unlock()
	route, found := rt.routes[msg.Origin]
	if found && sender != route {
		rt.routes[msg.Origin] = sender
		rt.PrintUpdate(msg.Origin)
	}
}
