package telemetry

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Varsilias/concile/internal/pkg"
)

type IngestionStats struct {
	stats pkg.Stats
	mu    sync.Mutex
}

func New() *IngestionStats {
	return &IngestionStats{
		stats: pkg.Stats{
			StartedAt: time.Now(),
		},
	}
}

func (s *IngestionStats) GetCurrentStats() pkg.Stats {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.stats
}

func (s *IngestionStats) IncrFailed() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats.Failed++
}

func (s *IngestionStats) IncrDuplicates() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats.Duplicates++
}

func (s *IngestionStats) IncrProcessed() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats.Processed++
}

func (s *IngestionStats) Finish() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.EndedAt = time.Now()

	currentStats := s.stats

	elapsed := currentStats.EndedAt.Sub(currentStats.StartedAt)
	durationStr := CalcTimeDiff(elapsed)

	fmt.Println("\n" + strings.Repeat("=", 30))
	fmt.Println("       INGESTION REPORT       ")
	fmt.Println(strings.Repeat("=", 30))

	fmt.Printf("%-15s %d\n", "Processed:", currentStats.Processed)
	fmt.Printf("%-15s %d\n", "Failed:", currentStats.Failed)
	fmt.Printf("%-15s %d\n", "Duplicates:", currentStats.Duplicates)
	fmt.Printf("%-15s %s\n", "Duration:", durationStr)

	fmt.Println(strings.Repeat("=", 30))

}
