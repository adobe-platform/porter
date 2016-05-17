package cli

import (
	"errors"
	"flag"
	"fmt"
	"github.com/armon/go-radix"
	"io"
	"math"
	"os"
	"regexp"
)

type (
	Command interface {
		// The name of the command
		Name() string

		// A one-line description of this command
		ShortHelp() string

		// A multi-line description of this command.
		//
		// Its subcommands' ShortHelp message will also be printed.
		LongHelp() string

		// Execute executes with the remaining passed in arguments.
		//
		// Return false if the command can't execute which will display the
		// command's LongHelp message
		Execute([]string) bool

		// Any sub commands this command is capable of
		SubCommands() []Command
	}

	Driver struct {
		// This is exposed so commands creating flag.FlagSet can use the setting
		// on the driver for consistency
		ErrorHandling flag.ErrorHandling

		// os.Args
		args []string

		// to communicate out we only need a writer so there's no need to couple
		// simple communication with a *os.File
		stdout io.Writer

		/* command-related fields */
		tree *radix.Tree
	}

	commandNode struct {
		command           Command
		longestSubCommand float64
	}
)

var newlineRE = regexp.MustCompile(`\n`)

func New(errorHandling flag.ErrorHandling) *Driver {
	return NewWithEnv(errorHandling, nil, nil)
}

// NewWithEnv inverts control of the outside world and enables testing
func NewWithEnv(errorHandling flag.ErrorHandling, args []string, stdout io.Writer) *Driver {
	if args == nil {
		args = os.Args
	}

	if stdout == nil {
		stdout = os.Stdout
	}

	return &Driver{
		ErrorHandling: errorHandling,
		args:          args,
		stdout:        stdout,
	}
}

func (d *Driver) ParseInput() error {
	var (
		node   commandNode
		iface  interface{}
		exists bool
		ok     bool
	)

	if d.tree == nil {
		return errors.New("root command doesn't exist. call RegisterRoot first")
	}

	iface, exists = d.tree.Get("")
	if !exists {
		return errors.New("tree exists without a root")
	}

	node, ok = iface.(commandNode)
	if !ok {
		return errors.New("node is not a commandNode")
	}

	i := 1 // 0 is the program name (similar to ARGV)
	path := ""
	for ; i < len(d.args); i++ {

		// fmt.Fprintf(d.stdout, "arg %d %s\n", i, d.args[i])
		path = path + "/" + d.args[i]
		if subCmd, exists := d.tree.Get(path); exists {
			node, ok = subCmd.(commandNode)
			if !ok {
				return fmt.Errorf("node at path [%s] is not a commandNode", path)
			}
		} else {
			break
		}
	}

	cmd := node.command
	if !cmd.Execute(d.args[i:]) {

		fmt.Fprintln(d.stdout, cmd.LongHelp())

		subCmds := cmd.SubCommands()

		if len(subCmds) > 0 {

			fmt.Fprintln(d.stdout)
			fmt.Fprintln(d.stdout, "Commands:")

			// create format string with correct padding to accommodate
			// the longest command name.
			//
			// e.g. "    %-42s - %s\n" if 42 is the longest
			fmtStr := fmt.Sprintf("    %%-%.fs - %%s\n", node.longestSubCommand)

			for _, subCmd := range subCmds {
				cmdName := subCmd.Name()

				shortHelp := newlineRE.ReplaceAllString(subCmd.ShortHelp(), "")
				fmt.Fprintf(d.stdout, fmtStr, cmdName, shortHelp)
			}
		}

		switch d.ErrorHandling {
		case flag.ContinueOnError:
			// nothing to do
		case flag.ExitOnError:
			// same as an unsuccessful flag.Parse()
			// and http://tldp.org/LDP/abs/html/exitcodes.html
			os.Exit(2)
		case flag.PanicOnError:
			panic("invalid call to command at path " + path)
		}
	}

	return nil
}

func (d *Driver) RegisterRoot(newRoot Command) error {
	if d.tree != nil {
		return errors.New("RegisterRoot already called")
	}

	if newRoot == nil {
		return errors.New("root command is nil")
	}

	if newRoot.Name() != "" {
		return errors.New("root command name must be \"\"")
	}

	d.tree = radix.New()

	return d.registerCmd("", newRoot, nil)
}

func (d *Driver) registerCmd(path string, cmd Command, maxLen *float64) error {
	if cmd == nil {
		return nil
	}

	cmdName := cmd.Name()
	path = path + cmdName

	if maxLen != nil {
		*maxLen = math.Max(*maxLen, float64(len(cmdName)))
	}

	if _, exists := d.tree.Get(path); exists {
		return fmt.Errorf("command path %s already exists", path)
	}

	longestSub := new(float64)

	subCmds := cmd.SubCommands()
	if subCmds != nil {

		for _, subCmd := range subCmds {

			err := d.registerCmd(path+"/", subCmd, longestSub)
			if err != nil {
				return err
			}
		}
	}

	node := commandNode{
		command:           cmd,
		longestSubCommand: *longestSub,
	}

	d.tree.Insert(path, node)

	return nil
}
