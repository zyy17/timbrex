package providers

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func waveChunks(chunks ...[]byte) []byte {
	b := []byte("RIFF\x00\x00\x00\x00WAVE")
	for _, c := range chunks {
		b = append(b, c...)
	}
	binary.LittleEndian.PutUint32(b[4:], uint32(len(b)-8))
	return b
}

func waveChunk(id string, data []byte) []byte {
	b := append([]byte(id), make([]byte, 4)...)
	binary.LittleEndian.PutUint32(b[4:], uint32(len(data)))
	b = append(b, data...)
	if len(data)%2 != 0 {
		b = append(b, 0)
	}
	return b
}

func TestDecodeWAV(t *testing.T) {
	format := wavHeader(4)[20:36]
	pcm := []byte{0, 1, 2, 3}
	for _, chunks := range [][][]byte{
		{waveChunk("fmt ", format), waveChunk("data", pcm)},
		{waveChunk("JUNK", []byte{1}), waveChunk("fmt ", format), waveChunk("LIST", []byte{2, 3, 4}), waveChunk("data", pcm)},
		{waveChunk("data", pcm), waveChunk("fmt ", append(append([]byte{}, format...), 0, 0))},
	} {
		got, err := DecodeWAV(waveChunks(chunks...))
		if err != nil || !bytes.Equal(got, pcm) {
			t.Fatalf("got %x, %v", got, err)
		}
	}
}

func TestDecodeWAVRejectsInvalidAudio(t *testing.T) {
	format := wavHeader(4)[20:36]
	valid := waveChunks(waveChunk("fmt ", format), waveChunk("data", []byte{0, 1, 2, 3}))
	bad := map[string][]byte{
		"empty":            nil,
		"missing format":   waveChunks(waveChunk("data", []byte{0, 0})),
		"missing data":     waveChunks(waveChunk("fmt ", format)),
		"empty data":       waveChunks(waveChunk("fmt ", format), waveChunk("data", nil)),
		"odd PCM":          waveChunks(waveChunk("fmt ", format), waveChunk("data", []byte{0})),
		"duplicate format": waveChunks(waveChunk("fmt ", format), waveChunk("fmt ", format), waveChunk("data", []byte{0, 0})),
		"duplicate data":   waveChunks(waveChunk("fmt ", format), waveChunk("data", []byte{0, 0}), waveChunk("data", []byte{0, 0})),
		"truncated header": waveChunks([]byte("data")),
		"oversized chunk":  waveChunks([]byte("data\xff\xff\xff\xff")),
		"missing padding":  waveChunks([]byte("JUNK\x01\x00\x00\x00x")),
		"short format":     waveChunks(waveChunk("fmt ", []byte{1})),
		"extension":        waveChunks(waveChunk("fmt ", append(append([]byte{}, format...), 1, 0))),
		"trailing bytes":   append(append([]byte{}, valid...), 0, 0),
		"concatenated WAV": append(append([]byte{}, valid...), valid...),
	}
	for _, change := range []struct {
		name   string
		offset int
		value  byte
	}{
		{"RIFX", 3, 'X'}, {"not WAVE", 8, 'X'}, {"streaming size", 4, 255},
		{"float", 20, 3}, {"channels", 22, 2}, {"rate", 24, 1}, {"byte rate", 28, 1}, {"alignment", 32, 4}, {"bits", 34, 8},
	} {
		b := append([]byte{}, valid...)
		b[change.offset] = change.value
		bad[change.name] = b
	}
	for name, b := range bad {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeWAV(b); err == nil {
				t.Fatal("accepted invalid WAV")
			}
		})
	}
	for i := range valid {
		if _, err := DecodeWAV(valid[:i]); err == nil {
			t.Fatalf("accepted truncation at %d", i)
		}
	}
}

func FuzzDecodeWAV(f *testing.F) {
	f.Add(append(wavHeader(4), 0, 1, 2, 3))
	f.Add(waveChunks(waveChunk("JUNK", []byte{1})))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 65536 {
			t.Skip()
		}
		pcm, err := DecodeWAV(data)
		if err == nil && (len(pcm) == 0 || len(pcm)%2 != 0 || len(pcm) > len(data)) {
			t.Fatal("invalid decoded PCM")
		}
	})
}

func TestDecodeWAVSizeBoundary(t *testing.T) {
	data := make([]byte, MaxResponseBytes+2)
	copy(data, wavHeader(MaxResponseBytes-44))
	pcm, err := DecodeWAV(data[:MaxResponseBytes])
	if err != nil || len(pcm) != MaxResponseBytes-44 {
		t.Fatalf("exact bound: bytes=%d err=%v", len(pcm), err)
	}
	// Keep the oversized fixture structurally valid, so only the size guard rejects it.
	copy(data, wavHeader(MaxResponseBytes+2-44))
	if _, err := DecodeWAV(data); err == nil {
		t.Fatal("accepted response above limit")
	}
}
