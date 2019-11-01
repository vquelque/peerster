package gossiper

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/vquelque/Peerster/message"
	"github.com/vquelque/Peerster/storage"
	"github.com/vquelque/Peerster/utils"
)

const ChunkSize = 8000 //in bytes
const FileTempDirectory = "./_SharedFiles/"
const FileOutDirectory = "./_Downloads/"
const maxChunkDownloadTries = 10

func (gsp *Gossiper) processFile(filename string) {
	fileURI := FileTempDirectory + filename
	file, err := os.Open(fileURI)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	buffer := make([]byte, ChunkSize)
	metafile := make([]byte, 0)
	var count uint32 = 0

	for {
		//for each chunk of 8KB
		bytesread, err := file.Read(buffer)
		hash := sha256.Sum256(buffer[:bytesread])
		metafile = append(metafile, hash[:]...)
		if err != nil {
			if err != io.EOF {
				//error reading file
				log.Println(err)
				//TODO clean all previously stored chunks
				return
			}
			//EOF
			break
		}
		count++
		data := make([]byte, bytesread)
		copy(data, buffer)
		c := &storage.Chunk{Data: data, Hash: hash}
		gsp.FileStorage.StoreChunk(c)
	}
	metaHash := sha256.Sum256(metafile)
	f := &storage.File{Name: filename, MetafileHash: metaHash, ChunkCount: count}
	gsp.FileStorage.StoreFile(f, metafile)
	fmt.Printf("File stored in memory. Metahash : %x\n", metaHash)
}

func (gsp *Gossiper) startFileDownload(metahash utils.SHA256, peer string, filename string) *storage.File {
	var f *storage.File
	file := gsp.FileStorage.GetFile(metahash)
	if file != nil && file.Completed {
		// already have file
		fmt.Print("File already downloaded")
		f = file
		return f
	}
	if file == nil {
		f = &storage.File{Name: filename, MetafileHash: metahash, ChunkCount: 0}
	}
	go func() {
		//get or request metafile
		meta := gsp.FileStorage.GetMetafile(metahash)
		if meta == nil {
			fmt.Printf("DOWNLOADING metafile of %s from %s\n", filename, peer)
			meta = gsp.downloadFromPeer(metahash, peer)
			gsp.FileStorage.StoreMetafile(metahash, meta)
		}
		toDownload := len(meta) / sha256.Size //number of chunks to download

		// create the slice of all hashes
		var chunksHash []utils.SHA256
		for i := 0; i < toDownload; i += sha256.Size {
			j := i + sha256.Size - 1
			var hash utils.SHA256
			copy(hash[:], meta[i:j])
			chunksHash = append(chunksHash, hash)
		}
		// download all the chunks
		for f.ChunkCount < uint32(toDownload) {
			h := chunksHash[f.ChunkCount]
			fmt.Printf("DOWNLOADING %s chunk %d from %s \n", filename, f.ChunkCount+1, peer)
			data := gsp.downloadFromPeer(h, peer)
			chunk := &storage.Chunk{data, h}
			gsp.FileStorage.StoreChunk(chunk)
			f.ChunkCount++
		}

		out, err := os.Create(FileOutDirectory + filename)
		if err != nil {
			fmt.Println("Impossible to create a new file", err)
			return
		}
		//have all the chunks. Reconstructing the file
		gsp.FileStorage.WriteChunksToFile(chunksHash, out)
		file.Completed = true
		fmt.Printf("RECONSTRUCTED file %s", filename)
	}()
	return f
}

func (gsp *Gossiper) downloadFromPeer(hash utils.SHA256, peer string) []byte {
	var data []byte
	tries := 0
	for tries < maxChunkDownloadTries {
		tries++
		callback := gsp.WaitingForData.RegisterFileObserver(hash)
		dr := message.NewDataRequest(gsp.Name, peer, 0, hash)
		gsp.forwardDataRequest(dr)
		reply := <-callback
		gsp.WaitingForData.UnregisterFileObserver(hash)
		data := reply.Data
		h := sha256.Sum256(data)
		if bytes.Compare(h[:], reply.HashValue) == 0 {
			//data is good
			break
		}
	}
	return data
}
func (gsp *Gossiper) processDataRequest(dr *message.DataRequest) {
	if dr.Destination != gsp.Name {
		if dr.HopLimit == 0 {
			return
		}
		gsp.forwardDataRequest(dr)
		return
	}
	// this data request is for us
	var hash utils.SHA256
	copy(hash[:], dr.HashValue) //convert slice to hash array
	data := gsp.FileStorage.GetChunkOrMeta(hash)
	if data != nil {
		r := message.NewDataReply(gsp.Name, 0, dr, data)
		gsp.forwardDataReply(r)
	}
}

func (gsp *Gossiper) processDataReply(r *message.DataReply) {
	if r.Destination != gsp.Name {
		if r.HopLimit == 0 {
			return
		}
		gsp.forwardDataReply(r)
		return
	}
	// this data reply is for us
	if gsp.WaitingForData.
}

func (gsp *Gossiper) forwardDataRequest(dr *message.DataRequest) {
	gp := &GossipPacket{DataRequest: dr}
	dr.HopLimit--
	nextHopAddr := gsp.Routing.GetRoute(dr.Destination)
	// println("sending data request to " + dr.Destination + " via " + nextHopAddr)
	if nextHopAddr != "" {
		gsp.send(gp, nextHopAddr)
	}
}

func (gsp *Gossiper) forwardDataReply(r *message.DataReply) {
	gp := &GossipPacket{DataReply: r}
	r.HopLimit--
	nextHopAddr := gsp.Routing.GetRoute(r.Destination)
	// println("sending dara reply to " + r.Destination + " via " + nextHopAddr)
	if nextHopAddr != "" {
		gsp.send(gp, nextHopAddr)
	}
}
