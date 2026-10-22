package providers

import (
	"encoding/binary"
	"fmt"
)

// DecodeWAV extracts 24 kHz mono S16LE PCM from a complete, bounded RIFF file.
// Unknown chunks are skipped with RIFF word padding. Streaming sizes, multiple
// data chunks and compressed/extensible formats are intentionally unsupported.
// The returned PCM aliases data.
func DecodeWAV(data []byte) ([]byte, error) {
	if len(data) > MaxResponseBytes || len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, fmt.Errorf("expected a bounded RIFF/WAVE response")
	}
	if uint64(binary.LittleEndian.Uint32(data[4:8]))+8 != uint64(len(data)) {
		return nil, fmt.Errorf("WAV length does not match RIFF size")
	}
	var pcm []byte
	var haveFormat, haveData bool
	for rest := data[12:]; len(rest) > 0; {
		if len(rest) < 8 {
			return nil, fmt.Errorf("truncated WAV chunk header")
		}
		size := uint64(binary.LittleEndian.Uint32(rest[4:8]))
		padded := size + size%2
		if padded > uint64(len(rest)-8) {
			return nil, fmt.Errorf("truncated WAV chunk")
		}
		chunk := rest[8 : 8+int(size)]
		switch string(rest[:4]) {
		case "fmt ":
			if haveFormat {
				return nil, fmt.Errorf("duplicate WAV format chunk")
			}
			haveFormat = true
			if err := checkWAVFormat(chunk); err != nil {
				return nil, err
			}
		case "data":
			if haveData {
				return nil, fmt.Errorf("duplicate WAV data chunk")
			}
			haveData = true
			pcm = chunk
		}
		rest = rest[8+int(padded):]
	}
	if !haveFormat || !haveData || len(pcm) == 0 || len(pcm)%2 != 0 {
		return nil, fmt.Errorf("WAV needs a format chunk and nonempty 16-bit PCM data")
	}
	return pcm, nil
}

func checkWAVFormat(b []byte) error {
	if len(b) != 16 && (len(b) != 18 || binary.LittleEndian.Uint16(b[16:18]) != 0) {
		return fmt.Errorf("unsupported WAV format chunk")
	}
	if binary.LittleEndian.Uint16(b) != 1 || binary.LittleEndian.Uint16(b[2:]) != 1 ||
		binary.LittleEndian.Uint32(b[4:]) != SampleRate || binary.LittleEndian.Uint32(b[8:]) != SampleRate*2 ||
		binary.LittleEndian.Uint16(b[12:]) != 2 || binary.LittleEndian.Uint16(b[14:]) != 16 {
		return fmt.Errorf("expected WAV PCM1, mono, 24000 Hz, 16-bit, block alignment 2 and byte rate 48000")
	}
	return nil
}
