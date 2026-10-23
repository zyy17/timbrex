package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zyy17/timbrex/internal/check"
	"github.com/zyy17/timbrex/internal/config"
	"github.com/zyy17/timbrex/internal/lower"
	"github.com/zyy17/timbrex/internal/providers"
)

type synthFlags struct {
	provider string
	out      string
	model    string
	voice    string
}

func synthCmd(cfg *config.Config) *cobra.Command {
	var flags synthFlags
	cmd := &cobra.Command{
		Use:   "synth <file>",
		Short: "synthesize a script to an audio file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return synthesize(cmd, args[0], flags, *cfg)
		},
	}
	cmd.Flags().StringVar(&flags.provider, "provider", "", "provider id (overrides front matter and configuration)")
	cmd.Flags().StringVarP(&flags.out, "out", "o", "", "output audio file (default: script path with .wav)")
	cmd.Flags().StringVar(&flags.model, "model", "", "provider model (overrides front matter and configuration)")
	cmd.Flags().StringVar(&flags.voice, "voice", "", "default voice for speakers without a cast voice mapping")
	addStrictFlags(cmd)
	return cmd
}

func synthesize(cmd *cobra.Command, path string, flags synthFlags, cfg config.Config) error {
	script, diags, err := loadScript(cmd, path, cfg.StrictTags)
	if err != nil {
		return err
	}
	writeDiags(cmd.ErrOrStderr(), diags)
	if err := cmd.Context().Err(); err != nil {
		return err
	}
	if err := check.Failed(diags); err != nil {
		return err
	}
	doc := lower.Script(script)
	if err := cmd.Context().Err(); err != nil {
		return err
	}
	id := config.First(flags.provider, doc.Provider, cfg.Provider)
	p, err := providers.Get(id)
	if err != nil {
		return err
	}
	settings, err := cfg.Resolve(id, config.First(flags.model, doc.Model), config.First(flags.voice, doc.Voice))
	if err != nil {
		return err
	}
	outPath := flags.out
	if outPath == "" {
		outPath = strings.TrimSuffix(path, filepath.Ext(path)) + ".wav"
	}
	if err := checkOutput(path, outPath); err != nil {
		return err
	}
	opt := providers.Options{
		Out: outPath, Model: settings.Model, Voice: settings.Voice,
		APIKey: settings.APIKey, Endpoint: settings.Endpoint,
	}
	if err := cmd.Context().Err(); err != nil {
		return err
	}
	res, err := p.Synthesize(cmd.Context(), doc, opt)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), res.Path)
	return nil
}

func checkOutput(path, out string) error {
	if !strings.EqualFold(filepath.Ext(out), ".wav") {
		return fmt.Errorf("output must use the .wav extension")
	}
	src, err := os.Stat(path)
	if err != nil {
		return err
	}
	dst, err := os.Stat(out)
	if err == nil && dst.IsDir() {
		return fmt.Errorf("output path is a directory")
	}
	if err == nil && os.SameFile(src, dst) {
		return fmt.Errorf("output must not replace the source script")
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
