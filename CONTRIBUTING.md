# Contributing to SharkAuth

First off, thank you for considering contributing to SharkAuth. It's people like you that make SharkAuth the open-source identity platform for the agentic era.

We welcome contributions of all kinds — code, documentation, bug reports, feature suggestions, design feedback, and community support. No contribution is too small.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [How Can I Contribute?](#how-can-i-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Enhancements](#suggesting-enhancements)
  - [Pull Requests](#pull-requests)
- [Development Setup](#development-setup)
- [Style Guides](#style-guides)
  - [Git Commit Messages](#git-commit-messages)
  - [Go Style Guide](#go-style-guide)
- [Community](#community)

## Code of Conduct

This project and everyone participating in it is governed by our [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to [raul@sharkauth.com](mailto:raul@sharkauth.com).

## Getting Started

- Make sure you have a [GitHub account](https://github.com/signup/free).
- Search existing issues and PRs before creating new ones.
- If you're new to open source, check out [First Timers Only](https://www.firsttimersonly.com/) or look for issues labeled `good first issue`.

## How Can I Contribute?

### Reporting Bugs

Before creating a bug report, please check the [existing issues](https://github.com/shark-auth/shark/issues) to see if the problem has already been reported. If it has and the issue is still open, add a comment to the existing issue instead of opening a new one.

When you are creating a bug report, please include as many details as possible:

- **Use a clear and descriptive title** for the issue.
- **Describe the exact steps to reproduce the problem** in as much detail as possible.
- **Provide specific examples** to demonstrate the steps (e.g., curl commands, config snippets).
- **Describe the behavior you observed** and why it is a problem.
- **Explain which behavior you expected** instead.
- **Include screenshots or logs** if applicable.
- **Specify your environment:** OS, Go version, and SharkAuth version.

### Suggesting Enhancements

Enhancement suggestions are tracked as [GitHub issues](https://github.com/shark-auth/shark/issues). When creating an enhancement suggestion, please include:

- **Use a clear and descriptive title**.
- **Provide a step-by-step description** of the suggested enhancement.
- **Provide specific examples** to demonstrate the enhancement.
- **Explain why this enhancement would be useful** to SharkAuth users.
- **List some other auth systems or tools** where this enhancement exists, if applicable.

### Pull Requests

1. Fork the repository and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the relevant documentation.
4. Ensure the test suite passes (`go test ./...`).
5. Make sure your code follows the existing style (see [Style Guides](#style-guides)).
6. Issue that pull request!

**Pull Request Process:**

- Update the README.md or documentation with details of changes if applicable.
- Increase version numbers in any examples files and the README.md to the new version that this Pull Request would represent if applicable.
- Your PR will be reviewed by a maintainer. We aim to respond within 48 hours.

## Development Setup

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/) (for admin UI changes)
- Make (optional, for convenience commands)

### Running SharkAuth locally

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/shark.git
cd shark

# Run in dev mode (no config needed, in-memory SQLite)
go run ./cmd/shark serve --dev

# Admin UI will be available at http://localhost:8080/admin
# Issuer at http://localhost:8080
```

### Running tests

```bash
# Run all Go tests
go test ./...

# Run with race detector
go test -race ./...

# Run smoke tests (requires built binary)
./scripts/smoke.sh
```

### Building the admin UI

```bash
cd admin
npm install
npm run build
```

The built assets are embedded into the Go binary via `go:embed`.

## Style Guides

### Git Commit Messages

- Use the present tense ("Add feature" not "Added feature").
- Use the imperative mood ("Move cursor to..." not "Moves cursor to...").
- Limit the first line to 72 characters or less.
- Reference issues and pull requests liberally after the first line.

Example:
```
Add DPoP key rotation endpoint

- Implements RFC 9449 key rotation
- Adds `rotate_dpop_key` to the session API
- Closes #123
```

### Go Style Guide

- Follow standard [Go formatting](https://go.dev/doc/effective_go.html).
- Run `gofmt` before committing.
- Keep functions focused and testable.
- Prefer explicit error handling over panics.
- Document all exported types and functions with Go doc comments.

## Community

- **Discord**: [discord.gg/zq9t6VSt5r](https://discord.com/invite/zq9t6VSt5r) — ask questions, share deployments, meet other contributors
- **Twitter**: [@raulgooo](https://twitter.com/raulgooo) — updates and agent identity threads
- **Docs**: [sharkauth.com/docs](https://sharkauth.com/docs)

If you have any questions that aren't covered here, feel free to open a [Discussion](https://github.com/shark-auth/shark/discussions) or ask on Discord.

Thank you for building the future of agent identity with us! 🦈
