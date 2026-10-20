package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/zyy17/timbrex/internal/ast"
	"github.com/zyy17/timbrex/internal/check"
	"github.com/zyy17/timbrex/internal/parse"
)

func addStrictFlags(cmd *cobra.Command, on, off *bool) {
	cmd.Flags().BoolVar(on, "strict-tags", false, "reject unregistered tags")
	cmd.Flags().BoolVar(off, "no-strict-tags", false, "allow unregistered tags as free-form acting")
	cmd.MarkFlagsMutuallyExclusive("strict-tags", "no-strict-tags")
}

func loadScript(path string, strictOn, strictOff bool) (*ast.Script, []check.Diag, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	script, err := parse.Parse(path, string(src))
	if err != nil {
		return nil, nil, err
	}
	diags := check.Apply(script, strictMode(strictOn, strictOff, script.Front.StrictTags))
	return script, diags, nil
}

func strictMode(on, off bool, front *bool) bool {
	if on {
		return true
	}
	if off {
		return false
	}
	if front != nil {
		return *front
	}
	return false
}

func writeDiags(w io.Writer, diags []check.Diag) {
	for _, d := range diags {
		fmt.Fprintln(w, d.Error())
	}
}
