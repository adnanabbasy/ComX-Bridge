# Contributing to ComX-Bridge

First off, thanks for taking the time to contribute to ComX-Bridge! 🎉

This document guides you through the contribution process.

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Pull Request Guide](#pull-request-guide)
- [Code Style](#code-style)
- [Commit Message Guidelines](#commit-message-guidelines)

## Code of Conduct

This project is committed to providing a welcoming environment for all contributors.
Please participate in a spirit of respect and cooperation.

### Respect for AI Usage
We embrace the use of AI tools (LLMs, Copilots, etc.) in development. **Discrimination, disparagement, or harassment based on the use of AI assistance is strictly prohibited.** Evaluate code based on its quality, correctness, and maintainability, not whether it was written by a human or an AI.

## Getting Started

### Submitting Issues

- **Bug Reports**: Found a bug? Please create an [Issue](https://github.com/commatea/ComX-Bridge/issues).
- **Feature Requests**: Have an idea for a new feature? Propose it via an Issue.
- **Questions**: Ask questions via Discussions or Issues.

### Writing Good Issues

```markdown
## Description
Clear description of the problem or feature

## Reproduction Steps (for bugs)
1. With this config...
2. Run this command...
3. This error occurs...

## Expected Behavior
What you expected to happen

## Environment
- OS: Windows 11 / Ubuntu 22.04 / macOS 14
- Go Version: 1.24
- ComX-Bridge Version: v1.0.0
```

## Development Setup

### Prerequisites

- Go 1.24+
- Git
- Make (Optional, direct `go build` works on Windows)

### Setup Steps

```bash
# 1. Fork and Clone
git clone https://github.com/commatea/ComX-Bridge.git
cd ComX-Bridge

# 2. Install Dependencies
go mod download

# 3. Verify Build
go build ./...

# 4. Run Tests
go test -v ./...

# 5. Install Dev Tools (Optional)
make tools
```

## How to Contribute

### 1. Fork & Clone

```bash
# After fork
git clone https://github.com/commatea/ComX-Bridge.git
cd ComX-Bridge
git remote add upstream https://github.com/commatea/ComX-Bridge.git
```

### 2. Create Branch

```bash
# Sync with latest code
git fetch upstream
git checkout main
git merge upstream/main

# Create new branch
git checkout -b feature/your-feature-name
# or
git checkout -b fix/bug-description
```

### 3. Make Changes

- Write code
- Add tests
- Update documentation (if needed)

### 4. 🤖 AI-Assisted Development (Recommended)

**We strongly encourage the use of AI tools** to improve productivity and code quality. 

Feel free to use AI for:
- Generating boilerplate code
- Writing unit tests
- Refactoring and optimizing
- Improving documentation and commit messages

> **Note**: While AI is a powerful tool, you are responsible for the quality of your contribution. Please review and understand the code you submit.

### 5. Run Tests

```bash
# Run all tests
go test -v -race ./...

# Run specific package tests
go test -v ./pkg/transport/...

# Check coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 6. Commit & Push

```bash
git add .
git commit -m "feat: add new transport driver for XYZ"
git push origin feature/your-feature-name
```

### 7. Create Pull Request

Create a Pull Request on GitHub.

## Pull Request Guide

### PR Checklist

- [ ] Code builds (`go build ./...`)
- [ ] All tests pass (`go test ./...`)
- [ ] No lint errors (`golangci-lint run`)
- [ ] Tests added for new features
- [ ] Documentation updated (if needed)
- [ ] Commit messages follow the guidelines

### PR Template

```markdown
## Changes
Describe the changes made in this PR.

## Related Issues
Fixes #123

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Documentation update
- [ ] Refactoring
- [ ] Test addition

## Testing
Describe how to test the changes.
```

## Code Style

### Go Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Format with `gofmt` or `goimports`
- Lint with `golangci-lint`

```bash
# Format
go fmt ./...
goimports -w .

# Lint
golangci-lint run
```

### Naming Conventions

```go
// Good
type TransportConfig struct {}
func NewEngine(config *Config) (*Engine, error) {}
var ErrNotFound = errors.New("not found")

// Bad
type transport_config struct {}
func new_engine(c *Conf) *Eng {}
var NotFoundErr = errors.New("not found")
```

### Commenting

```go
// Package transport provides communication channel abstractions.
package transport

// Transport is the core interface for all communication channels.
// Implementations must be safe for concurrent use.
type Transport interface {
    // Connect establishes a connection to the remote endpoint.
    Connect(ctx context.Context) error
}
```

## Commit Message Guidelines

Follow [Conventional Commits](https://www.conventionalcommits.org/).

### Format

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Types

| Type | Description |
|------|-------------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation changes |
| `style` | Formatting (no code change) |
| `refactor` | Refactoring |
| `test` | Adding/modifying tests |
| `chore` | Build/config updates |

### Examples

```bash
feat(transport): add BLE transport driver
fix(modbus): handle CRC calculation error
docs(readme): update installation instructions
test(ai): add unit tests for anomaly detector
chore(ci): add golangci-lint to workflow
```

## 🙏 Thank You!

Thank you for contributing to make ComX-Bridge better!

If you have any questions, feel free to ask via Issues.
