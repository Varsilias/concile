package processor

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

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
			return Run(path, provider, walEnabled)
		},
	)
}

func Run(filePath, partner string, enableWAL bool) error {
	stats := telemetry.New()
	defer stats.Finish()
	defer telemetry.Track("Transaction Processor")()

	store, err := persistence.NewMemoryStore(enableWAL)

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
	lineNumber := 0

	for {
		line, err := buffer.ReadSlice('\n') // Given our file structure and each line size, we will never hit [bufio.ErrBufferFull] error
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("corrupted file content: %v", err)
		}

		if len(line) > 0 {
			lineNumber++
			size += len(line)
			trx, recErr := reconcile(line, partner)
			key := pkg.IdempotencyKey(trx)
			seen := store.Seen(key)
			if recErr != nil {
				log.Printf("Failed processing line %d\n, reason %v", lineNumber, recErr)
				stats.IncrFailed()
			} else if seen {
				stats.IncrDuplicates()
			} else {
				store.Record(key) // write to WAL and updates in-memory map
				// TODO: Add actual processing logic
				stats.IncrProcessed()
			}
		}

		if errors.Is(err, io.EOF) { // means we have reached the end out the file
			break
		}
	}

	fmt.Printf("Processed %s of data\n", utils.Bytes(size))

	return nil
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
