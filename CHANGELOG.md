# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- GitHub Actions workflow for building and publishing container images to ghcr.io
- Multi-architecture support (amd64, arm64) for container images
- Comprehensive documentation (README.md, DEPLOYMENT.md)
- Build status badge in README

### Changed
- Improved README with better structure and installation options
- Simplified container image tagging to version+SHA format (e.g., v0.0.5-a1b2c3d)

## [0.0.4] - 2025-01-21

### Added
- Network policy enforcement for Kubernetes namespaces
- Automatic recreation of deleted network policies
- Configuration via environment variables (NS_EXCEPTION_LIST, NS_TARGET_FOR_NP)
- Namespace annotations to track enforcement status
- Owner references for automatic cleanup
- Comprehensive deployment examples and scenarios
- Verification scripts for testing

### Changed
- Initial operator implementation with namespace controller

## [0.0.1] - 2025-01-21

### Added
- Initial project setup using Operator SDK
- Basic namespace controller structure
- RBAC configuration for namespace and network policy management
- Makefile with build, test, and deployment targets
- Docker support with multi-stage builds
- E2E test framework

[Unreleased]: https://github.com/danieloa/np4ns/compare/v0.0.4...HEAD
[0.0.4]: https://github.com/danieloa/np4ns/compare/v0.0.1...v0.0.4
[0.0.1]: https://github.com/danieloa/np4ns/releases/tag/v0.0.1
