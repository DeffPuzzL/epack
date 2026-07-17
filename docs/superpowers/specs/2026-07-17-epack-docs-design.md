# epack 文档补充设计

**日期：** 2026-07-17  
**状态：** 待用户确认后实施  
**范围：** 仅非代码文档（README / LICENSE / docs / 社区治理文件）；**不修改任何 `.go` 源码**

## 背景与目标

参照 [bytedance/sonic](https://github.com/bytedance/sonic) 的开源文档体系，为当前几乎无文档的 epack 仓库补齐入口文档、原理/用法/线格式说明，以及贡献与安全相关文件。

epack 是 Go 二进制序列化库，公开 API 主要为：

- `Marshal(obj interface{}) ([]byte, error)`
- `Unmarshal(buffer []byte, obj interface{}) error`
- `LoadTemplate(obj ...interface{}) error`
- 结构体字段通过 `` `epack:"N"` `` 指定序列化顺序（1-based）

## 已确认决策

| 项 | 决策 |
|----|------|
| 文档范围 | 完整对齐 sonic（方案 B / 落地为方案 1） |
| 协议 | **MIT License**（非 Apache-2.0） |
| 语言 | 中英双语（`README.md` + `README_ZH_CN.md`；docs 同理） |
| 版权行 | `Copyright (c) 2026 epack contributors` |
| 代码 | 禁止修改业务代码；示例代码仅出现在 Markdown 中 |

## 将新增的文件清单

```
LICENSE
README.md
README_ZH_CN.md
CONTRIBUTING.md
CODE_OF_CONDUCT.md
SECURITY.md
CHANGELOG.md
CREDITS
docs/INTRODUCTION.md
docs/INTRODUCTION_ZH_CN.md
docs/USAGE.md
docs/USAGE_ZH_CN.md
docs/WIRE_FORMAT.md
docs/WIRE_FORMAT_ZH_CN.md
```

说明：

- 本设计稿位于 `docs/superpowers/specs/`，属于过程文档，实施时保留。
- 不新增 sonic 的 `docs/imgs` 基准图（epack 无现成图资源）；Benchmark 以命令 + 文字结果占位为主。
- 不修改 `examples/` 下已有 Go 示例代码；README/USAGE 中链接到它们。

## README 结构（中英一致）

对齐 sonic 的信息架构，内容改为 epack：

1. 标题 + 一句话定位（高性能 Go 二进制序列化 / 反序列化库）
2. 语言切换：`English | 中文`
3. **Requirement**：Go 版本（与 `go.mod` 一致：`go 1.26.1` 所声明环境）、常见 OS/Arch
4. **Features**：运行时绑定无需 codegen、`epack` tag 控制字段序、模板缓存、缓冲池、小端线格式等
5. **APIs**：列出公开 API，并说明可在包文档 / 源码中查看
6. **Benchmarks**：说明如何运行 `go test -bench=...`（如 `BenchmarkMarshal_Epack` 等）；若本地可跑出结果则摘录，否则给命令与对比说明（vs `encoding/json`）
7. **How it works**：链接 `docs/INTRODUCTION.md`（中文链 `_ZH_CN`）
8. **Usage**（精简）：
   - Marshal / Unmarshal
   - 结构体与 `` `epack:"N"` ``
   - LoadTemplate
   - 更多见 `docs/USAGE.md`
9. **Documentation** 索引：INTRODUCTION / USAGE / WIRE_FORMAT
10. **Contributing** → `CONTRIBUTING.md`
11. **License** → MIT，链到 `LICENSE`

示例代码使用 `github.com/DeffPuzzL/epack` 作为 import path（与 `go.mod` / `examples` 一致）。

## docs 内容大纲

### INTRODUCTION / INTRODUCTION_ZH_CN

- 项目定位与适用场景
- 与 JSON / 其他编解码库的差异（二进制、字段序由 tag 决定、非自描述字段名）
- 编解码主路径：反射绑定 →（可选）`LoadTemplate` 预编译 units → encode/decode
- 类型缓存（`sync.Map`）、缓冲池（`sync.Pool`）
- 线格式固定小端（不依赖本机端序作为线格式）
- 已知约束与建议：
  - 结构体导出字段建议使用连续正整数 `epack` tag
  - Unmarshal 目标必须为非 nil 指针
  - `time.Time` 按 UnixNano 编解码（时区信息可能丢失）
  - 嵌套指针结构体等边界以当前实现为准，文档如实描述、不承诺未实现行为

### USAGE / USAGE_ZH_CN

- 安装：`go get github.com/DeffPuzzL/epack`
- 基本类型 round-trip
- 结构体 + tag、嵌套结构体
- 切片 / 数组、map、指针与 nil、`time.Time`
- `LoadTemplate` 预热与重复调用场景
- 常见错误（nil、非指针 Unmarshal、tag 非法、尾部多余数据等）——描述行为，不改代码
- 链接仓库内 `examples/`、`examples_test.go`

### WIRE_FORMAT / WIRE_FORMAT_ZH_CN

基于当前实现文档化（只描述、不改）：

- **Head**：
  - 短头 2 字节：`size < 0x0400` 时使用；小端；低 5 bit 为 `reflect.Kind`，其余编码 size
  - 长头 8 字节：首字节 bit7=1 标识；小端 uint64；低 5 bit kind，高位 size
- **Number**：类型头 + 小端 payload（含 bool）
- **String**：类型头（size=字节长度）+ 原始字节
- **Slice / Array**：类型头（size=元素个数）；数字切片可有 `0xFF` + elem kind 的紧凑路径
- **Map**：类型头（size=entry 数）+ 交替 key/value
- **Struct**：类型头（size=字段数）+ 按 `epack` tag 序号顺序的字段载荷
- **time.Time**：按 int64 UnixNano 数值路径编码
- **兼容性**：勿随意改动已发布类型的 tag 序号；跨语言实现需遵循本文

## 社区与治理文件

### LICENSE

标准 MIT 全文，版权：

```text
Copyright (c) 2026 epack contributors
```

### CONTRIBUTING.md

对齐 sonic 结构，适配 epack：

- First PR / Issues / 提交 PR 步骤
- 分支前缀建议：`optimize|feature|bugfix|doc|ci|test|refactor`/
- Conventional Commits
- 前置：`gofmt`、相关测试通过
- **明确**：本次及默认贡献可包含文档；安全问题走 SECURITY
- 去掉 ByteDance 专用邮箱与 sonic 特有 develop 分支硬性要求；默认以 `main` 为目标分支（若仓库无 develop）

### CODE_OF_CONDUCT.md

采用 Contributor Covenant 2.x 通用文本（与 sonic 同类），报告渠道指向 SECURITY / 维护者（占位说明可在仓库 Issues 中联系维护者）。

### SECURITY.md

- 请勿在公开 Issue 中披露未修复的安全漏洞
- 通过私密渠道联系维护者（GitHub Security Advisories 优先；若未开启则说明通过私信/邮件联系维护者）
- 不编造不存在的邮箱；若无公开邮箱，写「通过仓库维护者联系」

### CHANGELOG.md

- `## [Unreleased]`
- 初始条目说明文档与开源元数据就绪；不虚构未发生的 API 变更

### CREDITS

- 简短致谢贡献者与参考项目（可注明文档结构参考 sonic，实现与协议独立）

## 实施约束

1. **只写文档文件**；不改 `.go`、`go.mod`（除非用户另行要求）。
2. 文档描述必须与当前代码行为一致；不确定处用「当前实现」措辞，避免过度承诺。
3. 中英文内容语义对齐，专有名词（API 名、tag）保持英文一致。
4. 不复制 sonic 的 Apache-2.0 LICENSE 或 ByteDance 联系方式。
5. Benchmark 章节不得捏造性能数字；无现成数字时只给复现命令。

## 非目标

- 不实现网站 / 不生成 pkg.go.dev 徽章依赖的 CI
- 不添加 `.github` workflows（除非后续单独要求）
- 不重写 examples 代码
- 不引入 `docs/imgs` 基准图

## 验收标准

- [ ] 上表文件均存在且中英 README 可互相跳转
- [ ] LICENSE 为 MIT，版权行为约定文案
- [ ] README 能让新用户 5 分钟内完成 Marshal/Unmarshal 试用
- [ ] WIRE_FORMAT 与 head/number/slice 实现描述一致
- [ ] 无业务代码 diff
- [ ] 未将协议误写为 Apache-2.0 / 未误称与 sonic 协议兼容
```
