package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Varsilias/concile/internal/command"
	_ "github.com/Varsilias/concile/internal/hook" // The loader
)

func main() {
	command.Register("help", "Show CLI usage",
		func(fs *flag.FlagSet, values map[string]*string) {
			values["file"] = fs.String("file", "", "XLSX file path")
		},
		func(args []string, values map[string]*string) error {
			printGlobalUsage()
			return nil
		})

	if len(os.Args) < 2 {
		printGlobalUsage()
		return
	}

	cmdName := os.Args[1]
	cmd, exists := command.Registry[cmdName]
	if !exists {
		fmt.Printf("Unknown command %s\n", cmdName)
		printGlobalUsage()
		os.Exit(1)
	}

	cmd.FlagSet.Parse(os.Args[2:])

	args := cmd.FlagSet.Args()
	if err := cmd.Action(args, cmd.Values); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		printGlobalUsage()
		os.Exit(1)
	}
}

func printGlobalUsage() {
	fmt.Printf("Concile: CLI tool to perform transaction reconciliation\n\n")

	fmt.Printf("Usage:   %s <COMMAND> [OPTIONS] [ARGUMENTS]\n\n", os.Args[0])

	fmt.Println("Commands:")

	for name, cmd := range command.Registry {
		fmt.Printf("  %-10s %s\n", name, cmd.Description)
	}
}
