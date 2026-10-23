package lower

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/ast"
	"github.com/zyy17/timbrex/internal/check"
	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/parse"
)

func checkedScript(t testing.TB, src string, strict bool) *ast.Script {
	t.Helper()
	script, err := parse.Parse("test.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if script == nil {
		t.Fatal("parse succeeded without a script")
	}
	if diags := check.Apply(script, strict); check.Failed(diags) != nil {
		t.Fatal(diags)
	}
	return script
}

func TestLowerDeepSpans(t *testing.T) {
	const depth = 12000
	script := checkedScript(t, strings.Repeat("[happy|", depth)+"hello"+strings.Repeat("]", depth), true)
	doc := Script(script)
	nodes := doc.Utterances[0].Nodes
	if len(nodes) != 2*depth+1 || nodes[depth].Kind != ir.Text || nodes[depth].Text != "hello" {
		t.Fatalf("nodes = %d", len(nodes))
	}
	for i := range depth {
		if nodes[i].Kind != ir.SpanStart || nodes[i].Name != "happy" || nodes[2*depth-i].Kind != ir.SpanEnd || nodes[2*depth-i].Name != "happy" {
			t.Fatalf("unbalanced span %d", i)
		}
	}
}

func BenchmarkLowerDeepSpans(b *testing.B) {
	for _, depth := range []int{400, 4000} {
		b.Run(strconv.Itoa(depth), func(b *testing.B) {
			script := checkedScript(b, strings.Repeat("[happy|", depth)+"hello"+strings.Repeat("]", depth), true)
			b.ReportAllocs()
			for b.Loop() {
				Script(script)
			}
		})
	}
}

func TestLowerDeepSpansAllocationGrowth(t *testing.T) {
	const depth = 1000
	script := checkedScript(t, strings.Repeat("[happy|", depth)+"hello"+strings.Repeat("]", depth), true)
	// Empty-argument spans only need space in the output buffer; building a
	// separate descendant slice for every span makes memory use quadratic.
	allocs := testing.AllocsPerRun(1, func() { Script(script) })
	if allocs > 100 {
		t.Fatalf("lowering %d empty-argument spans used %.0f allocations", depth, allocs)
	}
}

