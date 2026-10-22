package cli

import (
	"github.com/spf13/cobra"

	"github.com/zyy17/timbrex/internal/config"
)

func configCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "show file and environment settings with API keys redacted",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			data, err := cfg.RedactedYAML()
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		},
	}
}
