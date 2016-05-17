package cmd

import "github.com/phylake/go-cli"

// Root implements cli.Command.
//
// It represents the command to your CLI with no arguments and has only a help
// string and a list of sub commands.
type Root struct {
	Help           string
	SubCommandList []cli.Command
}

func (cmd *Root) Name() string {
	return ""
}

// There's no short help for the root command
func (cmd *Root) ShortHelp() string {
	return ""
}

func (cmd *Root) LongHelp() string {
	return cmd.Help
}

// The root command isn't run because a call to the CLI with no arguments should
// print out the Help string
func (cmd *Root) Execute([]string) bool {
	return false
}

func (cmd *Root) SubCommands() []cli.Command {
	return cmd.SubCommandList
}
