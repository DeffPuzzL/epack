# Wire Format

English | [中文](WIRE_FORMAT_ZH_CN.md)

This document describes the **current** epack on-wire layout. The format is little-endian throughout.

Type discriminators reuse Go `reflect.Kind` values in the low 5 bits of the head (`kind & 0x1F`).

## Value layout

Every encoded value is:

```
[ head ][ payload… ]
```

Composite values nest the same pattern for children.

## Head

Two head forms exist. Choice depends on the size field `s`:

| Form | When | Length |
|------|------|--------|
| Short (2-byte) | `s < 0x0400` (1024) | 2 bytes |
| Long (8-byte) | otherwise | 8 bytes |

### Short head (2 bytes, little-endian `uint16`)

Decoded as:

- `kind = head & 0x1F`
- `size = ((head >> 5) & 0x3) << 8 | (head >> 8)`  
  i.e. bit5–6 of the first byte are the high 2 bits of size; the second byte is the low 8 bits of size.
- bit7 of the first byte is **0** (distinguishes from long head)

Meaning of `size` depends on kind (byte length for strings/numbers, element/entry/field count for composites, etc.).

### Long head (8 bytes, little-endian `uint64`)

Encoded as:

```
value = (s << 8) | (0x80 | (t & 0x1F))
```

Decoded as:

- First byte has bit7 = **1**
- `kind = head & 0x1F`
- `size = head >> 8` (56-bit size)

## Numbers (incl. bool)

```
[ head(kind, sizeof) ][ LE payload ]
```

Payload widths:

| Kind | Payload |
|------|---------|
| bool | 1 byte (`0` / `1`) |
| int8 / uint8 | 1 byte |
| int16 / uint16 | 2 bytes LE |
| int32 / uint32 / float32 | 4 bytes LE |
| int64 / uint64 / float64 | 8 bytes LE |
| int / uint | host `sizeof` written as LE integer of that width |

Floats use IEEE-754 bit patterns stored little-endian.

## String

```
[ head(String, byte_len) ][ raw UTF-8 bytes… ]
```

## time.Time

Encoded via the numeric path as `UnixNano()` (`int64` payload). Time zone / location is not preserved as a separate field.

## Pointer

- Non-nil: encode the element value (no extra envelope beyond the element encoding used by the pointer encoder path).
- Nil pointer: a head with `kind=Pointer` and size `0` (empty pointer sentinel used by the implementation).

## Interface

- Nil interface: interface head with empty payload semantics used by the encoder.
- Non-nil: encode the dynamic element value.

## Slice

```
[ head(Slice, len) ][ body ]
```

### Numeric element slices (compact)

When element kind is a supported number/bool kind, body starts with:

```
0xFF | elem_kind | payload…
```

- `0xFF` is the numeric-slice compact marker
- `elem_kind` is one byte (`reflect.Kind`)
- `payload` is `len` tightly packed LE element payloads (and may be bulk-copied on LE hosts)

### Other element types

Body is `len` concatenated element encodings (each with its own head+payload as usual).

## Array

Same idea as slice: head carries element count, then each element (fixed Go array length must match on decode).

## Map

```
[ head(Map, entry_count) ][ key₀ ][ val₀ ][ key₁ ][ val₁ ]…
```

Keys and values are fully encoded values. Iteration order is not specified beyond Go's `MapKeys()`.

## Struct

```
[ head(Struct, field_count) ][ field₀ ][ field₁ ]…
```

- `field_count` is the Go `NumField()` count written by the encoder for that value.
- Field payloads are emitted in **`epack` tag order** (unit table order), not declaration order.
- Only exported fields with valid tags participate in the unit table.

Tag `N` selects slot `N-1` in the unit table. Gaps are not recommended; out-of-range tags error at plan build time.

## Compatibility notes

1. Treat tag indices and field types of a published schema as frozen.
2. Cross-language implementations must follow this head/payload layout and LE rule.
3. Do not assume JSON field names appear on the wire — epack does not encode field names.

## Related

- [INTRODUCTION.md](INTRODUCTION.md)
- [USAGE.md](USAGE.md)
