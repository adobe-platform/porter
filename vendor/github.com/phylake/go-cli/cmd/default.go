package cmd

import "github.com/phylake/go-cli"

// Default is the default implementation of cli.Command.
//
// It simply returns struct fields to satisfy the interface.
type Default struct {
	NameStr        string
	ShortHelpStr   string
	LongHelpStr    string
	ExecuteFunc    func([]string) bool
	SubCommandList []cli.Command
}

func (cmd *Default) Name() string {
	return cmd.NameStr
}

func (cmd *Default) ShortHelp() string {
	return cmd.ShortHelpStr
}

func (cmd *Default) LongHelp() string {
	return cmd.LongHelpStr
}

func (cmd *Default) Execute(args []string) bool {
	if cmd.ExecuteFunc != nil {
		return cmd.ExecuteFunc(args)
	}
	return false
}

func (cmd *Default) SubCommands() []cli.Command {
	return cmd.SubCommandList
}
