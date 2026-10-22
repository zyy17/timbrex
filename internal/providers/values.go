package providers

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/zyy17/timbrex/internal/ir"
)

func First(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func RequireEnv(name string) (string, error) {
	v := os.Getenv(name)
	if strings.TrimSpace(v) == "" {
		return "", fmt.Errorf("set %s to use this provider", name)
	}
	return v, nil
}

func Value(n ir.Node) (string, error) {
	if len(n.Args) != 1 || n.Args[0].Key != "" || n.Args[0].Raw == "" || n.Note != "" {
		return "", fmt.Errorf("[%s] needs one positional value", n.Name)
	}
	return n.Args[0].Raw, nil
}

func NoArgs(n ir.Node) error {
	if len(n.Args) != 0 || n.Note != "" {
		return fmt.Errorf("[%s] does not support arguments here", n.Name)
	}
	return nil
}

func Number(n ir.Node, suffix string, low, high float64) (float64, error) {
	v, err := Value(n)
	if err != nil {
		return 0, err
	}
	if suffix != "" && !strings.HasSuffix(v, suffix) {
		return 0, fmt.Errorf("[%s] needs a %s value", n.Name, suffix)
	}
	x, err := strconv.ParseFloat(strings.TrimSuffix(v, suffix), 64)
	if err != nil || math.IsNaN(x) || math.IsInf(x, 0) || x < low || x > high {
		return 0, fmt.Errorf("[%s] needs a value from %g to %g%s", n.Name, low, high, suffix)
	}
	return x, nil
}

func Speed(n ir.Node, low, high float64) (float64, error) {
	if n.Name == "fast" || n.Name == "slow" {
		if err := NoArgs(n); err != nil {
			return 0, err
		}
		if n.Name == "fast" {
			return 1.2, nil
		}
		return 0.8, nil
	}
	if len(n.Args) == 1 && n.Args[0].Kind == "percent" {
		x, err := Number(n, "%", low*100, high*100)
		return x / 100, err
	}
	return Number(n, "", low, high)
}

// Direction describes acting without putting instructions into spoken text.
func Direction(n ir.Node) (string, error) {
	if n.Name == "act" {
		if len(n.Args) == 0 && n.Note != "" {
			return n.Note, nil
		}
		return Value(n)
	}
	if feature(n.Name) == "emotion" && n.Name != "emotion" {
		if len(n.Args) == 0 {
			return n.Name, NoArgs(n)
		}
		x, err := Number(n, "", 0, 1)
		return fmt.Sprintf("%s (intensity %g of 1)", n.Name, x), err
	}
	switch n.Name {
	case "laugh", "cry", "sigh":
		if len(n.Args) != 0 {
			return "", fmt.Errorf("[%s] span does not support numeric arguments", n.Name)
		}
		return strings.TrimSpace("speak while " + n.Name + "ing " + n.Note), nil
	case "whisper", "shout", "emph":
		return n.Name, NoArgs(n)
	case "style", "role", "accent", "dialect", "emotion":
		v, err := Value(n)
		return n.Name + ": " + v, err
	default:
		return "", fmt.Errorf("unsupported tag [%s]", n.Name)
	}
}

func Text(nodes []ir.Node, events map[string]string) (string, error) {
	var b strings.Builder
	for _, n := range nodes {
		if n.Kind == ir.Text {
			b.WriteString(n.Text)
			continue
		}
		v, ok := events[n.Name]
		if !ok {
			return "", fmt.Errorf("unsupported event [%s]", n.Name)
		}
		if err := NoArgs(n); err != nil {
			return "", err
		}
		b.WriteString(v)
	}
	return b.String(), nil
}

func ReservedText(nodes []ir.Node, marks ...string) error {
	for _, n := range nodes {
		if n.Kind != ir.Text {
			continue
		}
		for _, mark := range marks {
			if strings.Contains(n.Text, mark) {
				return fmt.Errorf("literal %q conflicts with this provider's native markup; use a Unicode alternative", mark)
			}
		}
	}
	return nil
}
