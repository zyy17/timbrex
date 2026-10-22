package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zyy17/timbrex/internal/check"
	"github.com/zyy17/timbrex/internal/config"
)

func checkCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check <file>",
		Short: "parse a script and report errors",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, diags, err := loadScript(cmd, args[0], cfg.StrictTags)
			if err != nil {
				return err
			}
			writeDiags(cmd.ErrOrStderr(), diags)
			if err := check.Failed(diags); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ok")
			return nil
		},
	}
	addStrictFlags(cmd)
	return cmd
}
