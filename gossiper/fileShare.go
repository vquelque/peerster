package gossiper

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/vquelque/Peerster/storage"
)

const ChunkSize = 8000 //in bytes
const FileTempDirectory = "./_sharedFiles/"

type DataRequest struct {
	Origin      string
	Destination string
	HopLimit    uint32
	HashValue   []byte //hash of chunk or metafile if file request
}

type DataReply struct {
	Origin      string
	Destination string
	HopLimit    uint32
	HashValue   []byte
	Data        []byte
}

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
