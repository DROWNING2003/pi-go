package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

const usage = "Usage: pi [options] [message...]\n\nOptions:\n  --help       Show help\n  --version    Show version\n"

// Run executes the command-line entrypoint and returns the process exit code.
func Run(args []string, stdout io.Writer, stderr io.Writer, version string) int {
	command := &cobra.Command{
		Use:           "pi [options] [message...]",
		Short:         "Pi coding agent",
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       version,
	}
	command.SetArgs(args)
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetHelpFunc(func(_ *cobra.Command, _ []string) {
		_, _ = io.WriteString(stdout, usage)
	})
	command.SetVersionTemplate("pi {{.Version}}\n")

	if _, err := command.ExecuteC(); err != nil {
		message := strings.Replace(err.Error(), "unknown flag:", "unknown option:", 1)
		_, _ = fmt.Fprintln(stderr, message)
		return 2
	}
	return 0
}
