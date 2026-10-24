# 音频输出

[English](audio-output.md) · [教程](tutorial.zh-cn.md) · [厂商接入](providers.zh-cn.md)

`synth --out` / `-o` 按扩展名选择最终文件格式。省略时，`script.tbx` 会在脚本所在目录生成 `script.wav`。WAV 由原生代码写入，不需要 FFmpeg。其他格式要求 `PATH` 中有可信的 `ffmpeg` 可执行文件；Timbrex 不会自动下载或安装它。

## 格式与固定预设

| 扩展名 | 最终编码 | 实现与官方参考 |
| --- | --- | --- |
| `.wav` | 单声道、有符号 16 位小端 PCM | Timbrex 原生 WAV 写入器；真实 TTS 为 24 kHz，stub 测试音为 8 kHz |
| `.mp3` | MP3，请求 96 kbit/s CBR | FFmpeg [`libmp3lame`](https://ffmpeg.org/ffmpeg-codecs.html#libmp3lame) 与 [MP3 封装器](https://ffmpeg.org/ffmpeg-formats.html#mp3) |
| `.flac` | 无损 FLAC，压缩级别 5 | FFmpeg [`flac`](https://ffmpeg.org/ffmpeg-codecs.html#flac-2) |
| `.opus` | Ogg Opus，请求 48 kbit/s VBR，`audio` 应用模式 | FFmpeg [`libopus`](https://ffmpeg.org/ffmpeg-codecs.html#libopus-1) |
| `.aac` | ADTS 中的 AAC-LC，请求 96 kbit/s | FFmpeg [`aac`](https://ffmpeg.org/ffmpeg-codecs.html#aac) 与 [ADTS 封装器](https://ffmpeg.org/ffmpeg-formats.html#adts)，不是 MP4/M4A |

这些是固定导出预设，不是厂商请求选项。请求的码率不保证等于文件实测码率，短音频或 8 kHz stub 尤其如此；FFmpeg 可能按编码器限制调整。没有原始音频输出、码率或输出采样率参数。

真实适配器仍请求并拼接 24 kHz、单声道的 S16LE 音频；stub 仍生成 8 kHz 测试音。导出不会向厂商请求另一种原生格式，也不会提升源音频保真度。即使源音频为 24 kHz，Opus 工具也可能报告 48 kHz 解码时钟；[Opus 时间戳使用 48 kHz](https://opus-codec.org/docs/opusfile_api-0.12/group__stream__info.html)。ADTS AAC 存在[编码延迟和填充](https://ffmpeg.org/doxygen/trunk/structAVCodecContext.html#a8f95550ce04f236e9915516d04d3d1ab)，因此解码采样数与报告时长不一定与源音频精确相同。需要逐采样保留 PCM 时，请选 WAV 或 FLAC。

## 用法

保留默认的无额外依赖路径：

```bash
./timbrex synth examples/hello.tbx --provider stub -o tone.wav
```

已安装可信 FFmpeg 时，相同的离线测试也可验证导出：

```bash
./timbrex synth examples/hello.tbx --provider stub -o tone.mp3
```

生成已获授权的真实语音时，使用选定厂商，只需改变输出扩展名。每次运行都会重新合成；没有缓存或转换现有文件的命令。父目录需要事先存在，输出不能与源脚本为同一文件。不支持的扩展名和无效路径会在本地报错。

## 本地检查与发布

在任何厂商网络请求前，Timbrex 会检查目标路径，并用一个私有的微小 24 kHz 单声道 WAV 测试所选 FFmpeg 路径。检查会实际运行 WAV 解封装器、PCM 解码器、所选编码器和输出封装器，而不只是查看版本或编码器列表。组件缺失或不兼容会在本地停止。检查超时为十秒，不会向厂商发送脚本文本。检查通过不保证之后的完整导出一定成功。原有的 1 GiB PCM 限制保持不变，完成后的编码文件若超过 1 GiB 也会被拒绝。私有 WAV 与编码文件在发布前同时存在，因此磁盘需要容纳两者；目前没有统一的临时磁盘空间预算。

合成先在私有位置完成整个 WAV，包含按顺序排列的语音与本地停顿；随后 FFmpeg 只转码一次，将结果写入目标旁私有目录中的可寻址文件。可寻址文件使 [MP3 封装器能够写入时长和延迟元数据](https://ffmpeg.org/ffmpeg-formats.html#mp3)。Timbrex 直接调用 FFmpeg，不经过 shell，使用固定参数、不含厂商凭证的最小环境，以及仅允许本地文件的[协议白名单](https://ffmpeg.org/ffmpeg-protocols.html#Protocol-Options)。输入、输出与编码器参数语义见 [FFmpeg 命令行参考](https://ffmpeg.org/ffmpeg.html)。

只有成功后才通过重命名替换目标。发布前失败或取消，会清理临时输出并保留旧目标；已经发布的成功结果不会回滚。在 POSIX 系统上，文件使用私有的 `0600` 权限，并受 umask 约束。这不保证崩溃持久性或 `fsync`，重命名行为取决于操作系统与文件系统。导出失败无法撤销已经发生的 TTS 费用。导出不增加混音、重叠、背景音乐、流式输出、自动重试或厂商回退。
