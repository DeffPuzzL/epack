# Benchmarks

English | [中文](BENCHMARK_ZH_CN.md)

Numbers below were measured on **darwin/arm64 (Apple M1 Pro), Go 1.26.1**, using the same complex nested payload (`BenchPayload`: maps, slices, nested structs, `time.Time` / UnixNano, floats, bools).

Command shape: `go test -bench=... -benchmem -count=3` (median reported).

Libraries:

| Library | Role |
|---------|------|
| **epack** | binary, runtime reflection + template cache |
| **protobuf** | binary, generated code (`google.golang.org/protobuf`) |
| **msgpack** | binary, runtime (`github.com/vmihailenco/msgpack/v5`) |
| **sonic** | JSON text, JIT/SIMD (`github.com/bytedance/sonic`) |
| **encoding/json** | JSON text, stdlib |

> Fairness note: protobuf uses codegen; epack/msgpack/json/sonic bind at runtime. Protobuf timestamps are `int64` UnixNano to align with epack semantics.

## Summary chart

![five-way comparison](images/image.png)

## Payload size

| Format | Bytes | vs JSON |
|--------|------:|--------:|
| protobuf | 276 | 44.9% |
| epack | 373 | 60.7% |
| msgpack | 493 | 80.3% |
| sonic | 614 | 100% |
| JSON | 614 | 100% |

## Marshal (ns/op)

| Library | ns/op | vs JSON | B/op | allocs/op |
|---------|------:|--------:|-----:|----------:|
| epack | 865 | **2.39x** | 584 | 10 |
| protobuf | 1,071 | 1.93x | 432 | 13 |
| msgpack | 1,869 | 1.11x | 1,152 | 14 |
| JSON | 2,067 | 1.00x | 936 | 10 |
| sonic | 2,277 | 0.91x | 868 | 5 |

## Unmarshal (ns/op)

| Library | ns/op | vs JSON | B/op | allocs/op |
|---------|------:|--------:|-----:|----------:|
| epack | 1,423 | **5.40x** | 920 | 36 |
| protobuf | 1,498 | 5.13x | 1,208 | 39 |
| sonic | 1,855 | 4.14x | 1,678 | 19 |
| msgpack | 3,479 | 2.21x | 1,208 | 48 |
| JSON | 7,679 | 1.00x | 1,296 | 44 |

## RoundTrip (Marshal + Unmarshal)

| Library | ns/op | vs JSON | B/op | allocs/op |
|---------|------:|--------:|-----:|----------:|
| epack | 2,466 | **4.09x** | 1,836 | 47 |
| protobuf | 2,734 | 3.69x | 1,864 | 53 |
| sonic | 4,347 | 2.32x | 3,615 | 25 |
| msgpack | 5,562 | 1.81x | 2,674 | 63 |
| JSON | 10,082 | 1.00x | 2,563 | 55 |

## Takeaways

1. **Size**: protobuf smallest, then epack, then msgpack; JSON/sonic largest (text).
2. **Speed**: epack is fastest overall in this runtime-binding set; close to protobuf despite no codegen.
3. **vs JSON**: epack RoundTrip ~4x faster; Unmarshal ~5.4x faster.
4. **vs sonic**: epack RoundTrip ~1.8x faster; sonic wins on allocs/op for some paths.
5. **vs msgpack**: epack is both smaller and faster on this payload.
6. Results are machine-specific — re-run locally before citing.

## In-repo epack vs JSON

```bash
go test -bench='Benchmark(Marshal|Unmarshal|RoundTrip)_(Epack|JSON)' -benchmem .
```
