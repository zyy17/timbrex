package cli

import (
	"fmt"
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
	cmd := &cobra.Command{
		Use:   "synth <file>",
		Short: "synthesize a script to an audio file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return synthesize(cmd, args[0], providerName, outPath, strictOn, strictOff)
		},
	}
	cmd.Flags().StringVar(&providerName, "provider", "", "provider id (default: front matter, else stub)")
	cmd.Flags().StringVarP(&outPath, "out", "o", "", "output audio file (default: script path with .wav)")
	addStrictFlags(cmd, &strictOn, &strictOff)
	return cmd
}

func synthesize(cmd *cobra.Command, path, providerName, outPath string, strictOn, strictOff bool) error {
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
	res, err := p.Synthesize(cmd.Context(), doc, providers.Options{Path: path, Out: outPath})
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), res.Path)
	return nil
}
