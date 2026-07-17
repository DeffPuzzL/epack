# Usage

English | [中文](USAGE_ZH_CN.md)

## Install

```bash
go get github.com/DeffPuzzL/epack
```

```go
import "github.com/DeffPuzzL/epack"
```

Module path follows `go.mod` (`github.com/DeffPuzzL/epack`).

## Primitives

```go
b, err := epack.Marshal(true)
var out bool
err = epack.Unmarshal(b, &out)

b, err = epack.Marshal("hello")
var s string
err = epack.Unmarshal(b, &s)

b, err = epack.Marshal(int(42))
var n int
err = epack.Unmarshal(b, &n)
```

Supported kinds include bool, integers, floats, string, slice, array, map, pointer, interface, struct, and `time.Time`.

## Structs and tags

```go
type Person struct {
	Name    string   `epack:"1"`
	Age     int      `epack:"2"`
	Height  float64  `epack:"3"`
	Hobbies []string `epack:"4"`
}

p := Person{Name: "Alice", Age: 30, Height: 1.68, Hobbies: []string{"reading"}}
data, err := epack.Marshal(p)

var p2 Person
err = epack.Unmarshal(data, &p2)
```

Rules:

- Only **exported** fields are serialized.
- Tag values are **1-based** indices; keep them contiguous for a given struct.
- On-wire order follows tag index, not declaration order.

## Nested structs

```go
type Address struct {
	City    string `epack:"1"`
	ZipCode string `epack:"2"`
}

type Employee struct {
	Name    string  `epack:"1"`
	Address Address `epack:"2"`
	Salary  float64 `epack:"3"`
}
```

Prefer **value** nested structs. Avoid relying on nested `*Struct` fields unless you have verified the path you need.

## Slices, maps, pointers, time

```go
type Team struct {
	Name    string   `epack:"1"`
	Members []string `epack:"2"`
	Scores  []int    `epack:"3"`
}

type Config struct {
	Name       string            `epack:"1"`
	Properties map[string]string `epack:"2"`
}

type Document struct {
	Title   string  `epack:"1"`
	Content *string `epack:"2"` // may be nil
	Version *int    `epack:"3"`
}

type Event struct {
	Title     string    `epack:"1"`
	CreatedAt time.Time `epack:"2"`
}
```

`time.Time` is stored as Unix nanoseconds; compare instants with `Time.Equal` after round-trip.

## LoadTemplate

```go
type Item struct {
	ID    int     `epack:"1"`
	Price float64 `epack:"2"`
}

if err := epack.LoadTemplate(Item{}); err != nil {
	// handle
}

for i := 0; i < 1000; i++ {
	data, _ := epack.Marshal(Item{ID: i, Price: 1.5})
	var item Item
	_ = epack.Unmarshal(data, &item)
}
```

`LoadTemplate` only accepts struct values (or pointers to structs). Calling it again for a cached type is a no-op.

## Errors (common cases)

| Situation | Result |
|-----------|--------|
| `Marshal(nil)` / typed nil pointer | error |
| `Unmarshal` into non-pointer or nil pointer | error |
| Truncated buffer | short-buffer / unexpected EOF style errors |
| Trailing junk after top-level value | `invalid trailing data after top-level value` |
| Bad / out-of-range `epack` tag index | error |
| Unsupported kind | `unsupported type` |

## Examples in-repo

Runnable examples:

- `examples_test.go` (package `epack`)
- `examples/examples_test.go` (external test package)

```bash
go test -run Example ./...
```

## Related docs

- [INTRODUCTION.md](INTRODUCTION.md)
- [WIRE_FORMAT.md](WIRE_FORMAT.md)
