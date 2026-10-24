package audioexport

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/zyy17/timbrex/internal/providers"
)

// These tests use actual encoders and decoders, not the fake-process fixtures.
// Only absent tools are optional; an installed build missing a codec must fail.
func TestRealCodecsPreserveComposition(t *testing.T) {
	ffmpeg := realCodecTool(t, "ffmpeg")
	ffprobe := realCodecTool(t, "ffprobe")
	formats := []struct {
		extension string
		codec     string
		container string
		edgeError float64
	}{
		{".Mp3", "mp3", "mp3", 0.008},
		{".FLAC", "flac", "flac", 0.004},
		{".OpUs", "opus", "ogg", 0.006},
		{".AaC", "aac", "aac", 0.016},
	}
	for _, rate := range []int{24000, 8000} {
		for _, f := range formats {
			t.Run(fmt.Sprintf("%dHz/%s", rate, f.codec), func(t *testing.T) {
				wav, source, tones := codecFixture(rate)
				path := filepath.Join(t.TempDir(), "speech"+f.extension)
				previous := []byte("previous output must survive preflight")
				if err := os.WriteFile(path, previous, 0o600); err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
				defer cancel()
				export, err := Prepare(ctx, path)
				if err != nil {
					t.Fatalf("real codec preflight: %v", err)
				}
				defer export.Close()
				if export.InputPath() == path || filepath.Ext(export.InputPath()) != ".wav" {
					t.Fatalf("compressed export needs a separate WAV input: %q", export.InputPath())
				}
				writeCodecSource(t, ctx, export.InputPath(), rate, wav, source, tones)
				if before, err := os.ReadFile(path); err != nil || !bytes.Equal(before, previous) {
					t.Fatalf("destination changed before Finish: %q, %v", before, err)
				}
				if err := export.Finish(ctx); err != nil {
					t.Fatal(err)
				}
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				if info.Mode().Perm() != 0o600 {
					t.Errorf("published file permissions = %o, want 600", info.Mode().Perm())
				}
				var probe codecProbe
				if err := json.Unmarshal(codecCommand(t, ffprobe, "-v", "error", "-show_streams", "-show_format", "-show_packets", "-of", "json", path), &probe); err != nil {
					t.Fatal(err)
				}
				if len(probe.Streams) != 1 || probe.Streams[0].CodecName != f.codec || probe.Streams[0].Channels != 1 || probe.Format.FormatName != f.container {
					t.Fatalf("unexpected real ffprobe metadata: streams=%+v, format=%+v", probe.Streams, probe.Format)
				}
				decodeRate, err := strconv.Atoi(probe.Streams[0].SampleRate)
				if err != nil {
					t.Fatal(err)
				}
				wantRate := rate
				if f.codec == "opus" {
					wantRate = 48000 // Opus uses a 48 kHz decode clock, not a length change.
				}
				if decodeRate != wantRate {
					t.Fatalf("sample rate = %d, want %d", decodeRate, wantRate)
				}
				// Do not set -ar or -ac: that could conceal a changed rate or channels.
				pcm := codecCommand(t, ffmpeg, "-v", "error", "-i", path, "-map", "0:a:0", "-c:a", "pcm_s16le", "-f", "s16le", "pipe:1")
				if len(pcm)%2 != 0 {
					t.Fatalf("odd S16LE byte count: %d", len(pcm))
				}
				wantSamples := len(source) / 2 * decodeRate / rate
				delay := 0
				switch f.codec {
				case "flac":
					if !bytes.Equal(pcm, source) {
						t.Fatal("FLAC decode is not byte-identical to the source S16LE PCM")
					}
					if probe.Streams[0].DurationTS != int64(wantSamples) || probe.Streams[0].TimeBase != fmt.Sprintf("1/%d", rate) {
						t.Fatalf("FLAC duration does not preserve source samples: %+v", probe.Streams[0])
					}
				case "mp3":
					checkMP3Gapless(t, path, probe, wantSamples, rate)
				case "opus":
					checkOpusPreskip(t, path, probe, wantSamples, rate)
				case "aac":
					// ADTS cannot carry gapless trimming. Account for the native AAC
					// encoder's 1024-sample delay, never claim sample-exact output.
					delay = 1024
					padding := len(pcm)/2 - wantSamples - delay
					if padding < 0 || padding > 1023 || len(pcm)/2%1024 != 0 || len(probe.Packets)*1024 != len(pcm)/2 {
						t.Fatalf("AAC ADTS frames: source=%d decoded=%d delay=%d padding=%d packets=%d", wantSamples, len(pcm)/2, delay, padding, len(probe.Packets))
					}
					if probe.Streams[0].Profile != "LC" {
						t.Errorf("AAC profile = %q, want LC", probe.Streams[0].Profile)
					}
				}
				if f.codec != "aac" && len(pcm)/2 != wantSamples {
					t.Fatalf("decoded samples = %d, want %d (source duration preserved)", len(pcm)/2, wantSamples)
				}
				checkCodecComposition(t, pcm[delay*2:], decodeRate, tones, f.edgeError)
				t.Logf("source=%d Hz/%d samples, decoded=%d Hz/%d samples, accounted encoder delay=%d samples", rate, len(source)/2, decodeRate, len(pcm)/2, delay)
			})
		}
	}
}