func TestLowerCompilerBoundary(t *testing.T) {
	tests := []struct {
		name   string
		src    string
		strict bool
		want   ir.Document
	}{
		{
			name: "implicit acting and nested spans",
			src:  "[strong French accent]\nShe [holding back tears] says [thoughtfully|very [softly spoken|quiet]] now\n",
			want: ir.Document{
				Voices: map[string]map[string]string{},
				Utterances: []ir.Utterance{
					{Nodes: []ir.Node{{Kind: ir.State, Name: "act", Note: "strong French accent", Args: []ir.Arg{}}}},
					{Speaker: "narrator", Nodes: []ir.Node{
						{Kind: ir.Text, Text: "She "},
						{Kind: ir.State, Name: "act", Note: "holding back tears", Args: []ir.Arg{}},
						{Kind: ir.Text, Text: " says "},
						{Kind: ir.SpanStart, Name: "act", Note: "thoughtfully", Args: []ir.Arg{}},
						{Kind: ir.Text, Text: "very "},
						{Kind: ir.SpanStart, Name: "act", Note: "softly spoken", Args: []ir.Arg{}},
						{Kind: ir.Text, Text: "quiet"},
						{Kind: ir.SpanEnd, Name: "act"},
						{Kind: ir.SpanEnd, Name: "act"},
						{Kind: ir.Text, Text: " now"},
					}},
				},
			},
		},
		{
			name:   "explicit acting and quoted decoding",
			src:    `Say [ACT "hold \"steady\" \\ now"] [act holding back tears|hi]`,
			strict: true,
			want: ir.Document{
				Voices: map[string]map[string]string{},
				Utterances: []ir.Utterance{{Speaker: "narrator", Nodes: []ir.Node{
					{Kind: ir.Text, Text: "Say "},
					{Kind: ir.State, Name: "act", Args: []ir.Arg{{Raw: `hold "steady" \ now`, Kind: "quoted"}}},
					{Kind: ir.Text, Text: " "},
					{Kind: ir.SpanStart, Name: "act", Note: "holding back tears", Args: []ir.Arg{}},
					{Kind: ir.Text, Text: "hi"},
					{Kind: ir.SpanEnd, Name: "act"},
				}}},
			},
		},
		{
			name:   "mixed case names notes and nested spans",
			src:    `Start [LaUgH nervously level=50%|one [WhIsPeR|two] three] [QuIcK_BrEaTh] [PaUsE 250ms]`,
			strict: true,
			want: ir.Document{
				Voices: map[string]map[string]string{},
				Utterances: []ir.Utterance{{Speaker: "narrator", Nodes: []ir.Node{
					{Kind: ir.Text, Text: "Start "},
					{Kind: ir.SpanStart, Name: "laugh", Note: "nervously", Args: []ir.Arg{{Key: "level", Raw: "50%", Kind: "percent"}}},
					{Kind: ir.Text, Text: "one "},
					{Kind: ir.SpanStart, Name: "whisper", Args: []ir.Arg{}},
					{Kind: ir.Text, Text: "two"},
					{Kind: ir.SpanEnd, Name: "whisper"},
					{Kind: ir.Text, Text: " three"},
					{Kind: ir.SpanEnd, Name: "laugh"},
					{Kind: ir.Text, Text: " "},
					{Kind: ir.Event, Name: "quick-breath", Args: []ir.Arg{}},
					{Kind: ir.Text, Text: " "},
					{Kind: ir.Event, Name: "pause", Args: []ir.Arg{{Raw: "250ms", Kind: "duration"}}},
				}}},
			},
		},
		{
			name:   "all positional and named argument kinds",
			src:    `[bgm rain.wav -3 250ms +50% -2st -22dB "say \"hi\" \\ there" file=wind.wav amount=0.5 delay=1s ratio=10% pitch=+3st vol=-10dB label="a|b [c]"]`,
			strict: true,
			want: ir.Document{
				Voices: map[string]map[string]string{},
				Utterances: []ir.Utterance{{Nodes: []ir.Node{{Kind: ir.Event, Name: "bgm", Args: []ir.Arg{
					{Raw: "rain.wav", Kind: "word"},
					{Raw: "-3", Kind: "number"},
					{Raw: "250ms", Kind: "duration"},
					{Raw: "+50%", Kind: "percent"},
					{Raw: "-2st", Kind: "semitone"},
					{Raw: "-22dB", Kind: "decibel"},
					{Raw: `say "hi" \ there`, Kind: "quoted"},
					{Key: "file", Raw: "wind.wav", Kind: "word"},
					{Key: "amount", Raw: "0.5", Kind: "number"},
					{Key: "delay", Raw: "1s", Kind: "duration"},
					{Key: "ratio", Raw: "10%", Kind: "percent"},
					{Key: "pitch", Raw: "+3st", Kind: "semitone"},
					{Key: "vol", Raw: "-10dB", Kind: "decibel"},
					{Key: "label", Raw: "a|b [c]", Kind: "quoted"},
				}}}}},
			},
		},
		{
			name: "front matter scene turn and raw fence",
			src: "---\nprovider: stub\nmodel: demo\nvoice: default\nlang: zh-CN\nnarrator: host\ncast:\n" +
				"  host: {name: Host, voices: {stub: host-voice, azure: en-US-JennyNeural}}\n" +
				"  lin: {name: Lin, voices: {stub: lin-voice}}\n---\n" +
				"// dropped\n[SlOw]\nHello\nlin: [HaPpY] Hi\n```ssml azure\n<speak>hi</speak>\n```\n",
			strict: true,
			want: ir.Document{
				Provider: "stub", Model: "demo", Voice: "default", Lang: "zh-CN",
				Voices: map[string]map[string]string{
					"host": {"stub": "host-voice", "azure": "en-US-JennyNeural"},
					"lin":  {"stub": "lin-voice"},
				},
				Utterances: []ir.Utterance{
					{Nodes: []ir.Node{{Kind: ir.State, Name: "slow", Args: []ir.Arg{}}}},
					{Speaker: "host", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}},
					{Speaker: "lin", Nodes: []ir.Node{
						{Kind: ir.State, Name: "happy", Args: []ir.Arg{}},
						{Kind: ir.Text, Text: " Hi"},
					}},
					{Nodes: []ir.Node{{Kind: ir.Raw, Name: "ssml azure", Text: "<speak>hi</speak>\n"}}},
				},
			},
		},
		{
			name:   "turn timing remains events on scene and speaker lines",
			src:    "---\ncast: auto\n---\n[overlap -200ms]\nlin: [gap 1s] Hello\n[gap 500ms]\nlin: Again\n[pause]\n",
			strict: true,
			want: ir.Document{
				Voices: map[string]map[string]string{},
				Utterances: []ir.Utterance{
					{Nodes: []ir.Node{{Kind: ir.Event, Name: "overlap", Args: []ir.Arg{{Raw: "-200ms", Kind: "duration"}}}}},
					{Speaker: "lin", Nodes: []ir.Node{
						{Kind: ir.Event, Name: "gap", Args: []ir.Arg{{Raw: "1s", Kind: "duration"}}},
						{Kind: ir.Text, Text: " Hello"},
					}},
					{Nodes: []ir.Node{{Kind: ir.Event, Name: "gap", Args: []ir.Arg{{Raw: "500ms", Kind: "duration"}}}}},
					{Speaker: "lin", Nodes: []ir.Node{{Kind: ir.Text, Text: "Again"}}},
					{Nodes: []ir.Node{{Kind: ir.Event, Name: "pause", Args: []ir.Arg{}}}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Script(checkedScript(t, tt.src, tt.strict))
			if !reflect.DeepEqual(*got, tt.want) {
				t.Fatalf("document = %#v\nwant = %#v", *got, tt.want)
			}
		})
	}
}
