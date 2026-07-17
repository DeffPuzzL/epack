# 线格式

[English](WIRE_FORMAT.md) | 中文

本文描述 **当前** epack 线上字节布局。全程小端（little-endian）。

类型判别使用 Go `reflect.Kind`，存放在 head 的低 5 位（`kind & 0x1F`）。

## 值布局

每个编码值形如：

```
[ head ][ payload… ]
```

复合类型的子值同样遵循该模式。

## Head

两种 head，由 size 字段 `s` 决定：

| 形式 | 条件 | 长度 |
|------|------|------|
| 短头（2 字节） | `s < 0x0400`（1024） | 2 |
| 长头（8 字节） | 其他 | 8 |

### 短头（2 字节，小端 `uint16`）

解码：

- `kind = head & 0x1F`
- `size = ((head >> 5) & 0x3) << 8 | (head >> 8)`  
  即首字节 bit5–6 为 size 高 2 位，第二字节为 size 低 8 位。
- 首字节 bit7 为 **0**（与长头区分）

`size` 含义随类型而变（字符串/数字为字节长度，复合类型为元素/条目/字段数等）。

### 长头（8 字节，小端 `uint64`）

编码：

```
value = (s << 8) | (0x80 | (t & 0x1F))
```

解码：

- 首字节 bit7 = **1**
- `kind = head & 0x1F`
- `size = head >> 8`（56 位 size）

## 数值（含 bool）

```
[ head(kind, sizeof) ][ LE payload ]
```

载荷宽度：

| Kind | Payload |
|------|---------|
| bool | 1 字节（`0` / `1`） |
| int8 / uint8 | 1 字节 |
| int16 / uint16 | 2 字节 LE |
| int32 / uint32 / float32 | 4 字节 LE |
| int64 / uint64 / float64 | 8 字节 LE |
| int / uint | 按本机 `sizeof` 宽度写出的 LE 整数 |

浮点为 IEEE-754 位型的小端存储。

## 字符串

```
[ head(String, byte_len) ][ 原始 UTF-8 字节… ]
```

## time.Time

走数值路径，载荷为 `UnixNano()`（`int64`）。时区 / Location 不单独编码。

## 指针

- 非 nil：编码元素值。
- nil：`kind=Pointer` 且 size `0` 的哨兵头（与当前实现对齐）。

## Interface

- nil：按实现写入 interface 空语义头。
- 非 nil：编码动态元素值。

## Slice

```
[ head(Slice, len) ][ body ]
```

### 数字元素切片（紧凑）

元素为受支持的数字/bool 时，body 以：

```
0xFF | elem_kind | payload…
```

开头：

- `0xFF` 为 `SIMPLE_NUMBER` 标记
- `elem_kind` 一字节（`reflect.Kind`）
- `payload` 为 `len` 个紧凑 LE 元素载荷（小端主机上可能整段拷贝）

### 其他元素类型

body 为 `len` 个完整元素编码（各自含 head+payload）的拼接。

## Array

与 slice 类似：head 携带元素个数，随后各元素；解码时固定数组长度必须匹配。

## Map

```
[ head(Map, entry_count) ][ key₀ ][ val₀ ][ key₁ ][ val₁ ]…
```

key/value 均为完整编码值。顺序不保证，取决于 `MapKeys()`。

## Struct

```
[ head(Struct, field_count) ][ field₀ ][ field₁ ]…
```

- `field_count` 为编码器对该值写入的 `NumField()`。
- 字段载荷按 **`epack` tag 顺序**（unit 表顺序）写出，而非声明顺序。
- 仅带有效 tag 的导出字段进入 unit 表。

Tag `N` 对应 unit 表下标 `N-1`。不建议留空洞；越界 tag 在构建计划时会报错。

## 兼容性

1. 已发布 schema 的 tag 序号与字段类型应视为冻结。
2. 跨语言实现须遵循本文 head/payload 与小端规则。
3. 线上不包含 JSON 式字段名。

## 相关文档

- [INTRODUCTION_ZH_CN.md](INTRODUCTION_ZH_CN.md)
- [USAGE_ZH_CN.md](USAGE_ZH_CN.md)
