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

	"github.com/Varsilias/concile/internal/telemetry"
)

const ShardCount = 256

type IdempotencyStore interface {
	Seen(key string) bool
	Record(key string) error
	Close()
}

// MemoryStore holds the keys that have been
// processed so far in the ingested file
type MemoryStore struct {
	shards []*Shard
	mask   uint64
	wg     sync.WaitGroup
}

func NewMemoryStore(ctx context.Context, enableWAL bool) (IdempotencyStore, error) {
	rtl := telemetry.NewReplayStats()
	defer rtl.Finish()
	defer telemetry.Track("WAL Replay")()

	store := &MemoryStore{}
	shards := make([]*Shard, 0, ShardCount)

	for i := range ShardCount {
		sh, err := NewShard(i)
		if err != nil {
			return nil, err
		}
		shards = append(shards, sh)
	}
	store.shards = shards
	store.mask = ShardCount - 1
	store.wg = sync.WaitGroup{}

	if err := store.rebuild(); err != nil {
		return nil, err
	}

	for _, shard := range store.shards {
		store.wg.Go(func() {
			shard.writer(ctx)
		})
	}

	return store, nil
}

// GetShard returns a shard based on the key
// we use the last 8-bits(1-byte) to determine which shard
// a key should fall into. Bitmasking is used for performance reasons
// Other implementation would be a modulo(%) operation
func (ms *MemoryStore) GetShard(key string) *Shard {
	hashSum := ms.hashKeyToUint64(key)
	shardIdx := hashSum & ms.mask // A bitmask of 255 (binary 11111111) extracts the last 8 bits
	return ms.shards[shardIdx]
}

// Seen guarantees that a key will always be found in the same Shard
// where it was initially added provided that the ShardCount does not change
func (ms *MemoryStore) Seen(key string) bool {
	shard := ms.GetShard(key)
	hashSum := ms.hashKeyToUint64(key)
	return shard.Check(hashSum)
}

// Record writes to durable log and then updates in-memory map
func (ms *MemoryStore) Record(key string) error {
	shard := ms.GetShard(key)
	hashSum := ms.hashKeyToUint64(key)
	shard.Append(hashSum)
	return nil
}

func (ms *MemoryStore) Close() {
	for _, shards := range ms.shards {
		// Closing the channel here is safe because by the time
		// Close() is called, the upstream workers have finished.
		close(shards.queue)
	}
	ms.wg.Wait()
	log.Println("All shards flushed and closed successfully.")
}

func (ms *MemoryStore) rebuild() error {
	dataDir := DataDir
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
		f, err := os.OpenFile(fullPath, os.O_RDWR, 0644) // open file in read-write mode, we need WRITE permission for trincating(if needed)
		if err != nil {
			return err
		}
		var validOffset int64 // accumulator for tracking valid offsets we read
		reader := bufio.NewReaderSize(f, 1<<20)
		buffer := make([]byte, 8)
		for {
			n, err := io.ReadFull(reader, buffer)

			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				// if we encounter an invalid or corrupted data in the current file,
				// we truncate the file to the last valid offset and move on to processing the next file
				// a data is considered corrupted if a block does not fit the buffer, io.ErrUnexpectedEOF error is thrown
				if errors.Is(err, io.ErrUnexpectedEOF) {
					log.Printf("Corruption found, Truncating to %d\n", validOffset)
					// truncate WAL file to last valid entry
					if err := f.Truncate(validOffset); err != nil {
						return err
					}
					break
				}
				f.Close()
				return err
			}
			validOffset += int64(n) //
			id := binary.BigEndian.Uint64(buffer)
			shardIdx := id & ms.mask
			shard := ms.shards[shardIdx]
			shard.mu.Lock()
			shard.seen[id] = struct{}{}
			shard.mu.Unlock()

		}
		f.Close()
	}

	return nil
}

// hashKeyToUint64 produces an unsigned 64-bit compatible hash from a given string key
func (ms *MemoryStore) hashKeyToUint64(key string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(key))
	return h.Sum64()
}
