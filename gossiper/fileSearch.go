package gossiper

import (
	"time"

	"github.com/vquelque/Peerster/constant"
	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/storage"
)

func (gsp *Gossiper) processSearchRequest(sr *message.SearchRequest) {
	// check if we already received this request
	if gsp.PendingSearchRequest.CheckRequestPresent(sr) {
		// search request already received -> do not process
		return
	}
	gsp.registerSearchRequest(sr)
	// search for local files matching a keyword
	var matches []*storage.File
	var results []*message.SearchResult
	for _, kw := range sr.Keywords {
		m := gsp.FileStorage.SearchForFile(kw)
		matches = append(matches, m...)
	}
	for _, f := range matches {
		count := gsp.FileStorage.ChunkCount(f.MetafileHash)
		chunks := make([]uint64, count)
		var index uint64
		for index = 0; index < count; index++ {
			chunks = append(chunks, index)
		}
		sr := message.NewSearchResult(f.Name, f.MetafileHash[:], chunks, count)
		results = append(results, sr)
	}
	if len(matches) > 0 {
		reply := message.NewSearchReply(gsp.Name, sr.Origin, constant.DefaultHopLimit, results)
		gsp.sendSearchReply(reply)
	}
	sr.Budget = sr.Budget - 1
	if sr.Budget > 0 {
		gsp.distributeSearchRequest(sr)
	}
}

func (gsp *Gossiper) sendSearchReply(r *message.SearchReply) {
	r.HopLimit = r.HopLimit - 1
	gp := &GossipPacket{SearchReply: r}
	nextHopAddr := gsp.Routing.GetRoute(r.Destination)
	// fmt.Printf("SENDING Search REPLY TO DEST %s VIA %s \n", r.Destination, nextHopAddr)
	if nextHopAddr != "" {
		gsp.send(gp, nextHopAddr)
	}
}

func (gsp *Gossiper) distributeSearchRequest(sr *message.SearchRequest) {
	neighbors := gsp.Peers.GetAllPeers()
	peers := uint64(len(neighbors))
	//split remaining budget evenly accross peers
	mBudget := sr.Budget / peers
	rBudget := sr.Budget % peers
	msr := message.NewSearchRequest(sr.Keywords, mBudget)
	rsr := message.NewSearchRequest(sr.Keywords, mBudget+1)
	for _, p := range neighbors {
		if rBudget > 0 {
			gsp.sendSearchRequest(rsr, p)
			rBudget--
		} else {
			gsp.sendSearchRequest(msr, p)
		}
	}
}

func (gsp *Gossiper) registerSearchRequest(sr *message.SearchRequest) {
	gsp.PendingSearchRequest.Add(sr)
	rTimerDuration := time.Duration(constant.SearchRequestTimeout) * time.Millisecond
	timer := time.NewTicker(rTimerDuration)
	//deregister after timeout
	go func() {
		select {
		case <-timer.C:
			// timer elapsed : unregister search request
			gsp.PendingSearchRequest.Delete(sr)
			return
		}
	}()
}

func (gsp *Gossiper) sendSearchRequest(sr *message.SearchRequest, peer string) {
	if sr.Budget > 0 {
		gp := &GossipPacket{SearchRequest: sr}
		gsp.send(gp, peer)
	}
}
