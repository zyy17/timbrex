package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/zyy17/timbrex/internal/config"
	"github.com/zyy17/timbrex/internal/parse"
	"github.com/zyy17/timbrex/internal/providers"
	_ "github.com/zyy17/timbrex/internal/providers/azure"
	_ "github.com/zyy17/timbrex/internal/providers/elevenlabs"
	_ "github.com/zyy17/timbrex/internal/providers/gemini"
	_ "github.com/zyy17/timbrex/internal/providers/minimax"
	_ "github.com/zyy17/timbrex/internal/providers/openai"
	_ "github.com/zyy17/timbrex/internal/providers/qwen"
	_ "github.com/zyy17/timbrex/internal/providers/stub"
)

const version = "0.1.0"

// Run parses args and runs a command.
// Interrupt and SIGTERM cancel the command context.
func Run(args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return run(ctx, args, os.Stdin, os.Stdout, os.Stderr)
}

// FormatError is the stderr line for a failed command.
// A parse error already starts with file:line:col, so it is not prefixed.
// Other failures get an "error:" prefix. The process still exits 1.
func FormatError(err error) string {
	var pe *parse.Error
	if errors.As(err, &pe) {
		return pe.Error()
	}
	return "error: " + err.Error()
}

func run(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) error {
	cmd := newCommand()
	cmd.SetArgs(args)
	cmd.SetIn(in)
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	return cmd.ExecuteContext(ctx)
}

func newCommand() *cobra.Command {
	var configPath string
	var cfg config.Config
	cmd := &cobra.Command{
		Use:           "timbrex",
		Short:         "compile expressive speech scripts to audio",
		Long:          longHelp(),
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			switch cmd.Name() {
			case "check", "synth", "config":
				var err error
				cfg, err = config.Load(configPath)
				return err
			default:
				return nil
			}
		},
	}
	cmd.PersistentFlags().StringVar(&configPath, "config", "", "configuration file (default: ./timbrex.yaml if present)")
	cmd.SetVersionTemplate("timbrex " + version + "\n")
	cmd.AddCommand(versionCmd(), checkCmd(&cfg), synthCmd(&cfg), configCmd(&cfg))
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
