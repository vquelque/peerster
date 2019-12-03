package storage

import (
	"fmt"
	"sync"

	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/utils"
)

type PendingRequests struct {
	requests map[string]bool
	lock     sync.RWMutex
}

type SearchResults struct {
	results map[utils.SHA256]map[uint64][]string // file metahash --> chiunkcount --> peer
	lock    sync.RWMutex
}

func NewPendingRequests() *PendingRequests {
	return &PendingRequests{requests: make(map[string]bool), lock: sync.RWMutex{}}
}

func NewSearchResult() *SearchResults {
	return &SearchResults{results: make(map[utils.SHA256]map[uint64][]string, 0)}
}

func (pr *PendingRequests) Add(r *message.SearchRequest) {
	pr.lock.Lock()
	defer pr.lock.Unlock()
	pr.requests[GetRequestID(r)] = true
}

func (pr *PendingRequests) Delete(r *message.SearchRequest) {
	pr.lock.Lock()
	defer pr.lock.Unlock()
	delete(pr.requests, GetRequestID(r))

}

func (pr *PendingRequests) CheckPendingRequestPresent(r *message.SearchRequest) bool {
	pr.lock.RLock()
	defer pr.lock.RUnlock()
	_, ok := pr.requests[GetRequestID(r)]
	return ok
}

func (sr *SearchResults) AddSearchResult(r *message.SearchResult, origin string) bool {
	new := false
	sr.lock.Lock()
	defer sr.lock.Unlock()
	metahash := utils.SliceToHash(r.MetafileHash)
	_, found := sr.results[metahash]
	if !found {
		sr.results[metahash] = make(map[uint64][]string, r.ChunkCount)
	}
	for _, ch := range r.ChunkMap {
		c := ch - 1
		if !utils.Contains(sr.results[metahash][c], origin) {
			_, found := sr.results[metahash][c]
			if !found {
				sr.results[metahash][c] = make([]string, 0)
			}
			sr.results[metahash][c] = append(sr.results[metahash][c], origin)
			new = true
		}
	}
	return new
}

func (sr *SearchResults) IsComplete(metahash utils.SHA256) bool {
	sr.lock.RLock()
	defer sr.lock.RUnlock()
	for _, c := range sr.results[metahash] {
		if len(c) < 1 {
			return false
		}
	}
	return true
}

func (sr *SearchResults) CountFullMatch(metahash utils.SHA256) uint64 {
	sr.lock.RLock()
	defer sr.lock.RUnlock()
	candidate := sr.results[metahash][0]
	var nMatch uint64 = 0
	for _, peer := range candidate {
		match := true
		for _, c := range sr.results[metahash] {
			if !utils.Contains(c, peer) {
				match = false
				break
			}
		}
		if match == true {
			nMatch++
		}
	}
	return nMatch
}

func (sr *SearchResults) GetChunksSourceMap(metahash utils.SHA256) map[uint64][]string {
	sr.lock.RLock()
	defer sr.lock.RUnlock()
	cMap, found := sr.results[metahash]
	if found {
		return cMap
	}
	return nil
}

func (sr *SearchResults) Clear(metahash utils.SHA256) {
	sr.lock.Lock()
	defer sr.lock.Unlock()
	delete(sr.results, metahash)
}

func GetRequestID(r *message.SearchRequest) string {
	return fmt.Sprintf("%s:%s", r.Origin, r.Keywords)
}
