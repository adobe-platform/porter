[![Build Status](https://travis-ci.org/phylake/go-cli.svg?branch=master)](https://travis-ci.org/phylake/go-cli) [![godoc reference](https://godoc.org/github.com/phylake/go-cli?status.png)](https://godoc.org/github.com/phylake/go-cli)

# go-cli

A minimalist framework for CLIs containing nested commands.

There's a surprising amount of entirely uninteresting code needed to create a
human and machine friendly CLI such as

- ensuring proper exit codes on invalid input
- calculating sub-command list padding
- parsing `os.Args` so each command is passed only the arguments scoped to it

[See the example](example/main.go)
