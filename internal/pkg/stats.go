package pkg

import "time"

type IngestionStats struct {
	Processed  int
	Failed     int
	Duplicates int
	StartedAt  time.Time
	EndedAt    time.Time
}
