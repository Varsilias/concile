package pkg

import "time"

type Stats struct {
	Processed  int
	Failed     int
	Duplicates int
	StartedAt  time.Time
	EndedAt    time.Time
}
