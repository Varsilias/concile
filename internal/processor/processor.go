package processor

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/Varsilias/concile/internal/command"
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
			return nil
		},
	)
}

func Run(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	buffer := bufio.NewReader(f)

	buffer.ReadString('\n')

	return nil
}
