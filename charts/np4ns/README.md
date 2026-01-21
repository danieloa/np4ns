# np4ns Helm Chart

Network Policies For Namespaces - Kubernetes operator that automatically enforces network policies across namespaces.

## Installation

### Add Helm Repository (if published)

```bash
helm repo add np4ns https://danieloa.github.io/np4ns
helm repo update
```

### Install from Local Chart

```bash
# From the repository root
helm install np4ns charts/np4ns

# Or with custom values
helm install np4ns charts/np4ns -f my-values.yaml
```

### Install from Specific Namespace

```bash
helm install np4ns charts/np4ns --namespace np4ns-system --create-namespace
```

## Configuration

The following table lists the configurable parameters and their default values.

### Image Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Image repository | `ghcr.io/danieloa/np4ns` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Image tag (defaults to chart appVersion) | `""` |
| `imagePullSecrets` | Image pull secrets | `[]` |

### Namespace Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `namespace.create` | Create the namespace | `true` |
| `namespace.name` | Namespace name | `np4ns-system` |

### Network Policy Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.nsExceptionList` | Comma-separated list of namespaces to exclude | `"kube-system,kube-public,kube-node-lease,local-path-storage"` |
| `config.nsTargetForNp` | Comma-separated list of namespaces to enforce (empty = all except exceptions) | `""` |

### Deployment Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `128Mi` |
| `resources.requests.cpu` | CPU request | `10m` |
| `resources.requests.memory` | Memory request | `64Mi` |

### RBAC Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `rbac.create` | Create RBAC resources | `true` |
| `serviceAccount.create` | Create service account | `true` |
| `serviceAccount.name` | Service account name | `""` (auto-generated) |

### Other Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `leaderElection.enabled` | Enable leader election | `true` |
| `nodeSelector` | Node selector | `{}` |
| `tolerations` | Tolerations | `[]` |
| `affinity` | Affinity rules | `{}` |

## Examples

### Default Installation

Enforces network policies on all namespaces except system namespaces:

```bash
helm install np4ns charts/np4ns
```

### Selective Enforcement

Only enforce on specific namespaces:

```bash
helm install np4ns charts/np4ns \
  --set config.nsTargetForNp="production,staging,team-a"
```

### Custom Resource Limits

```bash
helm install np4ns charts/np4ns \
  --set resources.limits.cpu=1000m \
  --set resources.limits.memory=256Mi
```

### Using Custom Values File

Create a `my-values.yaml`:

```yaml
config:
  nsExceptionList: "kube-system,kube-public,monitoring"
  nsTargetForNp: "prod-*"

resources:
  limits:
    cpu: 1000m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 128Mi

nodeSelector:
  kubernetes.io/os: linux
```

Install with:

```bash
helm install np4ns charts/np4ns -f my-values.yaml
```

## Upgrading

```bash
# Upgrade to new version
helm upgrade np4ns charts/np4ns

# Upgrade with new values
helm upgrade np4ns charts/np4ns -f my-values.yaml
```

## Uninstalling

```bash
helm uninstall np4ns
```

To also delete the namespace:

```bash
kubectl delete namespace np4ns-system
```

## Testing the Chart

```bash
# Lint the chart
helm lint charts/np4ns

# Template the chart to see the rendered output
helm template np4ns charts/np4ns

# Dry-run installation
helm install np4ns charts/np4ns --dry-run --debug
```

## More Information

For more details about the operator, see the [main README](../../README.md) and [DEPLOYMENT guide](../../DEPLOYMENT.md).
