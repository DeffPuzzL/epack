# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project aims to follow [Semantic Versioning](https://semver.org/) when tags are published.

## [Unreleased]

## [1.0.1] - 2026-07-18

### Changed

- Unexport internal types and helpers (`ePack`, `config`, `unit`, `intsToBytes`, `bytesToInts`, `simpleNumber`)
- Slim package godoc examples to `ExampleMarshal` and `ExampleLoadTemplate`
- Expand benchmarks to compare epack with protobuf, msgpack, sonic, and encoding/json

### Added

- Detailed benchmark docs: `docs/BENCHMARK.md`, `docs/BENCHMARK_ZH_CN.md`

## [1.0.0] - 2026-07-17

### Added

- Initial public release of epack binary marshal/unmarshal for Go
- Public APIs: `Marshal`, `Unmarshal`, `LoadTemplate`
- Bilingual documentation (README, introduction, usage, wire format)
- Community files: `LICENSE` (MIT), `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, `CREDITS`
- Benchmark comparison assets under `docs/images/`

[1.0.1]: https://github.com/DeffPuzzL/epack/releases/tag/v1.0.1
[1.0.0]: https://github.com/DeffPuzzL/epack/releases/tag/v1.0.0
