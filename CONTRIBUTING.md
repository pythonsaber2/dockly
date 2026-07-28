# Contributing to Dockly

Thanks for helping make self-hosted deployment simpler. Dockly favors small, understandable changes that preserve its low operational footprint.

## Before you start

- Search existing issues and discussions.
- Open an issue for substantial features or architecture changes before implementation.
- Keep the MVP boundaries in mind: one host, Docker, Git, and a single Dockly binary.
- Never include credentials, environment values, private repository URLs, or production logs in an issue or test fixture.

## Local development

Requirements: Go 1.22+, Git, and Docker for manual deployment tests.

```bash
git clone https://github.com/pythonsaber2/dockly.git
cd dockly
make check
DOCKLY_DATA_DIR="$PWD/.dockly" go run ./cmd/dockly
```

The unit tests use fake command runners and do not require Docker.

## Making a change

1. Create a focused branch from `main`.
2. Add a failing test that describes the behavior.
3. Implement the smallest complete change.
4. Run `make check`.
5. Update user-facing documentation when behavior, configuration, or API responses change.
6. Open a pull request using the repository template.

Use `gofmt`; prefer the standard library; keep interfaces at integration boundaries; and return actionable errors without exposing secrets. New dependencies need a clear size, security, and maintenance justification.

## Commits and pull requests

Write imperative commit subjects (for example, `Add deployment cancellation`). A pull request should solve one problem, explain the user impact, include verification steps, and call out security or migration implications.

By participating, you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md). Contributions are licensed under the MIT License.
