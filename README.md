# np4ns

[![Build and Publish Container](https://github.com/danieloa/np4ns/actions/workflows/build-and-publish.yaml/badge.svg)](https://github.com/danieloa/np4ns/actions/workflows/build-and-publish.yaml)

**Network Policies For Namespaces** - A Kubernetes operator that automatically enforces network policies across namespaces.

## Overview

`np4ns` is a Kubernetes operator written in Go that ensures every namespace in your cluster has a compliant network policy enforced. The operator continuously monitors namespaces and network policies, automatically:

- Creating network policies for new namespaces
- Recreating network policies if they are deleted
- Updating network policies that don't meet compliance requirements
- Tracking enforcement with namespace annotations

## Features

- **Automatic Enforcement**: Network policies are automatically created for all target namespaces
- **Self-Healing**: Deleted or modified policies are automatically restored to compliance
- **Flexible Configuration**: Control which namespaces are included or excluded via environment variables
- **Zero CRDs**: Works with native Kubernetes resources only (no custom resources required)
- **Namespace Annotations**: Track enforcement status with automatic timestamping
- **Owner References**: Policies are cleaned up automatically when namespaces are deleted

## Quick Start

### Prerequisites

- Kubernetes cluster (v1.11.3+)
- kubectl configured with cluster access
- Docker (for building images)
- Go 1.24.0+ (for local development)

### Installation

#### Option 1: Using Helm (Recommended)

The easiest way to install np4ns:

```bash
# Install using Helm
helm install np4ns charts/np4ns --namespace np4ns-system --create-namespace

# Or with custom configuration
helm install np4ns charts/np4ns \
  --namespace np4ns-system \
  --create-namespace \
  --set config.nsExceptionList="kube-system,kube-public,monitoring" \
  --set config.nsTargetForNp=""

# Verify deployment
kubectl get deployment -n np4ns-system
kubectl logs -n np4ns-system deployment/np4ns-controller-manager -f
```

See the [Helm Chart README](charts/np4ns/README.md) for all configuration options.

#### Option 2: Using Pre-built Images

Deploy using published images from GitHub Container Registry:

```bash
# Deploy a specific version (recommended for production)
make deploy IMG=ghcr.io/danieloa/np4ns:v0.0.5-a1b2c3d

# Or deploy the latest build from main
make deploy IMG=ghcr.io/danieloa/np4ns:latest-a1b2c3d

# Verify deployment
kubectl get deployment -n np4ns-system
kubectl logs -n np4ns-system deployment/np4ns-controller-manager -f
```

Image tag format: `v0.0.5-a1b2c3d` (version + 7-char commit SHA)
- Multi-architecture support (amd64, arm64)
- Browse available tags at: https://github.com/danieloa/np4ns/pkgs/container/np4ns

#### Option 3: Building from Source

Build and deploy locally (useful for development):

```bash
# Build and load image (for Kind clusters)
make docker-build IMG=np4ns:latest
kind load docker-image np4ns:latest

# Deploy to cluster
make deploy IMG=np4ns:latest

# Verify deployment
kubectl get deployment -n np4ns-system
kubectl logs -n np4ns-system deployment/np4ns-controller-manager -f
```

### Test It Out

Create a test namespace and watch the operator enforce a network policy:

```bash
# Create a test namespace
kubectl create namespace test-app

# Verify network policy was created
kubectl get networkpolicies -n test-app

# View the enforced policy
kubectl describe networkpolicy enforced-network-policy -n test-app

# Check namespace annotation
kubectl get namespace test-app -o jsonpath='{.metadata.annotations}'
```

## Configuration

The operator is configured via environment variables set in a ConfigMap:

| Variable | Description | Default |
|----------|-------------|---------|
| `NS_EXCEPTION_LIST` | Comma-separated namespaces to exclude from enforcement | `kube-system,kube-public,kube-node-lease,local-path-storage` |
| `NS_TARGET_FOR_NP` | Comma-separated namespaces to enforce (if empty, enforces on all except exceptions) | `""` (empty = all namespaces) |

### Configuration Examples

**Enforce on all namespaces except system namespaces:**
```yaml
NS_EXCEPTION_LIST: "kube-system,kube-public,kube-node-lease,local-path-storage"
NS_TARGET_FOR_NP: ""
```

**Enforce only on specific namespaces:**
```yaml
NS_EXCEPTION_LIST: "kube-system,kube-public"
NS_TARGET_FOR_NP: "production,staging,team-a,team-b"
```

To update configuration:
```bash
kubectl edit configmap np4ns-config -n np4ns-system
kubectl rollout restart deployment np4ns-controller-manager -n np4ns-system
```

## How It Works

1. **Namespace Watch**: The operator watches all namespace events (create, update, delete)
2. **Policy Check**: For each target namespace, it checks if a compliant network policy exists
3. **Enforcement**: If missing or non-compliant, the policy is created or updated
4. **Continuous Reconciliation**: Changes to policies trigger automatic reconciliation

### Enforced Network Policy

The operator creates a NetworkPolicy named `enforced-network-policy` with the following spec:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: enforced-network-policy
  namespace: <target-namespace>
spec:
  podSelector: {}  # Applies to all pods
  policyTypes:
  - Egress
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: target-namespace
      podSelector:
        matchLabels:
          app: myapp
    ports:
    - protocol: TCP
      port: 80
```

This policy allows egress traffic to port 80 on pods labeled `app: myapp` in namespaces labeled `name: target-namespace`, and blocks all other egress traffic.

## Documentation

### Getting Started

- **[README](README.md)**: This file - quick start and overview
- **[Deployment Guide](DEPLOYMENT.md)**: Comprehensive deployment scenarios and troubleshooting
- **[Helm Chart README](charts/np4ns/README.md)**: Helm installation and configuration

### Advanced Topics

- **[Customization Guide](docs/CUSTOMIZATION.md)**: How to customize network policies
- **[Real-World Examples](docs/EXAMPLES.md)**: Production use cases and patterns
- **[Observability Guide](docs/OBSERVABILITY.md)**: Monitoring, metrics, and alerting
- **[Performance Guide](docs/PERFORMANCE.md)**: Scaling and optimization
- **[Architecture](docs/ARCHITECTURE.md)**: Technical architecture and design decisions
- **[Upgrade Guide](docs/UPGRADE.md)**: How to upgrade between versions

### Community

- **[Contributing](CONTRIBUTING.md)**: How to contribute to the project
- **[Code of Conduct](CODE_OF_CONDUCT.md)**: Community guidelines
- **[Security Policy](SECURITY.md)**: Security and vulnerability reporting
- **[Changelog](CHANGELOG.md)**: Version history and changes
- **[License](LICENSE)**: Apache 2.0

## Development

### Run Locally

Run the operator locally against your cluster:

```bash
# Run the controller locally (connects via kubeconfig)
go run cmd/main.go
```

### Run Tests

```bash
# Run unit tests
make test

# Run with coverage
make test-coverage
```

### Project Structure

```
.
├── cmd/
│   └── main.go                    # Operator entry point
├── internal/controller/
│   ├── namespace_controller.go    # Main reconciliation logic
│   └── namespace_controller_test.go
├── config/                        # Kubernetes manifests
│   ├── manager/                   # Deployment manifests
│   ├── rbac/                      # RBAC configuration
│   └── samples/                   # Example configurations
└── test/                          # E2E tests
```

### Building

This operator was built using the Operator SDK. Useful resources:

- [Build Kubernetes Operator with Kubebuilder](https://www.codereliant.io/build-kubernetes-operator-kubebuilder/)
- [Hands-on Kubernetes Operator Development](https://www.codereliant.io/hands-on-kubernetes-operator-development-part-2/)
- [Writing a Controller for Pod Labels](https://kubernetes.io/blog/2021/06/21/writing-a-controller-for-pod-labels/)

## Community and Support

### Getting Help

- **Questions**: Open a [Discussion](https://github.com/danieloa/np4ns/discussions)
- **Bug Reports**: File an [Issue](https://github.com/danieloa/np4ns/issues/new?template=bug_report.yml)
- **Feature Requests**: Submit an [Issue](https://github.com/danieloa/np4ns/issues/new?template=feature_request.yml)
- **Security Issues**: See [SECURITY.md](SECURITY.md)

### Contributing

We welcome contributions! Here's how to get started:

1. Read the [Contributing Guide](CONTRIBUTING.md)
2. Check out [open issues](https://github.com/danieloa/np4ns/issues)
3. Submit a pull request

### Community

- Star the repository if you find it useful
- Watch for updates and new releases
- Share your use cases and feedback

## Roadmap

See our [GitHub Issues](https://github.com/danieloa/np4ns/issues) for planned features and enhancements. Key areas of focus:

- ConfigMap-based policy templates
- Enhanced observability and metrics
- Policy validation and testing tools
- Multi-policy support per namespace

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.

## Acknowledgments

Built with:
- [Operator SDK](https://sdk.operatorframework.io/)
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Kubebuilder](https://book.kubebuilder.io/)

Inspired by the Kubernetes community's commitment to security and best practices.