type codecProbe struct {
	Streams []struct {
		CodecName  string `json:"codec_name"`
		Profile    string `json:"profile"`
		SampleRate string `json:"sample_rate"`
		Channels   int    `json:"channels"`
		DurationTS int64  `json:"duration_ts"`
		TimeBase   string `json:"time_base"`
		Duration   string `json:"duration"`
		StartTime  string `json:"start_time"`
	} `json:"streams"`
	Format struct {
		FormatName string `json:"format_name"`
	} `json:"format"`
	Packets []struct {
		SideData []struct {
			Type    string `json:"side_data_type"`
			Skip    int    `json:"skip_samples"`
			Discard int    `json:"discard_padding"`
		} `json:"side_data_list"`
	} `json:"packets"`
}

func realCodecTool(t *testing.T, name string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		if os.Getenv("TIMBREX_TEST_FFMPEG") != "1" && errors.Is(err, exec.ErrNotFound) {
			t.Skipf("optional real-codec tests need %s on PATH: %v", name, err)
		}
		t.Fatalf("required real-codec tool %s: %v", name, err)
	}
	return path
}

func codecCommand(t *testing.T, executable string, args ...string) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Env = []string{"LANG=C", "LC_ALL=C"}
	cmd.WaitDelay = time.Second
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", executable, args, err, stderr.String())
	}
	return output
}

func checkMP3Gapless(t *testing.T, path string, probe codecProbe, samples, rate int) {
	t.Helper()
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	header := encoded[:min(len(encoded), 2048)]
	if !bytes.Contains(header, []byte("Info")) && !bytes.Contains(header, []byte("Xing")) {
		t.Fatal("MP3 is missing the seekable Info/Xing frame")
	}
	if len(probe.Packets) < 2 {
		t.Fatal("MP3 contains too few packets")
	}
	skip, discard := 0, 0
	for _, side := range probe.Packets[0].SideData {
		if side.Type == "Skip Samples" {
			skip += side.Skip
		}
	}
	for _, side := range probe.Packets[len(probe.Packets)-1].SideData {
		if side.Type == "Skip Samples" {
			discard += side.Discard
		}
	}
	// Both source rates use MPEG-2/2.5's 576-sample frames. Packet metadata
	// must describe both ends, unlike a blind pipe without a completed Xing tag.
	if skip <= 0 || skip > 1152 || discard <= 0 || discard >= 576 || len(probe.Packets)*576-skip-discard != samples {
		t.Fatalf("MP3 gapless accounting: packets=%d skip=%d discard=%d source=%d", len(probe.Packets), skip, discard, samples)
	}
	// FFmpeg 7 reports the full encoded frames; FFmpeg 8 subtracts the
	// encoder delay and padding. Accept only those two derived durations,
	// rounded to ffprobe's six decimal places, not a percentage tolerance.
	if !mp3DurationMatches(probe.Streams[0].Duration, samples, len(probe.Packets)*576, rate) {
		t.Errorf("MP3 reported duration = %q, want audible or encoded frame duration", probe.Streams[0].Duration)
	}
	start, err := strconv.ParseFloat(probe.Streams[0].StartTime, 64)
	if err != nil || math.Abs(start-float64(skip)/float64(rate)) > 0.000001 {
		t.Errorf("MP3 start_time = %q, want gapless skip time", probe.Streams[0].StartTime)
	}
}

func mp3DurationMatches(raw string, samples, frameSamples, rate int) bool {
	duration, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return false
	}
	return math.Abs(duration-float64(samples)/float64(rate)) <= 0.000001 ||
		math.Abs(duration-float64(frameSamples)/float64(rate)) <= 0.000001
}

