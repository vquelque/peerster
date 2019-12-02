package gossiper

import (
	"fmt"
	"log"
	"time"

	"github.com/vquelque/Peerster/constant"
	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/storage"
	"github.com/vquelque/Peerster/utils"
)

func (gsp *Gossiper) processSearchRequest(sr *message.SearchRequest, origin string) {
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
			chunks = append(chunks, index+1)
		}
		sr := message.NewSearchResult(f.Name, f.MetafileHash[:], chunks, count)
		results = append(results, sr)
	}
	if len(matches) > 0 {
		// log.Printf("GOT MATCHING FILE SENDING REPLY\n")
		reply := message.NewSearchReply(gsp.Name, sr.Origin, constant.DefaultHopLimit, results)
		gsp.sendSearchReply(reply)
	}
	if sr.Budget > 1 {
		// log.Printf("DISTRIBUTING REQUEST WITH BUDGET %d", sr.Budget)
		gsp.distributeSearchRequest(sr, origin)
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

func (gsp *Gossiper) distributeSearchRequest(sr *message.SearchRequest, origin string) {
	sr.Budget = sr.Budget - 1
	neighbors := gsp.Peers.GetAllPeersExcept(origin)
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
	//log.Printf("Sending search request to %s with budget %d \n", peer, sr.Budget)
	gp := &GossipPacket{SearchRequest: sr}
	gsp.send(gp, peer)
}

// search request initiated from this peer
func (gsp *Gossiper) startSearchRequest(keywords []string, budget uint64) {
	log.Printf("STARTING SEARCH REQUEST WITH KEYWORDS %s AND BUDGET %d", keywords, budget)
	expandingSearch := false
	if budget == 0 {
		expandingSearch = true
		budget = 2
	}
	timeout := 0
	sr := message.NewSearchRequest(gsp.Name, keywords, budget)
	currBudget := budget
	gsp.processSearchRequest(sr, "")
	rTimerDuration := time.Duration(constant.SearchRequestResendTimer) * time.Second
	timer := time.NewTicker(rTimerDuration)
	match := gsp.WaitingForSearchReply.RegisterSearchObserver(sr)
	fullMatch := 0
	matches := make(map[utils.SHA256]bool)     //metahash --> bool
	nMatches := make(map[utils.SHA256]uint32)  //metahash --> number of match
	filenames := make(map[utils.SHA256]string) //filename temp
	defer func() {
		gsp.WaitingForSearchReply.UnregisterSearchObserver(sr)
		gsp.PendingSearchRequest.Delete(sr)
		timer.Stop()
	}()
	for {
		select {
		case <-timer.C:
			timeout++
			if expandingSearch {
				if currBudget < constant.MaxBudget {
					currBudget *= 2
					sr.Budget = currBudget
					gsp.distributeSearchRequest(sr, "")
					log.Printf("EXPANDING SEARCH CIRCLE BUDGET %d \n", currBudget)
				}
			}
			if timeout > constant.SearchRequestMaxRetries {
				fmt.Printf("REQUEST TIMEOUT \n")
				return
			}
		case reply := <-match:
			for _, r := range reply.Results {
				metahash := utils.SliceToHash(r.MetafileHash)
				new := gsp.SearchResults.AddSearchResult(r, reply.Origin)
				if uint64(len(r.ChunkMap)) == r.ChunkCount {
					fullMatch++
				}
				if new {
					fmt.Printf("FOUND match %s at %s metafile=%x chunks=%s \n", r.FileName, reply.Origin, r.MetafileHash, utils.ChunkMapToString(r.ChunkMap))
					matches[metahash] = false
					filenames[metahash] = r.FileName
				}
			}
			var nMatch uint32 = 0
			for k, b := range matches {
				switch b {
				case false:
					if gsp.SearchResults.IsComplete(k) {
						cMap := gsp.SearchResults.GetChunksSourceMap(k)
						gsp.ToDownload.AddFileToDownload(k, filenames[k], cMap)
						gsp.UIStorage.AddDownloadableFile(filenames[k], k)
						matches[k] = true
						nMatches[k] = nMatches[k] + 1
						nMatch += nMatches[k]
					}
				case true:
					nMatch += nMatches[k]
				}
			}

			if fullMatch >= constant.SearchMatchThreshold {
				fmt.Printf("SEARCH FINISHED \n")
				for m := range matches {
					gsp.SearchResults.Clear(m)
				}
				return
			}
		}
	}
}

func (gsp *Gossiper) StartSearchedFileDownload(metahash utils.SHA256, filename string) {
	//filename := gsp.ToDownload.GetFilename(metahash)
	sources := gsp.ToDownload.GetChunkSources(metahash)
	if sources != nil {
		gsp.startFileDownload(metahash, sources[0][0], filename, sources)
	} else {
		fmt.Printf("No peer known for this file \n")
	}
}
