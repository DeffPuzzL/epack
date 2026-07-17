# 原理介绍

[English](INTRODUCTION.md) | 中文

epack 是一个 Go 二进制序列化 / 反序列化库。它通过反射在运行时绑定值，缓存每种类型的编解码计划，并写出紧凑的小端线格式。

## 为什么用 epack

- **二进制**：对同构 Go 结构体，通常比 JSON 等文本格式更紧凑、解析更快
- **无需 codegen**：直接调用 `Marshal` / `Unmarshal`
- **稳定字段序**：`` `epack:"N"` `` 决定线上顺序，重命名 Go 字段不会打乱载荷布局
- **可复用计划**：`LoadTemplate`（以及首次使用时的缓存）避免每次重建字段计划

epack **不是** JSON 库，也与 MessagePack、Protobuf、sonic **不兼容**。

## 编解码路径

1. `Marshal` / `Unmarshal` 接受 Go 值（`interface{}`）。
2. 对结构体，epack 构建（或加载）一组字段单元，描述每个带 tag 的导出字段及其编解码器。
3. 这些计划缓存在进程级 `sync.Map` 中，按类型字符串索引。
4. 编码将类型头 + 载荷写入池化缓冲区，再拷贝为结果字节。
5. 解码消费缓冲区；顶层值要求输入被完整消费（`isOver`）。

```
value ──► 类型缓存 (units) ──► encoder/decoder ──► 小端字节流
```

## 缓存与缓冲

- **类型缓存**：某结构体类型首次编解码（或显式 `LoadTemplate`）后复用 units。
- **缓冲池**：编码缓冲来自 `sync.Pool`，用后重置以降低分配。

## 端序

**线格式始终为小端**，与本机端序无关。本机端序仅可能影响数字切片在小端机器上的整段拷贝优化；线上字节仍为 LE。

## Tag 与演进

- Tag 为 1-based，对应稠密 unit 表下标。
- 建议使用连续 tag（`1..N`），与可序列化导出字段数量一致。
- 修改已发布字段的 tag 序号或类型属于破坏性变更。
- 未导出字段会被跳过。

## 重要行为

| 主题 | 行为 |
|------|------|
| `Unmarshal` 目标 | 必须是非 nil 指针 |
| 顶层 nil | `Marshal(nil)` 报错 |
| 尾部多余字节 | 顶层解码若有未读字节则报错 |
| `time.Time` | 按 `UnixNano`（int64）编码；往返后 Location 可能不同 |
| 嵌套 `*struct` | 优先使用值类型嵌套；指针嵌套结构体在当前实现中有边界问题 |
| map | 遍历顺序跟随 `MapKeys()`，不排序 |

API 与示例见 [USAGE_ZH_CN.md](USAGE_ZH_CN.md)。字节布局见 [WIRE_FORMAT_ZH_CN.md](WIRE_FORMAT_ZH_CN.md)。
