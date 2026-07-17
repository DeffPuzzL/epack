# epack

English | [中文](README_ZH_CN.md)

A fast Go binary serialization & deserialization library with runtime object binding, field-order tags, and template caching — no code generation required.

## Requirement

- Go: 1.26+ (see `go.mod`)
- OS: Linux / macOS / Windows
- Arch: amd64 / arm64 (and other platforms supported by the Go toolchain)

## Features

- Runtime binding without code generation
- Struct field order controlled by `` `epack:"N"` `` tags
- Template cache via `LoadTemplate` for repeated encode/decode
- Buffer pool and little-endian wire format
- Supports primitives, structs, slices/arrays, maps, pointers, `time.Time`, and interfaces

## APIs

Public entry points:

```go
func Marshal(obj interface{}) ([]byte, error)
func Unmarshal(buffer []byte, obj interface{}) error
func LoadTemplate(obj ...interface{}) error
```

Optional helper: `SetBufferMalloc` (buffer pool size).

Internal types (`config`, `unit`, …) and helpers are **unexported** and not part of the public API — call sites should only use the functions above.

See [docs/USAGE.md](docs/USAGE.md) for details.

## Benchmarks

On the same complex payload (`BenchPayload`), epack is compared with **protobuf**, **msgpack**, [sonic](https://github.com/bytedance/sonic), and `encoding/json` (size / Marshal / Unmarshal / RoundTrip / memory):

![epack vs protobuf vs msgpack vs sonic vs JSON](docs/images/image.png)

> `darwin/arm64` (Apple M1 Pro), Go 1.26.1, `-count=3` median. protobuf uses codegen; others use runtime binding. Details: [docs/BENCHMARK.md](docs/BENCHMARK.md).

In-repo epack vs JSON:

```bash
go test -bench='Benchmark(Marshal|Unmarshal|RoundTrip)_(Epack|JSON)' -benchmem .
```

## How it works

See [docs/INTRODUCTION.md](docs/INTRODUCTION.md).

## Usage

### Marshal / Unmarshal

```go
package main

import (
	"fmt"

	"github.com/DeffPuzzL/epack"
)

type Person struct {
	Name string  `epack:"1"`
	Age  int     `epack:"2"`
	Height float64 `epack:"3"`
}

func main() {
	in := Person{Name: "Alice", Age: 30, Height: 1.68}
	data, err := epack.Marshal(in)
	if err != nil {
		panic(err)
	}

	var out Person
	if err := epack.Unmarshal(data, &out); err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", out)
}
```

`Unmarshal` requires a non-nil pointer destination.

### Struct tags

Exported fields should use contiguous positive integer tags (1-based). Serialization order follows the tag index, not the source field declaration order.

### LoadTemplate

Pre-build encoder/decoder units for hot types:

```go
_ = epack.LoadTemplate(Person{})
```

### More examples

See [docs/USAGE.md](docs/USAGE.md) for nested structs, maps, pointers, `time.Time`, and more.

## Documentation

| Doc | Description |
|-----|-------------|
| [INTRODUCTION](docs/INTRODUCTION.md) | Design & internals |
| [USAGE](docs/USAGE.md) | Detailed usage |
| [WIRE_FORMAT](docs/WIRE_FORMAT.md) | Binary wire format |
| [BENCHMARK](docs/BENCHMARK.md) | Detailed benchmarks |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Please also read our [Code of Conduct](CODE_OF_CONDUCT.md) and [Security policy](SECURITY.md).

## License

[MIT](LICENSE)
