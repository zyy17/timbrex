# Audio output

[简体中文](audio-output.zh-cn.md) · [Tutorial](tutorial.md) · [Provider setup](providers.md)

`synth --out` / `-o` selects the final file format by extension. Without it, `script.tbx` produces `script.wav` beside the script. WAV is native and requires no FFmpeg. The other formats require a trusted `ffmpeg` executable on `PATH`; Timbrex never downloads or installs it.

## Formats and fixed presets

| Extension | Final encoding | Implementation and official reference |
| --- | --- | --- |
| `.wav` | Mono signed 16-bit little-endian PCM | Timbrex's native WAV writer; 24 kHz for real TTS, 8 kHz for the stub test tone |
| `.mp3` | MP3, requested 96 kbit/s CBR | FFmpeg [`libmp3lame`](https://ffmpeg.org/ffmpeg-codecs.html#libmp3lame) and [MP3 muxer](https://ffmpeg.org/ffmpeg-formats.html#mp3) |
| `.flac` | Lossless FLAC, compression level 5 | FFmpeg [`flac`](https://ffmpeg.org/ffmpeg-codecs.html#flac-2) |
| `.opus` | Ogg Opus, requested 48 kbit/s VBR, `audio` application | FFmpeg [`libopus`](https://ffmpeg.org/ffmpeg-codecs.html#libopus-1) |
| `.aac` | AAC-LC in ADTS, requested 96 kbit/s | FFmpeg [`aac`](https://ffmpeg.org/ffmpeg-codecs.html#aac) and [ADTS muxer](https://ffmpeg.org/ffmpeg-formats.html#adts), not MP4/M4A |

These are fixed export presets, not vendor request options. Bitrate requests are not a promise of the measured file bitrate, especially for short clips or the 8 kHz stub. FFmpeg may adjust to codec limits. There are no raw-output, bitrate, or output-sample-rate flags.

Real adapters still request and compose 24 kHz mono S16LE audio; the stub still produces its 8 kHz tone. Export does not request a different native vendor format or improve source fidelity. Opus tools may report a 48 kHz decoding clock even for a 24 kHz source; [Opus timestamps use 48 kHz](https://opus-codec.org/docs/opusfile_api-0.12/group__stream__info.html). ADTS AAC has [codec delay/padding](https://ffmpeg.org/doxygen/trunk/structAVCodecContext.html#a8f95550ce04f236e9915516d04d3d1ab), so decoded sample count and reported duration need not exactly match the source. Use WAV or FLAC when sample-exact PCM preservation matters.

## Usage

Keep the default dependency-free path:

```bash
./timbrex synth examples/hello.tbx --provider stub -o tone.wav
```

With a trusted FFmpeg installation, the same offline test can exercise export:

```bash
./timbrex synth examples/hello.tbx --provider stub -o tone.mp3
```

For authorized speech generation, use the selected real provider and change only the output extension. Each invocation synthesizes again; there is no cache or existing-file conversion command. The parent directory must exist, and the output must not be the source script. Unsupported extensions and invalid paths fail locally.

## Local checks and publication

Before any provider network request, Timbrex validates the destination and tests the selected FFmpeg path using a tiny private 24 kHz mono WAV. It runs the actual WAV demuxer/PCM decoder, chosen encoder, and output muxer, not just a version or encoder-list check. Missing or incompatible components stop locally. The test has a ten-second timeout and does not send script text to a provider. Passing it does not guarantee that a later full export will succeed. The existing 1 GiB PCM limit remains, and a completed encoded file larger than 1 GiB is rejected. The private WAV and encoded file coexist until publication, so export needs disk space for both; there is no aggregate temporary-disk budget.

Synthesis completes the entire WAV privately, including ordered speech and local pauses. Only then does FFmpeg transcode it once into a seekable file in a private directory beside the destination. Seekability lets the [MP3 muxer write duration and delay metadata](https://ffmpeg.org/ffmpeg-formats.html#mp3). Timbrex invokes FFmpeg directly, not through a shell, with fixed arguments, a minimal environment without provider credentials, and a local-file-only [protocol whitelist](https://ffmpeg.org/ffmpeg-protocols.html#Protocol-Options). See the [FFmpeg command-line reference](https://ffmpeg.org/ffmpeg.html) for input/output and codec option semantics.

The destination is replaced by rename only after success. A failure or cancellation before publication removes temporary output and preserves the previous destination; success already published is not rolled back. Files use private `0600` permissions on POSIX, subject to umask. This is not a crash-durability or `fsync` guarantee; rename behavior depends on the OS and filesystem. A failed export cannot undo TTS charges already incurred. Export adds no mixing, overlap, background music, streaming, automatic retries, or provider fallback.
