package gossiper

import (
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
const FileTempDirectory = "./_sharedFiles/"

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
	count := 0

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
	f := &storage.File{Name: filename, MetafileHash: metaHash}
	gsp.FileStorage.StoreFile(f, metafile)
	fmt.Printf("File stored in memory. Metahash : %x\n", metaHash)
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

func (gsp *Gossiper) processDataReply(msg *message.DataReply) {

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
