# cpubench

[中文](#中文) | [English](#english)

## 中文

`cpubench` 是一个用 Go 编写的跨平台 CPU Benchmark 命令行工具。项目只依赖 Go 标准库，便于交叉编译到 Linux、macOS、Windows，以及 x86/x64、ARM、mipsle 等架构。

它适合在相同操作系统、相同运行环境下做相对性能对比。它不是硬件计数器工具，也不会直接读取真实功耗；功耗效率项是基于 CPU 时间的代理指标。

### 使用方式

从源码运行：

```bash
go run ./cmd/cpubench -help
go run ./cmd/cpubench -list
go run ./cmd/cpubench -content int-crypto,hash,mem-stream -time 3s -cores 4
go run ./cmd/cpubench -content all -time 1s -cores 8 -json
```

使用 Release 包：

```bash
./cpubench -help
./cpubench -content all -time 3s -cores 8
./cpubench -content hash,mem-stream,render -time 5s -cores 4 -json
```

Windows:

```powershell
.\cpubench.exe -help
.\cpubench.exe -content all -time 3s -cores 8
```

### 参数

- `-content`: 指定要运行的测试子项，多个 ID 用英文逗号分隔，也可以使用 `all`。
- `-time`: 每个测试子项的运行时间，例如 `500ms`、`3s`、`1m`。
- `-cores`: 多核测试使用的 worker 数量。单核测试会固定使用 1 个 worker。
- `-json`: 输出机器可读的 JSON。
- `-list`: 打印所有可用测试 ID。
- `-help`: 打印详细帮助，包括每个测试子项的测试重点和应用场景。

### 测试内容

执行 `cpubench -help` 可以查看完整、权威的测试项列表。当前覆盖：

- 单核整数性能：软件加密风格整数混合、哈希、压缩模型。
- 单核浮点性能：科学计算、3D 向量运算、FFT。
- 多核并行性能：渲染模型、并行编译模型、批量数据处理。
- 内存与缓存：STREAM 风格带宽、Pointer Chasing 延迟、缓存层级、跨核缓存一致性。
- SIMD/向量基线：可被编译器向量化的连续数组计算，不强制绑定 SSE/AVX/NEON/SVE。
- 分支预测与乱序执行：高分支密度代码、长依赖链。
- 功耗效率代理：单位 CPU 时间的综合吞吐分数，不代表真实每瓦性能。

### 输出示例

```text
ID          Dimension                     Workers  Score   Unit       Duration
hash        single-core integer           1        687.05  MiB/s      0.01s
mem-stream  memory bandwidth and latency  2        140.70  GiB/s      0.01s
render      multi-core parallel           2        540.60  Mpixels/s  0.01s
```

### 交叉编译

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dist/cpubench-linux-amd64 ./cmd/cpubench
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o dist/cpubench-linux-arm64 ./cmd/cpubench
GOOS=linux GOARCH=mipsle GOMIPS=softfloat CGO_ENABLED=0 go build -o dist/cpubench-linux-mipsle ./cmd/cpubench
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o dist/cpubench-windows-amd64.exe ./cmd/cpubench
```

### 发布

GitHub Actions 会在推送 `v*` tag 时自动构建 Release 包，并上传 Linux、macOS、Windows 的压缩包和 `checksums.txt`。

```bash
git tag v0.1.0
git push origin v0.1.0
```

### 开发

```bash
go test ./...
go test ./... -cover
go vet ./...
```

## English

`cpubench` is a portable CPU benchmark CLI written in Go. It uses only the Go standard library, making it straightforward to cross-compile for Linux, macOS, Windows, x86/x64, ARM, and mipsle targets.

The benchmark is intended for relative comparisons under the same OS and runtime settings. It is not a hardware-counter profiler, and it does not read real power telemetry; the efficiency item is a CPU-time based proxy.

### Usage

Run from source:

```bash
go run ./cmd/cpubench -help
go run ./cmd/cpubench -list
go run ./cmd/cpubench -content int-crypto,hash,mem-stream -time 3s -cores 4
go run ./cmd/cpubench -content all -time 1s -cores 8 -json
```

Use a Release package:

```bash
./cpubench -help
./cpubench -content all -time 3s -cores 8
./cpubench -content hash,mem-stream,render -time 5s -cores 4 -json
```

Windows:

```powershell
.\cpubench.exe -help
.\cpubench.exe -content all -time 3s -cores 8
```

### Options

- `-content`: benchmark IDs to run, separated by commas, or `all`.
- `-time`: duration per selected benchmark item, such as `500ms`, `3s`, or `1m`.
- `-cores`: worker count for parallel benchmark items. Single-core items intentionally use one worker.
- `-json`: emit machine-readable JSON.
- `-list`: print available benchmark IDs.
- `-help`: print detailed help, including each item’s focus and application scenario.

### Benchmark Items

Run `cpubench -help` for the complete authoritative list. Current coverage includes:

- Single-core integer: software crypto-style integer mixing, hashing, compression model.
- Single-core floating point: scientific math, 3D vector math, FFT.
- Multi-core parallel: rendering model, parallel compile model, batch data processing.
- Memory and cache: STREAM-style bandwidth, pointer chasing latency, cache hierarchy, cross-core coherency.
- SIMD/vector baseline: compiler-vectorizable contiguous-array computation without forcing SSE/AVX/NEON/SVE.
- Branch prediction and out-of-order execution: branch-heavy code and long dependency chains.
- Efficiency proxy: combined throughput per CPU-time score, not real performance per watt.

### Output Example

```text
ID          Dimension                     Workers  Score   Unit       Duration
hash        single-core integer           1        687.05  MiB/s      0.01s
mem-stream  memory bandwidth and latency  2        140.70  GiB/s      0.01s
render      multi-core parallel           2        540.60  Mpixels/s  0.01s
```

### Cross Compile

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dist/cpubench-linux-amd64 ./cmd/cpubench
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o dist/cpubench-linux-arm64 ./cmd/cpubench
GOOS=linux GOARCH=mipsle GOMIPS=softfloat CGO_ENABLED=0 go build -o dist/cpubench-linux-mipsle ./cmd/cpubench
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o dist/cpubench-windows-amd64.exe ./cmd/cpubench
```

### Release

GitHub Actions automatically builds Release packages when a `v*` tag is pushed. The Release includes Linux, macOS, Windows archives and `checksums.txt`.

```bash
git tag v0.1.0
git push origin v0.1.0
```

### Development

```bash
go test ./...
go test ./... -cover
go vet ./...
```
