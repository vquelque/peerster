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

const ChunkSize = 8192 //in bytes
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

	EOF := false
	for {
		//for each chunk of 8KB
		bytesread, err := file.Read(buffer)
		if err != nil {
			if err != io.EOF {
				//error reading file
				log.Println(err)
				//TODO clean all previously stored chunks
				return
			}
			EOF = true
			break
		}
		count++
		hash := sha256.Sum256(buffer[:bytesread])
		metafile = append(metafile, hash[:]...)
		data := make([]byte, bytesread)
		copy(data, buffer[:bytesread])
		c := &storage.Chunk{Data: data, Hash: hash}
		gsp.FileStorage.StoreChunk(c)
		fmt.Printf("CHUNK %d STORED. HASH %x. \n", count, hash)
		if EOF {
			break
		}
	}
	metaHash := sha256.Sum256(metafile)
	f := &storage.File{Name: filename, MetafileHash: metaHash, ChunkCount: count}
	gsp.FileStorage.StoreFile(f, metafile)
	fmt.Printf("File stored in memory. Metahash : %x\n", metaHash)
	fmt.Printf("METAFILE CONTENT : %x\n", gsp.FileStorage.GetMetafile(f.MetafileHash))
}

func (gsp *Gossiper) startFileDownload(metahash utils.SHA256, peer string, filename string) *storage.File {
	file := gsp.FileStorage.GetFile(metahash)
	fmt.Printf("STARTING FILE DOWNLOAD. Filename : %s. Peer : %s \n", filename, peer)
	if file != nil && file.Completed {
		// already have file
		fmt.Print("File already downloaded")
		return file
	}
	if file == nil {
		file = &storage.File{Name: filename, MetafileHash: metahash, ChunkCount: 0}
	}
	go func() {
		//get or request metafile
		meta := gsp.FileStorage.GetMetafile(metahash)
		if meta == nil {
			fmt.Printf("DOWNLOADING metafile of %s from %s\n", filename, peer)
			meta, err := gsp.downloadFromPeer(metahash, peer)
			if err != nil {
				log.Printf("ERROR DOWNLOADING METAFILE FOR FILE %s FROM PEER %s", filename, peer)
				return
			}
			gsp.FileStorage.StoreMetafile(metahash, meta)
		}
		meta = gsp.FileStorage.GetMetafile(metahash)
		fmt.Printf("META %x \n", meta)
		toDownload := len(meta) / sha256.Size //number of chunks to download
		fmt.Printf("TO DOWN : %d \n", toDownload)

		// create the slice of all hashes
		var chunksHash []utils.SHA256
		for i := 0; i < toDownload; i++ {
			j := i * sha256.Size
			k := j + sha256.Size
			var hash utils.SHA256
			copy(hash[:], meta[j:k])
			chunksHash = append(chunksHash, hash)
		}
		fmt.Printf("%x \n", chunksHash)
		// download all the chunks
		for file.ChunkCount < uint32(toDownload) {
			h := chunksHash[file.ChunkCount]
			fmt.Printf("DOWNLOADING %s chunk %d from %s \n", filename, file.ChunkCount+1, peer)
			data, err := gsp.downloadFromPeer(h, peer)
			if err != nil {
				//ABORTING
				log.Print(err)
				file.Completed = false
				return
			}
			chunk := &storage.Chunk{Data: data, Hash: h}
			gsp.FileStorage.StoreChunk(chunk)
			file.ChunkCount++
		}

		out, err := os.Create(FileOutDirectory + filename)
		if err != nil {
			fmt.Println("Impossible to create a new file \n", err)
			return
		}
		//have all the chunks. Reconstructing the file
		gsp.FileStorage.WriteChunksToFile(chunksHash, out)
		file.Completed = true
		fmt.Printf("RECONSTRUCTED file %s \n", filename)
	}()
	return file
}

func (gsp *Gossiper) downloadFromPeer(hash utils.SHA256, peer string) ([]byte, error) {
	tries := 0
	for tries <= maxChunkDownloadTries {
		tries++
		callback := gsp.WaitingForData.RegisterFileObserver(hash)
		// fmt.Printf("REGISTERING OBSERVER %x \n", hash)
		dr := message.NewDataRequest(gsp.Name, peer, 0, hash)
		gsp.forwardDataRequest(dr)
		reply := <-callback
		gsp.WaitingForData.UnregisterFileObserver(hash)
		data := reply.Data
		h := sha256.Sum256(data)
		// fmt.Printf("RECEIVED PKT HASH : %x \n", h)
		// fmt.Printf("RECEIVED PKT data : %x \n", data)
		if bytes.Compare(h[:], reply.HashValue) == 0 {
			return data, nil
		}
	}
	err := fmt.Errorf("ERROR DOWNLOADING CHUNK %x FROM PEER %s : MAX RETRIES LIMIT REACHED. ABORTING", hash, peer)
	return nil, err
}
func (gsp *Gossiper) processDataRequest(dr *message.DataRequest) {
	fmt.Printf("RECEIVED DATA REQUEST FROM %s FOR hash %x \n", dr.Origin, dr.HashValue)
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
		fmt.Println(len(data))
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
	hash := utils.SliceToHash(r.HashValue)
	// fmt.Printf("GETTING OBSERVER %x \n", hash)
	err := gsp.WaitingForData.SendDataToObserver(hash, r)
	if err != nil {
		log.Print(err)
	}
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
	fmt.Printf("SENDING DATA REPLY TO DEST %s VIA %s \n", r.Destination, nextHopAddr)
	if nextHopAddr != "" {
		gsp.send(gp, nextHopAddr)
	}
}
