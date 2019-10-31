package storage

import (
	"log"
	"sync"

	"github.com/vquelque/Peerster/utils"
)

type File struct {
	Name         string
	MetafileHash utils.SHA256
}

type Chunk struct {
	Data []byte
	Hash utils.SHA256
}

type FileStorage struct {
	files         map[utils.SHA256]*File //metahash -> File
	filesLock     *sync.RWMutex
	chunks        map[utils.SHA256]*Chunk //chunk hash -> chunk
	chunksLock    *sync.RWMutex
	metafiles     map[utils.SHA256][]byte //metafile hash -> metafile
	metafilesLock *sync.RWMutex
}

func NewFileStorage() *FileStorage {
	return &FileStorage{
		files:         make(map[utils.SHA256]*File),
		filesLock:     &sync.RWMutex{},
		chunks:        make(map[utils.SHA256]*Chunk),
		chunksLock:    &sync.RWMutex{},
		metafiles:     make(map[utils.SHA256][]byte),
		metafilesLock: &sync.RWMutex{},
	}
}

//StoreFile stores file and associated metafile
func (fs *FileStorage) StoreFile(f *File, metafile []byte) {
	fs.filesLock.Lock()
	fs.metafilesLock.Lock()
	defer fs.filesLock.Unlock()
	defer fs.metafilesLock.Unlock()
	fs.files[f.MetafileHash] = f
	fs.metafiles[f.MetafileHash] = metafile
}

// StoreChunk stores chunnk
func (fs *FileStorage) StoreChunk(c *Chunk) {
	fs.chunksLock.Lock()
	defer fs.chunksLock.Unlock()
	fs.chunks[c.Hash] = c
}

func (fs *FileStorage) GetChunkOrMeta(hash utils.SHA256) []byte {
	fs.chunksLock.RLock()
	fs.filesLock.RLock()
	defer fs.chunksLock.Unlock()
	defer fs.filesLock.RUnlock()
	meta, mfound := fs.metafiles[hash]
	chunk, cfound := fs.chunks[hash]
	if mfound {
		return meta
	} else if cfound {
		return chunk.Data
	} else if cfound && mfound {
		log.Print("Hash problem : meta and chunk found for this hash")
	}
	return nil
}
