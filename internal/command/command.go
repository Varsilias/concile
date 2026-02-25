package command

import (
	"flag"
)

type Command struct {
	Name        string
	Description string
	FlagSet     *flag.FlagSet
	// Action takes the "Arguments" (the non-flag leftovers)
	Action func(args []string, values map[string]*string) error
	// Values stores the flag results
	Values map[string]*string
}

var Registry = make(map[string]*Command)

// Register adds a command to our global router
func Register(name, desc string, setup func(fs *flag.FlagSet, values map[string]*string), action func([]string, map[string]*string) error) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	vals := make(map[string]*string)
	setup(fs, vals)

	Registry[name] = &Command{
		Name:        name,
		Description: desc,
		FlagSet:     fs,
		Action:      action,
		Values:      vals,
	}
}
