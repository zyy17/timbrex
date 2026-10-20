package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zyy17/timbrex/internal/providers"
	_ "github.com/zyy17/timbrex/internal/providers/stub"
)

const version = "0.1.0"

// Run parses args and runs a command.
func Run(args []string) error {
	return run(args, os.Stdin, os.Stdout, os.Stderr)
}

func run(args []string, in io.Reader, out, errOut io.Writer) error {
	cmd := newCommand()
	cmd.SetArgs(args)
	cmd.SetIn(in)
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	return cmd.Execute()
}

func newCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "timbrex",
		Short:         "compile expressive speech scripts to audio",
		Long:          longHelp(),
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(versionCmd(), checkCmd(), synthCmd())
	return cmd
}

func longHelp() string {
	return fmt.Sprintf(`Timbrex reads a .tbx script and can check it or synthesize audio.

  timbrex check script.tbx
  timbrex synth script.tbx -o out.wav

Providers: %s
The stub provider writes a short tone, not speech.
`, strings.Join(providers.IDs(), ", "))
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print the timbrex version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "timbrex", version)
		},
	}
}
