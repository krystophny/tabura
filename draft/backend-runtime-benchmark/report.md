# Backend Runtime Benchmark (Draft)

- Timestamp (UTC): `2026-02-26T18:35:26.704647+00:00`
- Host: `mailuefterl`
- OS: `Linux 6.19.3-2-cachyos`
- Go: `go version go1.26.0-X:nodwarf5 linux/amd64`
- Node: `v22.22.0`
- Rust: `rustc 1.93.1 (01f6ddf75 2026-02-11) (Arch Linux rust 1:1.93.1-1.1)`
- Python: `3.14.3`
- Workload: `json decode/classify/hash/json encode`

## Method

- Synthetic backend-like CPU path per request: JSON decode, intent classification, token counting, SHA-256, JSON encode.
- Throughput measured on 300,000 ops; latency measured on 50,000 per-op samples.
- Compile/build time measured before execution for each runtime in this folder.
- Rust compile measured from a clean local `target/` in this benchmark folder; Go/TS/Python use fresh local build artifacts.

## Results

| Runtime | Compile Time (s) | Throughput (ops/s) | p50 (us) | p95 (us) | p99 (us) | Throughput vs Go | p95 vs Go |
|---|---:|---:|---:|---:|---:|---:|---:|
| go | 0.305 | 98105.31 | 8.055 | 12.173 | 16.551 | 1.00x | 1.00x |
| python | 0.077 | 87261.61 | 8.285 | 13.426 | 14.989 | 0.89x | 1.10x |
| typescript-node | 1.107 | 205121.75 | 5.550 | 7.133 | 7.975 | 2.09x | 0.59x |
| rust | 7.539 | 503096.75 | 2.846 | 3.777 | 4.018 | 5.13x | 0.31x |

## Porting Decision Notes

- Primary target for a Go port replacement should beat Go on both p95 latency and throughput, or provide clear non-performance benefits.
- Compile-time impact matters for dev loop and CI cost.

- Highest throughput in this run: `rust`.
- Recommendation: **Evaluate `rust` for a focused prototype.**
- It materially outperformed Go on throughput and p95 latency in this synthetic workload.
- Compile-time cost is significantly higher than Go for at least one high-performance alternative.

## Caveats

- This is a synthetic microbenchmark, not an end-to-end benchmark with WebSocket I/O, disk I/O, or SQLite contention.
- Use this as a directional signal; confirm with production-like load tests before committing to a full port.

