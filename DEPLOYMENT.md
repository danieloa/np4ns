# np4ns Deployment Guide

Complete guide for deploying and configuring the np4ns (Network Policies For Namespaces) Kubernetes operator.

## Table of Contents

- [Quick Start](#quick-start)
- [Configuration Options](#configuration-options)
- [Deployment Scenarios](#deployment-scenarios)
- [Testing & Verification](#testing--verification)
- [Troubleshooting](#troubleshooting)

---

## Quick Start

### Prerequisites

- Kubernetes cluster (v1.11.3+)
- kubectl configured to access your cluster
- Cluster admin permissions

### Deploy with Default Configuration

```bash
# Build and load image into Kind (if using Kind)
make docker-build IMG=np4ns:latest
kind load docker-image np4ns:latest

# Deploy to cluster
make deploy IMG=np4ns:latest

# Verify deployment
kubectl get deployment -n np4ns-system
kubectl logs -n np4ns-system deployment/np4ns-controller-manager -f
```

### Create Test Namespaces

```bash
# Create sample test namespaces
kubectl apply -f config/samples/test-namespaces.yaml

# Create target namespace (for network policy egress rules)
kubectl apply -f config/samples/target-namespace-setup.yaml

# Verify network policies were created
kubectl get networkpolicies --all-namespaces | grep enforced
```

---

## Configuration Options

The operator is configured via environment variables defined in a ConfigMap.

### Environment Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `NS_EXCEPTION_LIST` | Comma-separated list of namespaces to **exclude** from enforcement | `kube-system,kube-public,kube-node-lease,local-path-storage` | `kube-system,monitoring,istio-system` |
| `NS_TARGET_FOR_NP` | Comma-separated list of namespaces to **include** for enforcement (if set, ONLY these namespaces will be enforced) | `""` (empty = all namespaces except exceptions) | `production,staging,team-a` |

### Configuration Logic

The operator uses the following logic to determine if a namespace should have network policies enforced:

1. **Check Exception List First**: If namespace is in `NS_EXCEPTION_LIST`, skip enforcement
2. **Check Target List**:
   - If `NS_TARGET_FOR_NP` is **empty** → Enforce on all namespaces (except exceptions)
   - If `NS_TARGET_FOR_NP` is **set** → Only enforce on namespaces in this list

### Updating Configuration

To change the configuration:

```bash
# Edit the ConfigMap
kubectl edit configmap np4ns-config -n np4ns-system

# Restart the operator to pick up changes
kubectl rollout restart deployment np4ns-controller-manager -n np4ns-system
```

---

## Deployment Scenarios

### Scenario 1: Default - Enforce on All Namespaces

**Use Case**: Enforce network policies on all user namespaces, exclude system namespaces.

```bash
kubectl apply -f config/samples/deployment-examples/01-default-config.yaml
```

**ConfigMap**:
```yaml
NS_EXCEPTION_LIST: "kube-system,kube-public,kube-node-lease,local-path-storage"
NS_TARGET_FOR_NP: ""
```

**Result**: All namespaces get enforced network policies except kube-system, kube-public, kube-node-lease, local-path-storage.

---

### Scenario 2: Selective Enforcement - Production Only

**Use Case**: Only enforce on specific production namespaces.

```bash
kubectl apply -f config/samples/deployment-examples/02-selective-enforcement.yaml
```

**ConfigMap**:
```yaml
NS_EXCEPTION_LIST: "kube-system,kube-public,kube-node-lease,local-path-storage"
NS_TARGET_FOR_NP: "production,prod-api,prod-database,prod-frontend"
```

**Result**: Only the 4 specified production namespaces get enforced policies. All other namespaces (including dev, staging) are ignored.

---

### Scenario 3: Team-Based - Exclude Monitoring/Logging

**Use Case**: Enforce on all team namespaces, exclude monitoring infrastructure.

```bash
kubectl apply -f config/samples/deployment-examples/03-team-based-enforcement.yaml
```

**ConfigMap**:
```yaml
NS_EXCEPTION_LIST: "kube-system,kube-public,kube-node-lease,local-path-storage,monitoring,logging,istio-system"
NS_TARGET_FOR_NP: ""
```

**Result**: All namespaces except system, monitoring, logging, and istio-system get enforced policies.

---

### Scenario 4: Development/Staging Only

**Use Case**: Test network policies in dev/staging before production rollout.

```bash
kubectl apply -f config/samples/deployment-examples/04-development-only.yaml
```

**ConfigMap**:
```yaml
NS_EXCEPTION_LIST: "kube-system,kube-public,kube-node-lease"
NS_TARGET_FOR_NP: "dev-team-a,dev-team-b,staging-api,staging-frontend"
```

**Result**: Only the 4 specified dev/staging namespaces get enforced policies. Production namespaces are left alone.

---

## Enforced Network Policy Specification

The operator creates a NetworkPolicy named `enforced-network-policy` with the following spec:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: enforced-network-policy
  namespace: <target-namespace>
spec:
  podSelector: {}  # Applies to all pods in the namespace
  policyTypes:
  - Egress
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: target-namespace  # Target namespace must have this label
      podSelector:
        matchLabels:
          app: myapp  # Target pods must have this label
    ports:
    - protocol: TCP
      port: 80
```

### What This Policy Does

- **Applies to**: All pods in the namespace
- **Allows**: Egress traffic to port 80 (TCP) on pods labeled `app: myapp` in namespaces labeled `name: target-namespace`
- **Blocks**: All other egress traffic (implicit deny)

### Customizing the Policy

To customize the network policy spec, modify the `buildCompliantNetworkPolicySpec()` function in `internal/controller/namespace_controller.go`.

---

## Testing & Verification

### Automatic Verification Script

Use the provided verification script to test the operator:

```bash
# Test default namespace
./config/samples/verification-script.sh test-app-1

# Test another namespace
./config/samples/verification-script.sh production-api
```

The script will:
1. Check if namespace exists (create if not)
2. Verify NetworkPolicy was created
3. Show the NetworkPolicy spec
4. Check namespace annotation
5. Test deletion and recreation

### Manual Verification

#### 1. Check Operator is Running

```bash
kubectl get pods -n np4ns-system
kubectl logs -n np4ns-system deployment/np4ns-controller-manager
```

#### 2. Create a Test Namespace

```bash
kubectl create namespace test-enforcement
```

#### 3. Verify NetworkPolicy Created

```bash
# Should show enforced-network-policy
kubectl get networkpolicies -n test-enforcement

# View the policy details
kubectl describe networkpolicy enforced-network-policy -n test-enforcement
```

#### 4. Test Policy Enforcement

```bash
# Delete the network policy
kubectl delete networkpolicy enforced-network-policy -n test-enforcement

# Wait 10 seconds
sleep 10

# Should be recreated automatically
kubectl get networkpolicy enforced-network-policy -n test-enforcement
```

#### 5. Test Non-Compliant Policy Update

```bash
# Create a non-compliant policy
cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: enforced-network-policy
  namespace: test-enforcement
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  ingress: []
EOF

# Wait 10 seconds
sleep 10

# Should be updated to compliant spec (Egress policy)
kubectl get networkpolicy enforced-network-policy -n test-enforcement -o yaml | grep -A 5 "policyTypes:"
```

#### 6. Check Namespace Annotations

```bash
# Should show network-policy/enforced annotation with timestamp
kubectl get namespace test-enforcement -o jsonpath='{.metadata.annotations}'
```

---

## Troubleshooting

### NetworkPolicy Not Created

**Symptoms**: NetworkPolicy doesn't appear in target namespace

**Possible Causes**:

1. **Namespace in exception list**
   ```bash
   # Check operator logs
   kubectl logs -n np4ns-system deployment/np4ns-controller-manager | grep "Skipping namespace"
   ```

2. **Target list configured but namespace not included**
   ```bash
   # Check current configuration
   kubectl get configmap np4ns-config -n np4ns-system -o yaml
   ```

3. **Operator not running**
   ```bash
   # Check operator status
   kubectl get deployment -n np4ns-system
   kubectl describe deployment np4ns-controller-manager -n np4ns-system
   ```

### NetworkPolicy Not Recreated After Deletion

**Symptoms**: Deleted NetworkPolicy doesn't come back

**Possible Causes**:

1. **Operator watch not catching deletion events**
   - Check operator logs for reconciliation events
   - The watch should catch all events including deletions

2. **Operator crashed or restarting**
   ```bash
   kubectl get pods -n np4ns-system
   kubectl logs -n np4ns-system deployment/np4ns-controller-manager --previous
   ```

### Wrong NetworkPolicy Spec

**Symptoms**: NetworkPolicy exists but has wrong spec

**Possible Cause**: Operator reconciliation hasn't run yet or failed

```bash
# Trigger manual reconciliation by adding an annotation
kubectl annotate namespace <namespace> reconcile=true --overwrite

# Check logs
kubectl logs -n np4ns-system deployment/np4ns-controller-manager | grep "NetworkPolicy is not compliant"
```

### Configuration Not Taking Effect

**Symptoms**: Changed ConfigMap but operator still uses old config

**Solution**: Restart the operator to pick up new environment variables

```bash
kubectl rollout restart deployment np4ns-controller-manager -n np4ns-system
```

### View Operator Logs with Verbosity

```bash
# Standard logs
kubectl logs -n np4ns-system deployment/np4ns-controller-manager -f

# With more detailed logging (if operator supports --zap-log-level flag)
kubectl set env deployment/np4ns-controller-manager -n np4ns-system ZAP_LOG_LEVEL=2
```

---

## Uninstalling

To remove the operator and all its resources:

```bash
# Undeploy the operator
make undeploy

# Remove test namespaces (optional)
kubectl delete -f config/samples/test-namespaces.yaml
kubectl delete -f config/samples/target-namespace-setup.yaml
```

**Note**: NetworkPolicies created by the operator have owner references to their namespaces, so they will be automatically deleted when the namespace is deleted.

---

## Advanced Configuration

### Running Locally (Development)

```bash
# Run operator locally (connects to cluster via kubeconfig).
# np4ns defines no CRDs, so there is nothing to install first.
make run
```

### Building Custom Images

```bash
# Build for specific platform
make docker-build IMG=your-registry/np4ns:v1.0.0

# Push to registry
make docker-push IMG=your-registry/np4ns:v1.0.0

# Deploy with custom image
make deploy IMG=your-registry/np4ns:v1.0.0
```

### Multi-Architecture Builds

```bash
# Build for multiple architectures
make docker-buildx IMG=your-registry/np4ns:v1.0.0 PLATFORMS=linux/amd64,linux/arm64
```

---

## Security Considerations

1. **Least Privilege**: The operator runs with minimal RBAC permissions:
   - Read/Write NetworkPolicies
   - Read/Update Namespaces (for annotations)

2. **Non-Root Container**: Runs as non-root user (UID 65532)

3. **No Privilege Escalation**: Security context prevents privilege escalation

4. **Network Policy Enforcement**: The enforced policies follow the principle of least privilege (deny-all with explicit allow)

---

## Support & Contributing

- **Issues**: Report bugs at [GitHub Issues](https://github.com/danieloa/np4ns/issues)
- **Documentation**: See main [README.md](README.md)
- **License**: Apache 2.0

---

## Quick Reference

### Useful Commands

```bash
# Check operator status
kubectl get all -n np4ns-system

# View operator logs
kubectl logs -n np4ns-system -l control-plane=controller-manager -f

# List all enforced policies
kubectl get networkpolicies --all-namespaces -l "!kubernetes.io/metadata.name" | grep enforced

# Check namespace annotations
kubectl get namespaces -o custom-columns=NAME:.metadata.name,ENFORCED:.metadata.annotations.network-policy/enforced

# Restart operator
kubectl rollout restart deployment np4ns-controller-manager -n np4ns-system

# Update configuration
kubectl edit configmap np4ns-config -n np4ns-system
```
