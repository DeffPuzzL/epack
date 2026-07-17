# epack

[English](README.md) | 中文

高性能的 Go 二进制序列化 / 反序列化库：运行时对象绑定、`epack` 字段顺序标签、模板缓存，无需代码生成。

## 环境要求

- Go：1.26+（以 `go.mod` 为准）
- OS：Linux / macOS / Windows
- Arch：amd64 / arm64（及其他 Go 工具链支持的平台）

## 特性

- 运行时绑定，无需 codegen
- 通过 `` `epack:"N"` `` 控制结构体字段序列化顺序
- `LoadTemplate` 预编译模板，加速重复编解码
- 缓冲池 + 固定小端线格式
- 支持基本类型、结构体、切片/数组、map、指针、`time.Time`、interface 等

## API

公开入口：

```go
func Marshal(obj interface{}) ([]byte, error)
func Unmarshal(buffer []byte, obj interface{}) error
func LoadTemplate(obj ...interface{}) error
```

详见源码与 [docs/USAGE_ZH_CN.md](docs/USAGE_ZH_CN.md)。

## 性能测试

同一复杂 payload（`BenchPayload`）下，epack 与 [sonic](https://github.com/bytedance/sonic)、`encoding/json` 对比：

![epack vs sonic vs encoding/json](docs/images/image.png)

> 数据环境：`darwin/arm64`（Apple M1 Pro）。sonic 与 JSON 载荷同为 JSON 文本；epack 为二进制线格式。请以本机复测为准。

仓库内 epack vs JSON 基准：

```bash
go test -bench='Benchmark(Marshal|Unmarshal|RoundTrip)_(Epack|JSON)' -benchmem .
```

详见 `complex_bench_test.go`。

## 原理

见 [docs/INTRODUCTION_ZH_CN.md](docs/INTRODUCTION_ZH_CN.md)。

## 快速使用

### Marshal / Unmarshal

```go
package main

import (
	"fmt"

	"github.com/DeffPuzzL/epack"
)

type Person struct {
	Name   string  `epack:"1"`
	Age    int     `epack:"2"`
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

`Unmarshal` 的目标必须是非 nil 指针。

### 结构体标签

导出字段建议使用连续的正整数 tag（从 1 开始）。序列化顺序由 tag 序号决定，而非源码字段声明顺序。

### LoadTemplate

对热点类型预编译编解码单元：

```go
_ = epack.LoadTemplate(Person{})
```

### 更多示例

- [docs/USAGE_ZH_CN.md](docs/USAGE_ZH_CN.md)
- 包内示例：`examples_test.go`、`examples/`

## 文档索引

| 文档 | 说明 |
|------|------|
| [INTRODUCTION](docs/INTRODUCTION_ZH_CN.md) | 设计与内部原理 |
| [USAGE](docs/USAGE_ZH_CN.md) | 详细用法 |
| [WIRE_FORMAT](docs/WIRE_FORMAT_ZH_CN.md) | 二进制线格式 |

## 贡献

见 [CONTRIBUTING.md](CONTRIBUTING.md)。请同时阅读 [行为准则](CODE_OF_CONDUCT.md) 与 [安全策略](SECURITY.md)。

## 协议

[MIT](LICENSE)
