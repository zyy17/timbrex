package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zyy17/timbrex/internal/check"
	"github.com/zyy17/timbrex/internal/lower"
	"github.com/zyy17/timbrex/internal/providers"
)

func synthCmd() *cobra.Command {
	var providerName string
	var outPath string
	var strictOn, strictOff bool
	var opt providers.Options
	cmd := &cobra.Command{
		Use:   "synth <file>",
		Short: "synthesize a script to an audio file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return synthesize(cmd, args[0], providerName, outPath, strictOn, strictOff, opt)
		},
	}
	cmd.Flags().StringVar(&providerName, "provider", "", "provider id (default: front matter, else stub)")
	cmd.Flags().StringVarP(&outPath, "out", "o", "", "output audio file (default: script path with .wav)")
	cmd.Flags().StringVar(&opt.Model, "model", "", "provider model (overrides front matter and environment)")
	cmd.Flags().StringVar(&opt.Voice, "voice", "", "default voice for speakers without a cast voice mapping")
	addStrictFlags(cmd, &strictOn, &strictOff)
	return cmd
}

func synthesize(cmd *cobra.Command, path, providerName, outPath string, strictOn, strictOff bool, opt providers.Options) error {
	script, diags, err := loadScript(path, strictOn, strictOff)
	if err != nil {
		return err
	}
	writeDiags(cmd.ErrOrStderr(), diags)
	if err := check.Failed(diags); err != nil {
		return err
	}
	doc := lower.Script(script)
	id := providerName
	if id == "" {
		id = doc.Provider
	}
	if id == "" {
		id = "stub"
	}
	p, err := providers.Get(id)
	if err != nil {
		return err
	}
	if outPath == "" {
		outPath = strings.TrimSuffix(path, filepath.Ext(path)) + ".wav"
	}
	if err := checkOutput(path, outPath); err != nil {
		return err
	}
	opt.Path, opt.Out = path, outPath
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
