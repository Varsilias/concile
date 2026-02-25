package jsonl

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Varsilias/concile/internal/command"
	"github.com/xuri/excelize/v2"
)

func init() {
	command.Register("convert", "Convert XLSX to JSONL",
		func(fs *flag.FlagSet, values map[string]*string) {
			values["file"] = fs.String("file", "", "XLSX file path")
		},
		func(args []string, values map[string]*string) error {
			path := *values["file"]
			if path == "" {
				return fmt.Errorf("file path is required (use --file /path/to/file)")
			}
			Run(path)
			return nil
		},
	)
}

func Run(filePath string) {
	wd, err := os.Getwd()
	directoryPath := filepath.Join(wd, "data")
	if err != nil {
		log.Fatal("Could not retrieve current working directory")
	}

	err = os.MkdirAll(directoryPath, os.ModePerm)
	if err != nil {
		log.Fatal("Could not create data directory")
	}

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		log.Fatalf("Open error: %v", err)
	}

	defer f.Close()

	// 2. Process each sheet into its own file
	for _, sheetName := range f.GetSheetMap() {
		// Clean the sheet name to make it a safe filename
		safeName := strings.ReplaceAll(strings.ToLower(sheetName), " ", "_")
		fileName := filepath.Join(directoryPath, fmt.Sprintf("%s.jsonl", safeName))

		outFile, err := os.Create(fileName)
		if err != nil {
			log.Printf("Could not create file %s: %v", fileName, err)
			continue
		}

		fmt.Printf("Processing [%s] -> %s\n", sheetName, fileName)

		encoder := json.NewEncoder(outFile)
		rows, err := f.Rows(sheetName)
		if err != nil {
			log.Printf("Error reading rows in %s: %v", sheetName, err)
			outFile.Close()
			continue
		}

		var headers []string
		isFirstRow := true

		for rows.Next() {
			row, _ := rows.Columns()
			if len(row) == 0 {
				continue
			}

			if isFirstRow {
				headers = row
				isFirstRow = false
				continue
			}

			// Map row to JSON
			lineMap := make(map[string]interface{})
			for i, val := range row {
				if i < len(headers) {
					lineMap[headers[i]] = val
				}
			}

			encoder.Encode(lineMap)
		}
		outFile.Close()
	}
}
