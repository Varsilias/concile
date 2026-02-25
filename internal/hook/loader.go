package hook

import (
	// Blank imports trigger init() in each package
	_ "github.com/Varsilias/concile/internal/jsonl"
	_ "github.com/Varsilias/concile/internal/processor"
)

// This file exists solely to ensure all subcommands
// are registered in the command.Registry.
