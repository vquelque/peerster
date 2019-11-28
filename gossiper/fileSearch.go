package gossiper

import (
	"fmt"
	"time"

	"github.com/vquelque/Peerster/constant"
	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/storage"
	"github.com/vquelque/Peerster/utils"
)

func (gsp *Gossiper) processSearchRequest(sr *message.SearchRequest) {
	if gsp.PendingSearchRequest.CheckPendingRequestPresent(sr) {
		// search request already received -> do not process
		// fmt.Printf("SR ALREADY REGISTERED \n")
		return
	}
	gsp.registerSearchRequest(sr)
	// fmt.Printf("PROCESSING SEARCH REQUEST FROM %s\n", sr.Origin)
	// search for local files matching a keyword
	var matches []*storage.File
	var results []*message.SearchResult
	for _, kw := range sr.Keywords {
		m := gsp.FileStorage.SearchForFile(kw)
		matches = append(matches, m...)
	}
	for _, f := range matches {
		count := gsp.FileStorage.ChunkCount(f.MetafileHash)
		chunks := make([]uint64, 0)
		var index uint64
		for index = 0; index < count; index++ {
			chunks = append(chunks, index)
		}
		sr := message.NewSearchResult(f.Name, f.MetafileHash[:], chunks, count)
		results = append(results, sr)
	}
	if len(matches) > 0 {
		// log.Printf("GOT MATCHING FILE SENDING REPLY\n")
		reply := message.NewSearchReply(gsp.Name, sr.Origin, constant.DefaultHopLimit, results)
		gsp.sendSearchReply(reply)
	}
	sr.Budget = sr.Budget - 1
	if sr.Budget > 0 {
		// log.Printf("DISTRIBUTING REQUEST WITH BUDGET %d", sr.Budget)
		gsp.distributeSearchRequest(sr)
	}
}

func (gsp *Gossiper) processSearchReply(r *message.SearchReply) {
	// received a reply check if it is for us else forward
	if r.Destination != gsp.Name {
		gsp.sendSearchReply(r)
		return
	}
	for _, sr := range r.Results {
		gsp.WaitingForSearchReply.SendMatchToSearchObserver(r, sr.FileName)
	}

}

func (gsp *Gossiper) sendSearchReply(r *message.SearchReply) {
	r.HopLimit = r.HopLimit - 1
	gp := &GossipPacket{SearchReply: r}
	var nextHopAddr string
	if r.Destination == gsp.Name {
		nextHopAddr = gsp.PeersSocket.Address()
	} else {
		nextHopAddr = gsp.Routing.GetRoute(r.Destination)
	}
	// fmt.Printf("SENDING SEARCH REPLY TO DEST %s VIA %s \n", r.Destination, nextHopAddr)
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
	// log.Printf("mBudget %d, rBudget %d \n", mBudget, rBudget)
	msr := message.NewSearchRequest(sr.Origin, sr.Keywords, mBudget)
	rsr := message.NewSearchRequest(sr.Origin, sr.Keywords, mBudget+1)
	for _, p := range neighbors {
		if rBudget > 0 {
			gsp.sendSearchRequest(rsr, p)
			rBudget = rBudget - 1
		} else {
			gsp.sendSearchRequest(msr, p)
		}
	}
}

func (gsp *Gossiper) registerSearchRequest(sr *message.SearchRequest) {
	gsp.PendingSearchRequest.Add(sr)
	// log.Printf("REGISTERED SR WITH ID %s \n", storage.GetRequestID(sr))
	rTimerDuration := time.Duration(constant.SearchRequestTimeout) * time.Millisecond
	timer := time.NewTicker(rTimerDuration)
	//deregister after timeout
	go func() {
		select {
		case <-timer.C:
			// timer elapsed : unregister search request
			// log.Printf("UNREGISTERED SR WITH ID %s \n", storage.GetRequestID(sr))
			gsp.PendingSearchRequest.Delete(sr)
			timer.Stop()
		}
	}()
}

func (gsp *Gossiper) sendSearchRequest(sr *message.SearchRequest, peer string) {
	if sr.Budget > 0 {
		// log.Printf("Sending search request to %s with budget %d \n", peer, sr.Budget)
		gp := &GossipPacket{SearchRequest: sr}
		gsp.send(gp, peer)
	}
}

// search request initiated from this peer
func (gsp *Gossiper) startSearchRequest(keywords []string, budget uint64) {
	//TODO CHECK IF WE DO NOT STILL HAVE A PENDING REQUEST FOR THIS KEYWORD ?
	// log.Printf("STARTING SEARCH REQUEST WITH KEYWORDS %s AND BUDGET %d", keywords, budget)
	expandingSearch := false
	if budget == 0 {
		expandingSearch = true
		budget = 2
	}
	timeout := 0
	sr := message.NewSearchRequest(gsp.Name, keywords, budget)
	gsp.processSearchRequest(sr)
	rTimerDuration := time.Duration(constant.SearchRequestResendTimer) * time.Second
	timer := time.NewTicker(rTimerDuration)
	match := gsp.WaitingForSearchReply.RegisterSearchObserver(sr)
	matches := make(map[utils.SHA256]bool)     //metahash --> bool
	filenames := make(map[utils.SHA256]string) //filename temp
	defer func() {
		gsp.WaitingForSearchReply.UnregisterSearchObserver(sr)
		fmt.Printf("CLEANING SEARCH")
		for m, _ := range matches {
			gsp.SearchResults.Clear(m)
		}
		timer.Stop()
	}()
	for {
		select {
		case <-timer.C:
			timeout++
			if timeout > constant.SearchRequestMaxRetries {
				fmt.Printf("REQUEST TIMEOUT")
				return
			}
			if expandingSearch {
				if sr.Budget < constant.MaxBudget {
					sr.Budget = sr.Budget * 2
					gsp.distributeSearchRequest(sr)
					// log.Printf("EXPANDING SEARCH CIRCLE BUDGET %d \n", sr.Budget)
				}
			}
		case reply := <-match:
			for _, r := range reply.Results {
				metahash := utils.SliceToHash(r.MetafileHash)
				new := gsp.SearchResults.AddSearchResult(r, reply.Origin)
				if new {
					fmt.Printf("FOUND match %s at %s metafile=%x chunks=%s \n", r.FileName, reply.Origin, r.MetafileHash, utils.ChunkMapToString(r.ChunkMap))
				}
				_, f := matches[metahash]
				if !f {
					matches[metahash] = false
				}
				filenames[metahash] = r.FileName
			}
			nMatch := 0
			for k, b := range matches {
				switch b {
				case false:
					if gsp.SearchResults.IsComplete(k) {
						matches[k] = true
						nMatch++
					}
				case true:
					nMatch++
				}
			}

			if nMatch >= constant.SearchMatchThreshold {
				fmt.Printf("SEARCH FINISHED \n")
				for m, b := range matches {
					if b == true {
						cMap := gsp.SearchResults.GetChunksSourceMap(m)
						gsp.ToDownload.AddFileToDownload(m, filenames[m], cMap)
						gsp.UIStorage.AddDownloadableFile(filenames[m], m)
						gsp.SearchResults.Clear(m)
					}
				}
				return
			}
		}
	}
}

func (gsp *Gossiper) StartSearchedFileDownload(metahash utils.SHA256) {
	filename := gsp.ToDownload.GetFilename(metahash)
	sources := gsp.ToDownload.GetChunkSources(metahash)
	gsp.startFileDownload(metahash, sources[0][0], filename, sources)
}
