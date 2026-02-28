package telemetry

import (
	"fmt"
	"time"
)

func Track(name string) func() {
	start := time.Now()

	return func() {
		elapsed := time.Since(start)

		var output string
		if elapsed < time.Second {
			output = elapsed.String()
		} else {
			output = elapsed.Round(time.Millisecond * 10).String()
		}
		fmt.Printf("⏱️  %s took %s\n", name, output)
	}
}

func CalcTimeDiff(elapsed time.Duration) string {
	var output string
	if elapsed < time.Second {
		output = elapsed.String()
	} else {
		output = elapsed.Round(time.Millisecond * 10).String()
	}

	return fmt.Sprintf("%s\n", output)
}
