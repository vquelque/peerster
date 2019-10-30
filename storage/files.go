package storage

import (
	"sync"
)

type SHA256 = [32]byte

type File struct {
	Name         string
	MetafileHash SHA256
}

type Chunk struct {
	Data []byte
	Hash SHA256
}

type FileStorage struct {
	files         map[SHA256]*File //metahash -> File
	filesLock     *sync.RWMutex
	chunks        map[SHA256]*Chunk //chunk hash -> chunk
	chunksLock    *sync.RWMutex
	metafiles     map[SHA256][]byte //metafile hash -> metafile
	metafilesLock *sync.RWMutex
}

func NewFileStorage() *FileStorage {
	return &FileStorage{
		files:         make(map[SHA256]*File),
		filesLock:     &sync.RWMutex{},
		chunks:        make(map[SHA256]*Chunk),
		chunksLock:    &sync.RWMutex{},
		metafiles:     make(map[SHA256][]byte),
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
