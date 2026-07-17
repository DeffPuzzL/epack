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

可选辅助：`SetBufferMalloc`（缓冲池大小）。

内部类型（`config`、`unit` 等）与辅助函数为**未导出**，不属于公开 API；业务代码只需使用上述函数。

详见 [docs/USAGE_ZH_CN.md](docs/USAGE_ZH_CN.md)。

## 性能测试

同一复杂 payload（`BenchPayload`）下，epack 与 **protobuf**、**msgpack**、[sonic](https://github.com/bytedance/sonic)、`encoding/json` 对比（体积 / Marshal / Unmarshal / RoundTrip / 内存）：

![epack vs protobuf vs msgpack vs sonic vs JSON](docs/images/image.png)

> 环境：`darwin/arm64`（Apple M1 Pro），Go 1.26.1，`-count=3` 中位数。protobuf 为 codegen，其余为运行时绑定。详见 [docs/BENCHMARK_ZH_CN.md](docs/BENCHMARK_ZH_CN.md)。

仓库内 epack vs JSON：

```bash
go test -bench='Benchmark(Marshal|Unmarshal|RoundTrip)_(Epack|JSON)' -benchmem .
```

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

嵌套结构体、map、指针、`time.Time` 等见 [docs/USAGE_ZH_CN.md](docs/USAGE_ZH_CN.md)。

## 文档索引

| 文档 | 说明 |
|------|------|
| [INTRODUCTION](docs/INTRODUCTION_ZH_CN.md) | 设计与内部原理 |
| [USAGE](docs/USAGE_ZH_CN.md) | 详细用法 |
| [WIRE_FORMAT](docs/WIRE_FORMAT_ZH_CN.md) | 二进制线格式 |
| [BENCHMARK](docs/BENCHMARK_ZH_CN.md) | 详细性能对比 |

## 贡献

见 [CONTRIBUTING.md](CONTRIBUTING.md)。请同时阅读 [行为准则](CODE_OF_CONDUCT.md) 与 [安全策略](SECURITY.md)。

## 协议

[MIT](LICENSE)
