package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/zyy17/timbrex/internal/ast"
	"github.com/zyy17/timbrex/internal/check"
	"github.com/zyy17/timbrex/internal/inputfile"
	"github.com/zyy17/timbrex/internal/parse"
)

func addStrictFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("strict-tags", false, "reject unregistered tags")
	cmd.Flags().Bool("no-strict-tags", false, "allow unregistered tags as free-form acting")
	cmd.MarkFlagsMutuallyExclusive("strict-tags", "no-strict-tags")
}

func loadScript(cmd *cobra.Command, path string, defaultStrict bool) (*ast.Script, []check.Diag, error) {
	src, err := inputfile.Read(path)
	if err != nil {
		return nil, nil, err
	}
	script, err := parse.Parse(path, string(src))
	if err != nil {
		return nil, nil, err
	}
	diags := check.Apply(script, strictMode(cmd, script.Front.StrictTags, defaultStrict))
	return script, diags, nil
}

func strictMode(cmd *cobra.Command, front *bool, fallback bool) bool {
	if cmd.Flags().Changed("strict-tags") {
		on, _ := cmd.Flags().GetBool("strict-tags")
		return on
	}
	if cmd.Flags().Changed("no-strict-tags") {
		off, _ := cmd.Flags().GetBool("no-strict-tags")
		return !off
	}
	if front != nil {
		return *front
	}
	return fallback
}

func writeDiags(w io.Writer, diags []check.Diag) {
	for _, d := range diags {
		fmt.Fprintln(w, d.Error())
	}
}
