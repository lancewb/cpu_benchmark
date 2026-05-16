# cpubench

`cpubench` is a portable CPU benchmark CLI written in Go. It uses only the Go standard library, so it can be cross-compiled for Linux, macOS, Windows, x86/x64, ARM, and mipsle targets.

The benchmark is intended for relative comparisons under the same OS and runtime settings. It is not a replacement for hardware-counter based tools.

## Usage

```bash
go run ./cmd/cpubench -help
go run ./cmd/cpubench -list
go run ./cmd/cpubench -content int-crypto,hash,mem-stream -time 3s -cores 4
go run ./cmd/cpubench -content all -time 1s -cores 8 -json
```

Options:

- `-content`: comma-separated benchmark IDs, or `all`.
- `-time`: duration per selected benchmark item, such as `500ms`, `3s`, or `1m`.
- `-cores`: worker count for parallel benchmark items. Single-core items intentionally use one worker.
- `-json`: emit machine-readable JSON.
- `-help`: print every item, its focus, and application scenario.

## Benchmark Items

Run `cpubench -help` for the authoritative list. Current categories cover:

- single-core integer: software crypto-style mixing, hashing, compression-style match finding
- single-core floating point: scientific math, 3D vector math, FFT
- multi-core: rendering, compile-like scanning/parsing, batch processing
- memory and cache: STREAM-style bandwidth, pointer chasing latency, cache hierarchy, coherency
- vector baseline: compiler-vectorizable numeric loops
- branch and OoO: branch-heavy code, dependency chains
- efficiency proxy: score per CPU-time proxy, not true watts

## Cross Compile Examples

```bash
GOOS=linux GOARCH=amd64 go build -o dist/cpubench-linux-amd64 ./cmd/cpubench
GOOS=linux GOARCH=arm64 go build -o dist/cpubench-linux-arm64 ./cmd/cpubench
GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -o dist/cpubench-linux-mipsle ./cmd/cpubench
GOOS=windows GOARCH=amd64 go build -o dist/cpubench-windows-amd64.exe ./cmd/cpubench
```

## Development

```bash
go test ./...
go test ./... -cover
go vet ./...
```
