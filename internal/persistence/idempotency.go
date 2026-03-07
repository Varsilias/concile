package persistence

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
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
	seen map[string]struct{}
	wal  *WAL
}

func NewMemoryStore(enableWAL bool) (IdempotencyStore, error) {
	tl := telemetry.NewReplayStats()
	defer tl.Finish()
	defer telemetry.Track("WAL Replay")()

	seen := map[string]struct{}{}
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
	_, ok := ms.seen[key]
	return ok
}

// Record writes to durable log and then updates in-memory map
func (ms *MemoryStore) Record(key string) error {
	if ms.wal != nil { // check this because the WAL may not be enabled
		err := ms.wal.Append(key)
		if err != nil {
			return err
		}
	}
	ms.seen[key] = struct{}{}
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
		buffer := bufio.NewReaderSize(f, 1<<20)
		for {
			key, err := buffer.ReadSlice('\n')

			if len(key) > 0 {
				line := bytes.TrimRight(key, "\r\n")
				if len(line) > 0 {
					ms.seen[string(line)] = struct{}{}

				}
			}
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				f.Close()
				return err
			}
		}
		f.Close()
	}

	return nil
}
