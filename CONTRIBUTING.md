# How to Contribute

## Your First Pull Request

We use GitHub for our codebase. Start with [About pull requests](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/about-pull-requests).

## Branch Organization

Development targets the default branch (`main`). Prefer short-lived topic branches.

Suggested branch name prefixes:

- `feature/`
- `bugfix/`
- `doc/`
- `test/`
- `refactor/`
- `optimize/`
- `ci/`

## Bugs

### Finding known issues

Use this repository's GitHub Issues. Search before opening a duplicate.

### Reporting new issues

Please include:

- Go version and OS/Arch
- Minimal reproducible code (prefer a small test or [Go Playground](https://go.dev/play/) link)
- Expected vs actual behavior

### Security bugs

Do **not** file public issues for security vulnerabilities. See [SECURITY.md](SECURITY.md).

## Submit a Pull Request

1. Search existing PRs/issues to avoid duplicates.
2. Open or reference an issue that describes the problem or design.
3. Fork the repository and create a topic branch.
4. Make your change with appropriate tests (for code) or accurate wording (for docs).
5. Format with `gofmt` when touching Go files.
6. Use [Conventional Commits](https://www.conventionalcommits.org/) for commit messages and PR titles, e.g. `doc: add wire format guide`.
7. Push and open a PR against `main`.

## Contribution Prerequisites

- Familiarity with Go and GitHub
- Run relevant tests before submitting code changes:

```bash
go test ./...
```

- Documentation-only PRs are welcome and encouraged.

## Code Style

Follow [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) and [Effective Go](https://go.dev/doc/effective_go).

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
