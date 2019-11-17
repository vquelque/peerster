package storage

import (
	"sync"

	"github.com/vquelque/Peerster/message"
)

type PendingRequests struct {
	requests map[*message.SearchRequest]bool
	lock     sync.RWMutex
}

func NewPendingRequests() *PendingRequests {
	return &PendingRequests{requests: make(map[*message.SearchRequest]bool), lock: sync.RWMutex{}}
}

func (pr *PendingRequests) Add(r *message.SearchRequest) {
	pr.lock.Lock()
	defer pr.lock.Unlock()
	_, ok := pr.requests[r]
	if !ok {
		pr.requests[r] = true
	}
}

func (pr *PendingRequests) Delete(r *message.SearchRequest) {
	pr.lock.Lock()
	defer pr.lock.Unlock()
	_, ok := pr.requests[r]
	if ok {
		delete(pr.requests, r)
	}
}

func (pr *PendingRequests) CheckRequestPresent(r *message.SearchRequest) bool {
	pr.lock.RLock()
	defer pr.lock.RUnlock()
	_, ok := pr.requests[r]
	return ok
}
