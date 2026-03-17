package processor

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/Varsilias/concile/internal/command"
	"github.com/Varsilias/concile/internal/persistence"
	"github.com/Varsilias/concile/internal/pkg"
	"github.com/Varsilias/concile/internal/telemetry"
	"github.com/Varsilias/concile/internal/utils"
)

func init() {
	command.Register("ingest", "Ingest Transaction records",
		func(fs *flag.FlagSet, values map[string]*string) {
			values["file"] = fs.String("file", "", "JSONL file path")
			values["provider"] = fs.String("provider", "", "case insensitive name of the partner bank(Vbank, Globus, Providus)")
			values["enable-wal"] = fs.String("enable-wal", "true", "enable WAL for idempotency check")
			values["workers"] = fs.String("workers", "", "Number of concurrent processes to spin up to enable parallel processing")
		},
		func(args []string, values map[string]*string) error {
			path := *values["file"]
			if path == "" {
				return fmt.Errorf("file path is required (use --file /path/to/file)")
			}
			provider := *values["provider"]
			if provider == "" {
				return fmt.Errorf("provider not specified e.g Vbank, Globus, Providus")
			}
			enableWal := *values["enable-wal"]
			if enableWal == "" {
				return fmt.Errorf(`enable-wal can be either "true" or "false"`)
			}
			walEnabled, err := strconv.ParseBool(enableWal)
			if err != nil {
				return fmt.Errorf(`invalid value provided, enable-wal can be either "true" or "false"`)
			}
			workers := runtime.NumCPU()
			workerVal := *values["workers"]

			if workerVal != "" {
				workers, err = strconv.Atoi(workerVal)
				if err != nil {
					return fmt.Errorf("invalid value provided for worker count, --worker should be set to number >= 1")
				}
			} else {
				fmt.Println("worker count not set, defaulting to total number of CPU cores present", runtime.NumCPU())
			}

			return Run(path, provider, walEnabled, workers)
		},
	)
}

func Run(filePath, partner string, enableWAL bool, workers int) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	stats := telemetry.New()
	defer stats.Finish()
	defer telemetry.Track("Transaction Processor")()

	store, err := persistence.NewMemoryStore(ctx, enableWAL)

	done := make(chan struct{})

	var wg sync.WaitGroup
	jobs := make(chan []byte, workers*1000)
	for range workers {
		wg.Go(func() {
			worker(jobs, store, stats, partner) // backpressure is handled automatically in Golang due to the fact that channels also have built-in "Blocking mechanism"
		})
	}

	go func() {
		ticker := time.NewTicker(5 * time.Second) // reduce logging frequency for now, in future we can add support for verbose logging probably make it a global config
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Printf("Current Backlog: %d/%d", len(jobs), cap(jobs))
			case <-ctx.Done():
				log.Printf("interrupt recieved, stopping monitor...")
			case <-done:
				return
			}
		}

	}()

	path, err := utils.ResolvePath(filePath) // we already handled empty filepath check
	if err != nil {
		return fmt.Errorf("error resolving file path: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	buffer := bufio.NewReaderSize(f, 1<<20)
	size := 0

	for {
		select {
		case <-ctx.Done():
			goto SHUTDOWN
		default:
			line, err := buffer.ReadBytes('\n') // Given our file structure and each line size, we will never hit [bufio.ErrBufferFull] error
			if len(line) > 0 {
				size += len(line)
				jobs <- line // send clean ready to be processed data to channel
			}

			if errors.Is(err, io.EOF) { // means we have reached the end out the file
				goto SHUTDOWN
			}
			if err != nil {
				return fmt.Errorf("corrupted file content: %v", err)
			}
		}
	}

SHUTDOWN:
	close(jobs)
	wg.Wait()
	close(done)
	fmt.Printf("Processed %s of data\n", utils.Bytes(size))
	return nil
}

func worker(jobs <-chan []byte, store persistence.IdempotencyStore, stats *telemetry.IngestionStats, partner string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("worker panic: %v", r)
		}
	}()

	for line := range jobs {
		trx, recErr := reconcile(line, partner)
		if recErr != nil {
			log.Printf("Failed processing line, reason %v", recErr)
			stats.IncrFailed()
		} else if key := pkg.IdempotencyKey(trx); store.Seen(key) {
			stats.IncrDuplicates()
		} else {
			key := pkg.IdempotencyKey(trx)
			store.Record(key) // write to WAL and updates in-memory map
			stats.IncrProcessed()
		}

	}
}

func reconcile(line []byte, sourceBank string) (pkg.CanonicalTransaction, error) {
	var cnTrxEmpty pkg.CanonicalTransaction

	var rawTrx pkg.RawTransaction
	if err := json.Unmarshal(line, &rawTrx); err != nil {
		return cnTrxEmpty, err
	}

	cnTrx, err := pkg.Normalize(rawTrx, sourceBank)
	if err != nil {
		return cnTrxEmpty, err
	}

	return cnTrx, nil
}
