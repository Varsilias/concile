package persistence

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"

	"github.com/Varsilias/concile/internal/telemetry"
)

type IdempotencyStore interface {
	Seen(key string) bool
	Record(key string) error
}

// MemoryStore holds the keys that have been
// processed so far in the ingested file
type MemoryStore struct {
	seen map[uint64]struct{}
	wal  *WAL
}

func NewMemoryStore(enableWAL bool) (IdempotencyStore, error) {
	tl := telemetry.NewReplayStats()
	defer tl.Finish()
	defer telemetry.Track("WAL Replay")()

	seen := map[uint64]struct{}{}
	store := &MemoryStore{}
	store.seen = seen

	if enableWAL {
		wal, err := NewWAL()
		if err != nil {
			return nil, err
		}
		wal.CreateLogFile()
		store.wal = wal
		store.rebuild()
	}

	return store, nil
}

func (ms *MemoryStore) Seen(key string) bool {
	var hashSum uint64

	if ms.wal != nil {
		hashSum = ms.wal.HashKeyToUint64(key)
	} else {
		h := fnv.New64a()
		h.Write([]byte(key))
		hashSum = h.Sum64()
	}

	_, ok := ms.seen[hashSum]
	return ok
}

// Record writes to durable log and then updates in-memory map
func (ms *MemoryStore) Record(key string) error {
	var hashSum uint64

	if ms.wal != nil { // check this because the WAL may not be enabled
		err := ms.wal.Append(key)
		if err != nil {
			return err
		}
		hashSum = ms.wal.HashKeyToUint64(key)
	} else {
		h := fnv.New64a()
		h.Write([]byte(key))
		hashSum = h.Sum64()
	}

	ms.seen[hashSum] = struct{}{}
	return nil
}

func (ms *MemoryStore) rebuild() error {

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
