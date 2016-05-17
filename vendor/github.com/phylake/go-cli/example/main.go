package main

import (
	"flag"
	"fmt"

	"github.com/phylake/go-cli"
	"github.com/phylake/go-cli/cmd"
)

// Run `go run main.go`, `go run main.go punch`, `go run main.go punch -combo Shoryuken!`, etc.
func main() {
	driver := cli.New(flag.ContinueOnError)

	rootCmd := &cmd.Root{
		Help: `Usage: ryu COMMAND [args]

A madeup CLI to demonstrate this framework.

Ryu is a character from Street Fighter II`,
		SubCommandList: []cli.Command{

			&PunchCmd{},

			// A simple command so you don't have to implement all the methods
			// of cli.Command
			&cmd.Default{
				NameStr:      "version",
				ShortHelpStr: "Print out a version string",
				ExecuteFunc: func(args []string) bool {
					fmt.Println("v0.0.1")

					// since this command doesn't take any arguments it doesn't
					// need a LongHelp() since we always return a successful
					// execution
					return true
				},
			},
		},
	}

	if err := driver.RegisterRoot(rootCmd); err != nil {
		panic(err)
	}

	if err := driver.ParseInput(); err != nil {
		panic(err)
	}
}

// PunchCmd implements cli.Command
type PunchCmd struct{}

func (cmd *PunchCmd) Name() string {
	return "punch"
}

func (cmd *PunchCmd) ShortHelp() string {
	return "Punch your shell"
}

func (cmd *PunchCmd) LongHelp() string {
	return `NAME
	punch - Punch your shell

SYNOPSIS
	punch -combo <combo name>

DESCRIPTION
	punch is a fake command illustrating how to build a nested command CLI
	including parsing arguments and printing out this help text when a command
	is invoked incorrectly`
}

// Return false if this command wasn't correctly invoked and LongHelp() will be
// printed out
func (cmd *PunchCmd) Execute(args []string) bool {
	if len(args) > 0 {

		var combo string

		flagSet := flag.NewFlagSet("", flag.ContinueOnError)
		flagSet.Usage = func() {
			fmt.Println(cmd.LongHelp())
		}

		// don't use flag description since we're setting Usage
		flagSet.StringVar(&combo, "combo", "", "")

		// Parse will succeed or os.Exit
		flagSet.Parse(args)

		fmt.Println("You executed punch combo " + combo)

		return true
	}
	return false
}

func (cmd *PunchCmd) SubCommands() []cli.Command {
	return nil
}
