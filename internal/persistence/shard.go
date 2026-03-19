package persistence

import (
	"context"
	"sync"
	"time"
)

const QueueSize = 4 * 1024

// Shard is dedicated to write to a specific WAL Log file
type Shard struct {
	queue chan uint64
	mu    sync.RWMutex
	wal   *WAL
	seen  map[uint64]struct{}
}

func NewShard(idx int) (*Shard, error) {

	seen := map[uint64]struct{}{}
	queue := make(chan uint64, QueueSize)
	wal, err := NewWAL(idx)
	if err != nil {
		return nil, err
	}

	s := &Shard{
		queue: queue,
		seen:  seen,
		wal:   wal,
	}
	return s, nil

}

func (s *Shard) Append(key uint64) {
	s.queue <- key // we do not want to append to WAL file for every call
}

func (s *Shard) Check(key uint64) bool {
	s.mu.RLock()
	_, ok := s.seen[key]
	s.mu.RUnlock()
	return ok
}

func (s *Shard) writer(ctx context.Context) {
	var batch []uint64
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case item, ok := <-s.queue:
			if !ok {
				// The channel is closed and empty.
				// Final flush and exit the goroutine.
				if len(batch) > 0 {
					s.flush(batch)
				}
				return
			}
			batch = append(batch, item)
			if len(batch) >= QueueSize {
				s.flush(batch)
				batch = batch[:0] // reset batch to zero
			}
		case <-ticker.C:
			if len(batch) > 0 {
				s.flush(batch)
				batch = batch[:0] // reset batch to zero
			}
		case <-ctx.Done():
			if len(batch) > 0 {
				s.flush(batch)
			}
			return
		}
	}
}

func (s *Shard) flush(batch []uint64) error {
	s.wal.WriteBatch(batch) // write batch to WAL file without lock

	s.mu.Lock()
	for _, b := range batch {
		s.seen[b] = struct{}{}
	}
	s.mu.Unlock()

	return nil
}
