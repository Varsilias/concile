package persistence

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type WAL struct {
	dataDir  string
	filename string
	file     io.ReadWriteCloser
	mu       sync.Mutex
}

func NewWAL() (*WAL, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	dataDir := filepath.Join(wd, ".data")
	err = os.MkdirAll(dataDir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	wal := &WAL{
		dataDir:  dataDir,
		filename: "wal.log",
	}
	return wal, nil
}

func (wal *WAL) CreateLogFile() error {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	path := filepath.Join(wal.dataDir, wal.filename)

	flag := os.O_CREATE | os.O_APPEND | os.O_RDWR // we need read/write permission because, we need to append to log file and read when replaying the log

	f, err := os.OpenFile(path, flag, 0644) // 0644 because we want to file owner to be the only one writing and other can only read
	if err != nil {
		return fmt.Errorf("failed to open/create WAL: %w", err)
	}
	// No need to defer f.Close(), it will be called when we flush to disk
	wal.file = f
	return nil
}

func (wal *WAL) DataDir() string {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	return wal.dataDir
}

func (wal *WAL) Append(keyHash uint64) error {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, keyHash) // encode the hash into a compact 8 byte for easy storage

	_, err := wal.file.Write(buf) // TODO: handle error if necessary
	return err
}

func (wal *WAL) Flush() error {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	return wal.file.Close()
}
