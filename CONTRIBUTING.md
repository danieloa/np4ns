# Contributing to np4ns

Thank you for your interest in contributing to np4ns! This document provides guidelines and instructions for contributing.

## Code of Conduct

This project adheres to a Code of Conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates. When creating a bug report, include as many details as possible using the bug report template.

**Good bug reports include:**
- A clear and descriptive title
- Steps to reproduce the issue
- Expected vs actual behavior
- Environment details (Kubernetes version, np4ns version, etc.)
- Relevant logs and configuration

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, use the feature request template and include:

- A clear and descriptive title
- Detailed description of the proposed functionality
- Use cases and examples
- Why this enhancement would be useful

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Make your changes** following the coding standards
3. **Add tests** for your changes
4. **Update documentation** as needed
5. **Ensure tests pass** locally
6. **Submit a pull request** using the PR template

## Development Setup

### Prerequisites

- Go 1.24.0+
- Docker
- kubectl
- Kind or Minikube (for local testing)
- Make

### Setting Up Your Development Environment

```bash
# Clone your fork
git clone https://github.com/YOUR-USERNAME/np4ns.git
cd np4ns

# Add upstream remote
git remote add upstream https://github.com/danieloa/np4ns.git

# Install dependencies
go mod download

# Set up a local Kubernetes cluster (using Kind)
kind create cluster --name np4ns-dev
```

### Running Locally

```bash
# Run the operator locally (connects to your kubeconfig cluster)
go run cmd/main.go

# Or use make
make run
```

### Running Tests

```bash
# Run unit tests
make test

# Run tests with coverage
make test-coverage

# View coverage report
go tool cover -html=cover.out
```

### Building

```bash
# Build the binary
make build

# Build the Docker image
make docker-build IMG=np4ns:dev

# Load into Kind cluster
kind load docker-image np4ns:dev --name np4ns-dev

# Deploy to cluster
make deploy IMG=np4ns:dev
```

## Coding Standards

### Go Code Style

- Follow standard Go conventions and idioms
- Use `gofmt` to format your code
- Run `golangci-lint` before submitting PRs
- Write clear, descriptive variable and function names
- Add comments for exported functions and complex logic

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run
```

### Commit Messages

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, missing semicolons, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**
```
feat(controller): add support for custom network policy templates

fix(rbac): correct permissions for namespace updates

docs(readme): update installation instructions

test(controller): add tests for namespace exception list
```

### Branch Naming

Use descriptive branch names with the following prefixes:
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation updates
- `refactor/` - Code refactoring
- `test/` - Test updates

**Examples:**
- `feature/configurable-network-policy`
- `fix/namespace-annotation-race-condition`
- `docs/add-helm-chart-guide`

## Testing Guidelines

### Unit Tests

- Write tests for all new functionality
- Maintain or improve code coverage
- Use table-driven tests where appropriate
- Mock external dependencies

```go
func TestShouldEnforceNetworkPolicy(t *testing.T) {
    tests := []struct {
        name      string
        namespace string
        want      bool
    }{
        {"system namespace", "kube-system", false},
        {"regular namespace", "myapp", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := shouldEnforceNetworkPolicy(tt.namespace)
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Integration Tests

Test the operator against a real Kubernetes cluster when possible:
- Create test namespaces
- Verify network policy creation
- Test policy recreation after deletion
- Test policy compliance enforcement

## Documentation

### Code Documentation

- Add godoc comments for all exported functions and types
- Include examples where helpful
- Document edge cases and assumptions

### User Documentation

When adding new features, update:
- `README.md` - For user-facing changes
- `DEPLOYMENT.md` - For deployment-related changes
- `CHANGELOG.md` - Add entry under "Unreleased"
- Helm chart documentation if applicable

## Pull Request Process

1. **Update CHANGELOG.md** with your changes under the "Unreleased" section
2. **Update documentation** to reflect any changes
3. **Ensure all tests pass** and code is properly formatted
4. **Fill out the PR template** completely
5. **Link related issues** using keywords (Fixes #123, Relates to #456)
6. **Wait for review** - maintainers will review your PR and may request changes
7. **Address feedback** - make requested changes and push updates
8. **Merge** - once approved, a maintainer will merge your PR

### PR Review Criteria

Reviewers will check:
- Code quality and style
- Test coverage
- Documentation completeness
- Backward compatibility
- Performance implications
- Security considerations

## Release Process

Releases are automated via GitHub Actions:

1. Update `CHANGELOG.md` and move "Unreleased" items to a new version section
2. Create and push a version tag: `git tag v0.1.0 && git push origin v0.1.0`
3. GitHub Actions will:
   - Build and publish container images
   - Create a GitHub release with notes
   - Package and publish Helm chart

## Getting Help

- **Questions**: Open a discussion in GitHub Discussions
- **Bugs**: File an issue using the bug report template
- **Features**: File an issue using the feature request template
- **Security**: See [SECURITY.md](SECURITY.md) for security-related concerns

## Recognition

Contributors will be recognized in:
- Release notes for their contributions
- The project's contributor list

Thank you for contributing to np4ns!
