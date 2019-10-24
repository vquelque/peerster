package routing

import (
	"fmt"
	"sync"

	"github.com/vquelque/Peerster/message"
)

// Routing struct is a RoutingTable
type Routing struct {
	routes map[string]string //key -> Origin, value -> address:portnumber
	lock   sync.RWMutex
}

// RoutingTable returns a new routing table
func NewRoutingTable() *Routing {
	rt := &Routing{
		routes: make(map[string]string),
		lock:   sync.RWMutex{},
	}
	return rt
}

// AddRoute add a route to the routing table
// In fact, nextHopAddr is the next hop for origin and is also the peer from which
// we received the rumor message.
func (rt *Routing) AddRoute(origin string, nextHopAddr string) {
	rt.lock.Lock()
	defer rt.lock.Unlock()
	rt.routes[origin] = nextHopAddr
}

// DeleteRoute deletes a route from the routing table
func (rt *Routing) DeleteRoute(origin string) {
	rt.lock.Lock()
	defer rt.lock.Unlock()
	delete(rt.routes, origin)
}

func (rt *Routing) PrintUpdate(origin string) string {
	rt.lock.RLock()
	defer rt.lock.RUnlock()
	return fmt.Sprintf("DSDV %s %s", origin, rt.routes[origin])
}

func (rt *Routing) Contains(origin string) bool {
	rt.lock.RLock()
	defer rt.lock.RUnlock()
	_, ok := rt.routes[origin]
	return ok
}

func (rt *Routing) GetRoute(origin string) string {
	rt.lock.RLock()
	defer rt.lock.RUnlock()
	return rt.routes[origin]
}

func (rt *Routing) UpdateRoute(msg *message.RumorMessage, sender string) {
	rt.lock.Lock()
	defer rt.lock.Unlock()
	origin := msg.Origin
	if msg.ID == 1 {
		// initial message
		rt.routes[origin] = sender
		return
	}

	route, found := rt.routes[origin]
	if found {
		if sender != route {
			rt.routes[origin] = sender
		}
	}
}
