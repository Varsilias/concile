package persistence

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Varsilias/concile/internal/telemetry"
)

const QueueSize = 4 * 1024 // 4KB

type IdempotencyStore interface {
	Seen(key string) bool
	Record(key string) error
	Flush() error
}

// MemoryStore holds the keys that have been
// processed so far in the ingested file
type MemoryStore struct {
	seen  map[uint64]struct{}
	wal   *WAL
	mu    sync.Mutex
	queue chan uint64
}

func NewMemoryStore(ctx context.Context, enableWAL bool) (IdempotencyStore, error) {
	tl := telemetry.NewReplayStats()
	defer tl.Finish()
	defer telemetry.Track("WAL Replay")()

	seen := map[uint64]struct{}{}
	queue := make(chan uint64, QueueSize)
	store := &MemoryStore{}
	store.seen = seen
	store.queue = queue

	if enableWAL {
		wal, err := NewWAL()
		if err != nil {
			return nil, err
		}
		wal.CreateLogFile()
		store.wal = wal
		store.rebuild()
		go store.writer(ctx) // only enable if WAL is enabled

	}

	return store, nil
}

func (ms *MemoryStore) Seen(key string) bool {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	hashSum := ms.hashKeyToUint64(key)

	_, ok := ms.seen[hashSum]
	return ok
}

// Record writes to durable log and then updates in-memory map
func (ms *MemoryStore) Record(key string) error {
	hashSum := ms.hashKeyToUint64(key)
	ms.queue <- hashSum // add to WAL Queue

	return nil
}

func (ms *MemoryStore) Flush() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	close(ms.queue)

	if ms.wal == nil {
		return nil
	}
	return ms.wal.Flush()
}

func (ms *MemoryStore) rebuild() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	dataDir := ms.wal.DataDir()

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return fmt.Errorf("failed to read WAL dir: %w", err)
	}

	// Get all the files in the data directory, loop through them,
	// for each file, read its content and populate the "seen" map
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".log" {
			continue // skip directory, this will rarely be the case
		}

		fullPath := filepath.Join(dataDir, entry.Name())
		// assuming we are dealing with a log file
		f, err := os.Open(fullPath)
		if err != nil {
			return err
		}
		reader := bufio.NewReaderSize(f, 1<<20)
		buffer := make([]byte, 8)
		for {
			_, err := io.ReadFull(reader, buffer)

			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				f.Close()
				return err
			}
			id := binary.BigEndian.Uint64(buffer)
			ms.seen[id] = struct{}{}

		}
		f.Close()
	}

	return nil
}

func (ms *MemoryStore) writer(ctx context.Context) {
	var batch []uint64
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case item := <-ms.queue:
			batch = append(batch, item)
			if len(batch) >= QueueSize {
				ms.flush(batch)
				batch = batch[:0] // reset batch to zero
			}
		case <-ticker.C:
			if len(batch) > 0 {
				ms.flush(batch)
				batch = batch[:0] // reset batch to zero
			}
		case <-ctx.Done():
			if len(batch) > 0 {
				ms.flush(batch)
			}
			log.Println("WAL Writer shutting down...")
			return
		}
	}
}

func (ms *MemoryStore) flush(batch []uint64) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, b := range batch {
		err := ms.wal.Append(b)
		if err != nil {
			return err
		}
		ms.seen[b] = struct{}{}
	}
	return nil
}

// hashKeyToUint64 produces an unsigned 64-bit compatible hash from a given string key
func (ms *MemoryStore) hashKeyToUint64(key string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(key))
	return h.Sum64()
}
