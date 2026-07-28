package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// Version is set at build time: -ldflags="-X main.Version=..."
var Version = "0.1.0"

// exitError carries a process exit code through Cobra's error return.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exitError
	if errors.As(err, &ee) {
		return ee.code
	}
	// Missing required flags / unknown commands → usage
	return 2
}

// newRootCmd builds the Cobra command tree.
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tfguard",
		Short: "Terraform plan risk and policy reviewer",
		Long:  "tfguard parses terraform plan JSON, scores change risk, and evaluates security policies.",
		// Running with no subcommand shows help and returns an error we map to exit 2.
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = cmd.Help()
			return &exitError{code: 2, msg: "command required: scan | version"}
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(newScanCmd())
	cmd.AddCommand(newVersionCmd())
	return cmd
}

// execute runs the CLI with args (without the program name) and returns an exit code.
func execute(args []string) int {
	root := newRootCmd()
	root.SetArgs(args)

	err := root.Execute()
	if err != nil {
		// Avoid double-printing the "command required" message after Help.
		var ee *exitError
		if !(errors.As(err, &ee) && ee.code == 2 && ee.msg == "command required: scan | version") {
			fmt.Fprintln(root.ErrOrStderr(), err)
		}
	}
	return exitCode(err)
}

// executeForTest runs the CLI and captures writers for unit tests.
func executeForTest(stdout, stderr io.Writer, args []string) int {
	root := newRootCmd()
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)

	err := root.Execute()
	if err != nil {
		var ee *exitError
		if !(errors.As(err, &ee) && ee.code == 2 && ee.msg == "command required: scan | version") {
			fmt.Fprintln(stderr, err)
		}
	}
	return exitCode(err)
}
