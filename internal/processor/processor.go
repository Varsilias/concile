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

	"github.com/Varsilias/concile/internal/command"
	"github.com/Varsilias/concile/internal/pkg"
	"github.com/Varsilias/concile/internal/telemetry"
	"github.com/Varsilias/concile/internal/utils"
)

func init() {
	command.Register("ingest", "Ingest Transaction records",
		func(fs *flag.FlagSet, values map[string]*string) {
			values["file"] = fs.String("file", "", "JSONL file path")
		},
		func(args []string, values map[string]*string) error {
			path := *values["file"]
			if path == "" {
				return fmt.Errorf("file path is required (use --file /path/to/file)")
			}
			return Run(path)
		},
	)
}

func Run(filePath string) error {
	var seen = map[string]struct{}{}

	stats := telemetry.New()
	defer stats.Finish()
	defer telemetry.Track("Transaction Processor")()
	path, err := utils.ResolvePath(filePath) // we already handled empty filepath check
	if err != nil {
		return fmt.Errorf("error resolving file path: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var store = make([]pkg.CanonicalTransaction, 0, 100)

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
			trx, isDup, recErr := reconcile(line, lineNumber, seen)
			if recErr != nil {
				log.Printf("Failed processing line %d\n, reason %v", lineNumber, recErr)
				stats.IncrFailed()
			} else if isDup {
				stats.IncrDuplicates()
			} else {
				store = append(store, trx)
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

func reconcile(line []byte, lineNumber int, seen map[string]struct{}) (pkg.CanonicalTransaction, bool, error) {
	var cnTrxEmpty pkg.CanonicalTransaction
	var rawTrx pkg.RawTransaction
	if err := json.Unmarshal(line, &rawTrx); err != nil {
		log.Printf("error processing line number %d\n", lineNumber)
		return cnTrxEmpty, false, err
	}

	// checking for duplicate
	ref := string(rawTrx.Reference)
	if _, ok := seen[ref]; ok {
		log.Printf("duplicate reference [%s] detected on line %d\n", rawTrx.Reference, lineNumber)
		return cnTrxEmpty, true, nil
	}
	seen[rawTrx.Reference] = struct{}{}

	cnTrx, err := pkg.Normalize(rawTrx)
	if err != nil {
		return cnTrxEmpty, false, err
	}

	return cnTrx, false, nil
}