func TestMP3ReportedDurationConventions(t *testing.T) {
	for _, tc := range []struct {
		rate, samples, frameSamples int
	}{
		{24000, 37765, 68 * 576},
		{8000, 12613, 24 * 576},
	} {
		for _, samples := range []int{tc.samples, tc.frameSamples} {
			raw := fmt.Sprintf("%.6f", float64(samples)/float64(tc.rate))
			if !mp3DurationMatches(raw, tc.samples, tc.frameSamples, tc.rate) {
				t.Errorf("rejected valid %d Hz duration %q", tc.rate, raw)
			}
		}
		for _, raw := range []string{"", "invalid", "NaN", "+Inf", "-Inf", "-1", "0",
			fmt.Sprintf("%.6f", float64(tc.samples+1)/float64(tc.rate)),
			fmt.Sprintf("%.6f", float64(tc.frameSamples+576)/float64(tc.rate)),
			fmt.Sprintf("%.6f", float64(tc.samples)*1.01/float64(tc.rate)),
		} {
			if mp3DurationMatches(raw, tc.samples, tc.frameSamples, tc.rate) {
				t.Errorf("accepted invalid %d Hz duration %q", tc.rate, raw)
			}
		}
	}
}

func checkOpusPreskip(t *testing.T, path string, probe codecProbe, samples, sourceRate int) {
	t.Helper()
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	head := bytes.Index(encoded, []byte("OpusHead"))
	if head < 0 || len(encoded)-head < 19 || len(probe.Packets) == 0 {
		t.Fatal("Opus identification header or packets are missing")
	}
	preSkip := int(binary.LittleEndian.Uint16(encoded[head+10:]))
	inputRate := int(binary.LittleEndian.Uint32(encoded[head+12:]))
	packetSkip := 0
	for _, side := range probe.Packets[0].SideData {
		if side.Type == "Skip Samples" {
			packetSkip += side.Skip
		}
	}
	if preSkip <= 0 || packetSkip != preSkip || inputRate != sourceRate {
		t.Fatalf("Opus pre-skip=%d packet skip=%d input rate=%d, want positive matching pre-skip and rate=%d", preSkip, packetSkip, inputRate, sourceRate)
	}
	if probe.Streams[0].TimeBase != "1/48000" || probe.Streams[0].DurationTS != int64(samples+preSkip) {
		t.Fatalf("Opus granule duration must include pre-skip, not change audible duration: %+v", probe.Streams[0])
	}
}

type codecTone struct {
	start, end, frequency float64
}

