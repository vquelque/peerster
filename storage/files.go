package storage

import (
	"crypto/sha256"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/vquelque/Peerster/utils"
)

type File struct {
	Name         string
	MetafileHash utils.SHA256
	ChunkCount   uint64
	Completed    bool
}

type Metafile []byte

type Chunk struct {
	Data []byte
	Hash utils.SHA256
}

type FileStorage struct {
	files     map[utils.SHA256]*File    //metahash -> File
	chunks    map[utils.SHA256]*Chunk   //chunk hash -> chunk
	metafiles map[utils.SHA256]Metafile //metafile hash -> metafile
	lock      sync.RWMutex
}

func NewFileStorage() *FileStorage {
	return &FileStorage{
		files:     make(map[utils.SHA256]*File),
		chunks:    make(map[utils.SHA256]*Chunk),
		metafiles: make(map[utils.SHA256]Metafile),
		lock:      sync.RWMutex{},
	}
}

//StoreFile stores file and associated metafile
func (fs *FileStorage) StoreFile(f *File, metafile []byte) {
	fs.lock.Lock()
	defer fs.lock.Unlock()
	fs.files[f.MetafileHash] = f
	fs.metafiles[f.MetafileHash] = metafile
}

// StoreChunk stores chunnk
func (fs *FileStorage) StoreChunk(c *Chunk) {
	fs.lock.Lock()
	defer fs.lock.Unlock()
	fs.chunks[c.Hash] = c
}

func (fs *FileStorage) GetChunkOrMeta(hash utils.SHA256) []byte {
	fs.lock.RLock()
	defer fs.lock.RUnlock()
	meta, mfound := fs.metafiles[hash]
	chunk, cfound := fs.chunks[hash]
	//	log.Printf("GETTING CHUNK %x  FOUND %s ", hash, cfound)
	if mfound {
		return meta
	} else if cfound {
		return chunk.Data
	} else if cfound && mfound {
		log.Print("Hash problem : meta and chunk found for this hash")
	}
	return nil
}

func (fs *FileStorage) GetFile(hash utils.SHA256) *File {
	fs.lock.RLock()
	defer fs.lock.RUnlock()
	f, found := fs.files[hash]
	if !found {
		return nil
	}
	return f
}

func (fs *FileStorage) GetMetafile(hash utils.SHA256) Metafile {
	fs.lock.RLock()
	defer fs.lock.RUnlock()
	f, found := fs.metafiles[hash]
	if !found {
		return nil
	}
	return f
}

func (fs *FileStorage) StoreMetafile(metahash utils.SHA256, meta Metafile) {
	fs.lock.Lock()
	defer fs.lock.Unlock()
	_, found := fs.metafiles[metahash]
	if !found {
		fs.metafiles[metahash] = make(Metafile, len(meta))
		copy(fs.metafiles[metahash], meta)
	}
}

func (fs *FileStorage) WriteChunksToFile(chunks []utils.SHA256, file *os.File) {
	fs.lock.Lock()
	defer fs.lock.Unlock()
	defer file.Close()
	for _, h := range chunks {
		data := fs.chunks[h]
		file.Write(data.Data)
	}
}

func (fs *FileStorage) SearchForFile(keyword string) []*File {
	fs.lock.RLock()
	defer fs.lock.RUnlock()
	matchingFiles := make([]*File, 0)
	for _, f := range fs.files {
		if strings.Contains(f.Name, keyword) {
			matchingFiles = append(matchingFiles, f)
		}
	}
	return matchingFiles
}

func (fs *FileStorage) ChunkCount(metahash utils.SHA256) uint64 {
	fs.lock.RLock()
	defer fs.lock.RUnlock()
	meta := fs.metafiles[metahash]
	count := uint64(len(meta) / sha256.Size)
	return count
}
