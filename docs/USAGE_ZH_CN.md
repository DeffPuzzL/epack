# 使用指南

[English](USAGE.md) | 中文

## 安装

```bash
go get github.com/DeffPuzzL/epack
```

```go
import "github.com/DeffPuzzL/epack"
```

模块路径以 `go.mod` 为准（`github.com/DeffPuzzL/epack`）。

## 基本类型

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

支持 bool、整型、浮点、string、slice、array、map、pointer、interface、struct、`time.Time` 等。

## 结构体与 tag

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

规则：

- 仅序列化**导出**字段。
- Tag 为 **1-based** 序号；同一结构体建议连续编号。
- 线上顺序由 tag 决定，而非源码声明顺序。

## 嵌套结构体

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

嵌套优先使用**值类型**结构体。除非你已验证所需路径，否则不要依赖嵌套 `*Struct` 字段。

## 切片、map、指针、时间

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
	Content *string `epack:"2"` // 可为 nil
	Version *int    `epack:"3"`
}

type Event struct {
	Title     string    `epack:"1"`
	CreatedAt time.Time `epack:"2"`
}
```

`time.Time` 按 Unix 纳秒存储；往返后请用 `Time.Equal` 比较时刻。

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

`LoadTemplate` 仅接受结构体（或其指针）。对已缓存类型再次调用是空操作。

## 常见错误

| 情况 | 结果 |
|------|------|
| `Marshal(nil)` / typed nil 指针 | 报错 |
| `Unmarshal` 到非指针或 nil 指针 | 报错 |
| 缓冲区截断 | short-buffer / unexpected EOF 类错误 |
| 顶层值后仍有多余字节 | `invalid trailing data after top-level value` |
| 非法 / 越界 `epack` tag | 报错 |
| 不支持的类型 | `unsupported type` |

## 仓库内示例

- `examples_test.go`（包内）
- `examples/examples_test.go`（外部测试包）

```bash
go test -run Example ./...
```

## 相关文档

- [INTRODUCTION_ZH_CN.md](INTRODUCTION_ZH_CN.md)
- [WIRE_FORMAT_ZH_CN.md](WIRE_FORMAT_ZH_CN.md)
