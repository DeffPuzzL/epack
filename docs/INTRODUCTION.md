# Introduction

English | [中文](INTRODUCTION_ZH_CN.md)

epack is a Go library for binary serialization and deserialization. It binds values at runtime with reflection, caches per-type encode/decode plans, and writes a compact little-endian wire format.

## Why epack

- **Binary**: typically smaller and faster to parse than text formats such as JSON for homogeneous Go structs
- **No codegen**: call `Marshal` / `Unmarshal` directly
- **Stable field order**: `` `epack:"N"` `` tags define on-wire order so renaming Go fields does not reshuffle payload layout
- **Reusable plans**: `LoadTemplate` (and first-use caching) avoids rebuilding field plans on every call

epack is **not** a JSON library and is **not** wire-compatible with MessagePack, Protobuf, or sonic.

## Encode / decode path

1. `Marshal` / `Unmarshal` accept Go values (`interface{}`).
2. For structs, epack builds (or loads) a list of `Unit`s describing each tagged exported field and its encoder/decoder.
3. Units are stored in a process-wide `sync.Map` keyed by type string.
4. Encoding writes type heads + payloads into a pooled buffer, then copies out the result bytes.
5. Decoding consumes the buffer and requires the entire input to be consumed for top-level values (`isOver`).

```
value ──► type cache (units) ──► encoder/decoder ──► little-endian bytes
```

## Caching and buffers

- **Type cache**: first encode/decode of a struct type (or an explicit `LoadTemplate`) stores units for reuse.
- **Buffer pool**: encode buffers are obtained from `sync.Pool` and reset after use to reduce allocations.

## Endianness

The **wire format is always little-endian**, independent of host endianness. Host endianness may only affect optional bulk-copy optimizations for numeric slices on little-endian machines; the on-wire bytes remain LE.

## Tags and schema evolution

- Tags are 1-based indices into a dense unit table.
- Prefer contiguous tags (`1..N`) matching the number of exported serialized fields.
- Changing a published tag index or type of an existing field is a breaking wire change.
- Unexported fields are skipped.

## Notable behaviors

| Topic | Behavior |
|-------|----------|
| `Unmarshal` target | Must be a non-nil pointer |
| Nil top-level | `Marshal(nil)` errors |
| Trailing bytes | Top-level decode errors if unread bytes remain |
| `time.Time` | Encoded as `UnixNano` (int64); location may differ after round-trip |
| Nested `*struct` fields | Prefer value nested structs; pointer-to-struct nesting has sharp edges in the current encoder |
| Maps | Entry order follows `MapKeys()`; not sorted |

For APIs and examples see [USAGE.md](USAGE.md). For byte-level layout see [WIRE_FORMAT.md](WIRE_FORMAT.md).
