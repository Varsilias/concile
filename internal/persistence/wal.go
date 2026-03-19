package persistence

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const DataDir = ".data"

type WAL struct {
	dataDir  string
	filename string
	writer   *bufio.Writer
	file     *os.File
	mu       sync.Mutex
	buf      [8]byte
}

func NewWAL(idx int) (*WAL, error) {
	err := os.MkdirAll(DataDir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	wal := &WAL{
		dataDir:  DataDir,
		filename: fmt.Sprintf("wal_%d.log", idx),
	}
	if err := wal.CreateLogFile(); err != nil {
		return nil, err
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
	wal.writer = bufio.NewWriterSize(f, 64*1024)
	return nil
}

func (wal *WAL) Append(keyHash uint64) error {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	binary.BigEndian.PutUint64(wal.buf[:], keyHash) // encode the hash into a compact 8 byte for easy storage

	_, err := wal.file.Write(wal.buf[:]) // TODO: handle error if necessary
	return err
}

func (wal *WAL) WriteBatch(batch []uint64) error {
	for _, b := range batch {
		binary.BigEndian.PutUint64(wal.buf[:], b)
		if _, err := wal.writer.Write(wal.buf[:]); err != nil {
			return err
		}
	}

	// flush the internal 64KB buffer
	if err := wal.writer.Flush(); err != nil {
		return err
	}

	if err := wal.file.Sync(); err != nil {
		return err
	}
	return nil
}

func (wal *WAL) Flush() error {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	return wal.file.Close()
}
