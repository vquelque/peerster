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

// RoutingTable returns a new routing table with routing for direct neighbors
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
	rt.lock.Lock()
	defer rt.lock.Unlock()
	return fmt.Sprintf("DSDV %s %s", origin, rt.routes[origin])
}

func (rt *Routing) Contains(origin string) bool {
	rt.lock.Lock()
	defer rt.lock.Unlock()
	_, ok := rt.routes[origin]
	return ok
}

func (rt *Routing) GetRoute(origin string) string {
	rt.lock.Lock()
	defer rt.lock.Unlock()
	return rt.routes[origin]
}

func (rt *Routing) UpdateRoute(msg *message.RumorMessage, sender string) {
	rt.lock.Lock()
	defer rt.lock.Unlock()
	origin := msg.Origin
	// print("update route for " + origin + " : " + sender)
	rt.routes[origin] = sender
}

func (rt *Routing) String() string {
	rt.lock.Lock()
	defer rt.lock.Unlock()
	str := "Routing table :"
	for origin, route := range rt.routes {
		str += "\n"
		str += origin + " : " + route
	}
	return str
}

// GetAllRoutes returns a new map with origins and corresponding route names
func (rt *Routing) GetAllRoutes() map[string]string {
	rt.lock.Lock()
	defer rt.lock.Unlock()
	allRoutes := make(map[string]string)
	for origin, route := range rt.routes {
		allRoutes[origin] = route
	}
	return allRoutes
}
