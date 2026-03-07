package telemetry

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Varsilias/concile/internal/pkg"
)

type ReplayStats struct {
	stats pkg.Stats
	mu    sync.Mutex
}

func NewReplayStats() *ReplayStats {
	tracker := &ReplayStats{
		stats: pkg.Stats{
			StartedAt: time.Now(),
		},
	}
	return tracker
}

func (rs *ReplayStats) Finish() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	rs.stats.EndedAt = time.Now()

	currentStats := rs.stats

	elapsed := currentStats.EndedAt.Sub(currentStats.StartedAt)
	durationStr := CalcTimeDiff(elapsed)

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("       WAL REPLAY REPORT       ")
	fmt.Println(strings.Repeat("=", 50))

	fmt.Printf("%-15s %s\n", "Started At:", currentStats.StartedAt.Format(timeFormatRFC3339Milli))
	fmt.Printf("%-15s %s\n", "Ended At:", currentStats.EndedAt.Format(timeFormatRFC3339Milli))
	fmt.Printf("%-15s %s\n", "Duration:", durationStr)

	fmt.Println(strings.Repeat("=", 50))
}