func writeCodecSource(t *testing.T, ctx context.Context, path string, rate int, wav, pcm []byte, tones []codecTone) {
	t.Helper()
	if rate != providers.SampleRate {
		// The stub's 8 kHz exception does not use the real providers' renderer.
		if err := os.WriteFile(path, wav, 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	segments := []providers.Segment{
		{Pause: 80 * time.Millisecond}, {}, {Pause: 139 * time.Millisecond}, {}, {Pause: 197 * time.Millisecond}, {},
	}
	secondReady := make(chan struct{})
	_, err := providers.Render(ctx, "codec-fixture", providers.Options{Out: path, Concurrency: 4}, segments, func(ctx context.Context, index int) ([]byte, error) {
		// Make the second utterance ready before the first to exercise ordered
		// publication, while the renderer itself inserts the local pauses.
		switch index {
		case 1:
			select {
			case <-secondReady:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		case 3:
			close(secondReady)
		case 5:
		default:
			return nil, fmt.Errorf("local pause %d incorrectly reached synthesis", index)
		}
		tone := tones[index/2]
		start := int(math.Round(tone.start*float64(rate))) * 2
		end := int(math.Round(tone.end*float64(rate))) * 2
		if index == 5 {
			end = len(pcm) // Include the final provider's quiet tail and 37 samples.
		}
		return pcm[start:end], nil
	})
	if err != nil {
		t.Fatalf("render ordered source with local pauses: %v", err)
	}
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(actual, wav) {
		t.Fatalf("provider composition differs from the independent WAV fixture: %v", err)
	}
}

func codecFixture(rate int) (wav, pcm []byte, tones []codecTone) {
	sections := []struct {
		milliseconds int
		frequency    float64
	}{
		{80, 0}, {271, 375}, {139, 0}, {353, 825}, {197, 0}, {419, 1575}, {113, 0},
	}
	for _, section := range sections {
		start := len(pcm) / 2
		count := section.milliseconds * rate / 1000
		for i := range count {
			// Short ramps avoid clicks, without concealing the local boundaries.
			ramp := min(1, float64(i)/(0.003*float64(rate)), float64(count-1-i)/(0.003*float64(rate)))
			value := int16(math.Round(10000 * ramp * math.Sin(2*math.Pi*section.frequency*float64(i)/float64(rate))))
			pcm = binary.LittleEndian.AppendUint16(pcm, uint16(value))
		}
		if section.frequency != 0 {
			tones = append(tones, codecTone{float64(start) / float64(rate), float64(start+count) / float64(rate), section.frequency})
		}
	}
	pcm = append(pcm, make([]byte, 37*2)...) // Deliberately not codec-frame aligned.
	wav = make([]byte, 44)
	copy(wav, "RIFF")
	binary.LittleEndian.PutUint32(wav[4:], uint32(36+len(pcm)))
	copy(wav[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(wav[16:], 16)
	binary.LittleEndian.PutUint16(wav[20:], 1)
	binary.LittleEndian.PutUint16(wav[22:], 1)
	binary.LittleEndian.PutUint32(wav[24:], uint32(rate))
	binary.LittleEndian.PutUint32(wav[28:], uint32(rate*2))
	binary.LittleEndian.PutUint16(wav[32:], 2)
	binary.LittleEndian.PutUint16(wav[34:], 16)
	copy(wav[36:], "data")
	binary.LittleEndian.PutUint32(wav[40:], uint32(len(pcm)))
	return append(wav, pcm...), pcm, tones
}

func checkCodecComposition(t *testing.T, pcm []byte, rate int, tones []codecTone, edgeError float64) {
	t.Helper()
	samples := make([]float64, len(pcm)/2)
	for i := range samples {
		samples[i] = float64(int16(binary.LittleEndian.Uint16(pcm[i*2:])))
	}
	// Detect all audible regions, not just the file's overall duration. A 10 ms
	// RMS window at 1 ms hops contributes at most 4 ms to these ramped edges.
	var regions [][2]float64
	active := false
	window, hop := rate/100, rate/1000
	for start := 0; start+window <= len(samples); start += hop {
		on := codecRMS(samples[start:start+window]) > 2000
		at := float64(start+window/2) / float64(rate)
		if on && !active {
			regions = append(regions, [2]float64{at, -1})
		} else if !on && active {
			regions[len(regions)-1][1] = at
		}
		active = on
	}
	if len(regions) != len(tones) {
		t.Fatalf("audible regions = %v, want three separated utterances", regions)
	}
	for i, tone := range tones {
		if math.Abs(regions[i][0]-tone.start) > edgeError || math.Abs(regions[i][1]-tone.end) > edgeError {
			t.Errorf("tone %d boundaries = %.6f..%.6f, want %.6f..%.6f within %.0f ms", i, regions[i][0], regions[i][1], tone.start, tone.end, edgeError*1000)
		}
		mid := (tone.start + tone.end) / 2
		center := samples[int((mid-0.020)*float64(rate)):int((mid+0.020)*float64(rate))]
		amplitude := codecToneAmplitude(center, rate, tone.frequency)
		if amplitude < 5000 {
			t.Errorf("tone %d: expected %.0f Hz amplitude >= 5000, got %.1f", i, tone.frequency, amplitude)
		}
		for j, other := range tones {
			if j != i && codecToneAmplitude(center, rate, other.frequency) > amplitude/10 {
				t.Errorf("tone %d contains the tone from section %d: utterance order/content changed", i, j)
			}
		}
	}
	// Check local silence, including the leading and trailing margins. This is
	// an absolute amplitude bound, independent of the total file duration.
	for i := 0; i <= len(tones); i++ {
		start, end := 0.0, float64(len(samples))/float64(rate)
		if i > 0 {
			start = tones[i-1].end
		}
		if i < len(tones) {
			end = tones[i].start
		}
		mid := (start + end) / 2
		quiet := samples[int((mid-0.010)*float64(rate)):int((mid+0.010)*float64(rate))]
		if rms := codecRMS(quiet); rms > 150 {
			t.Errorf("pause %d RMS = %.1f, want <= 150", i, rms)
		}
	}
}

func codecRMS(samples []float64) float64 {
	var sum float64
	for _, sample := range samples {
		sum += sample * sample
	}
	return math.Sqrt(sum / float64(len(samples)))
}

func codecToneAmplitude(samples []float64, rate int, frequency float64) float64 {
	var realPart, imaginaryPart float64
	for i, sample := range samples {
		phase := 2 * math.Pi * frequency * float64(i) / float64(rate)
		realPart += sample * math.Cos(phase)
		imaginaryPart += sample * math.Sin(phase)
	}
	return 2 * math.Hypot(realPart, imaginaryPart) / float64(len(samples))
}
