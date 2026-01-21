# Upgrade Guide

This guide covers upgrading np4ns to newer versions safely and efficiently.

## Table of Contents

- [Before You Upgrade](#before-you-upgrade)
- [Upgrade Methods](#upgrade-methods)
- [Version-Specific Notes](#version-specific-notes)
- [Rollback Procedure](#rollback-procedure)
- [Troubleshooting Upgrades](#troubleshooting-upgrades)

---

## Before You Upgrade

### Pre-Upgrade Checklist

- [ ] Review [CHANGELOG.md](../CHANGELOG.md) for breaking changes
- [ ] Check version compatibility with your Kubernetes version
- [ ] Backup current configuration
- [ ] Review current operator logs for issues
- [ ] Test upgrade in non-production environment first
- [ ] Schedule maintenance window (if required)
- [ ] Notify relevant teams

### Backup Current State

```bash
# Backup operator configuration
kubectl get configmap np4ns-config -n np4ns-system -o yaml > np4ns-config-backup.yaml

# Backup current deployment
kubectl get deployment np4ns-controller-manager -n np4ns-system -o yaml > np4ns-deployment-backup.yaml

# Backup all enforced network policies (for reference)
kubectl get networkpolicies --all-namespaces -l 'managed-by=np4ns' -o yaml > networkpolicies-backup.yaml

# Note: The operator doesn't use this label by default, so this might return empty
# Instead, backup by name:
for ns in $(kubectl get ns -o jsonpath='{.items[*].metadata.name}'); do
  kubectl get networkpolicy enforced-network-policy -n $ns -o yaml >> networkpolicies-backup.yaml 2>/dev/null
done
```

### Check Current Version

```bash
# Get current image version
kubectl get deployment np4ns-controller-manager -n np4ns-system -o jsonpath='{.spec.template.spec.containers[0].image}'

# Check operator logs for version info
kubectl logs -n np4ns-system deployment/np4ns-controller-manager | head -20
```

---

## Upgrade Methods

### Method 1: Helm Upgrade (Recommended)

#### Standard Upgrade

```bash
# Update Helm chart (if using chart repository)
helm repo update

# View available versions
helm search repo np4ns --versions

# Upgrade to latest version
helm upgrade np4ns charts/np4ns \
  --namespace np4ns-system \
  --reuse-values

# Or upgrade to specific version
helm upgrade np4ns charts/np4ns \
  --namespace np4ns-system \
  --version 0.0.6 \
  --reuse-values
```

#### Upgrade with New Values

```bash
# Upgrade with custom values
helm upgrade np4ns charts/np4ns \
  --namespace np4ns-system \
  --set image.tag=v0.0.6-abc1234 \
  --set resources.limits.memory=256Mi \
  --reuse-values

# Or use values file
helm upgrade np4ns charts/np4ns \
  --namespace np4ns-system \
  -f my-values.yaml
```

#### Verify Upgrade

```bash
# Check deployment status
kubectl rollout status deployment/np4ns-controller-manager -n np4ns-system

# Verify new image
kubectl get deployment np4ns-controller-manager -n np4ns-system -o jsonpath='{.spec.template.spec.containers[0].image}'

# Check operator logs
kubectl logs -n np4ns-system deployment/np4ns-controller-manager -f
```

### Method 2: kubectl Apply

#### Update Image Tag

```bash
# Edit deployment directly
kubectl set image deployment/np4ns-controller-manager \
  manager=ghcr.io/danieloa/np4ns:v0.0.6-abc1234 \
  -n np4ns-system

# Verify rollout
kubectl rollout status deployment/np4ns-controller-manager -n np4ns-system
```

#### Update via Manifest

```bash
# Update manifests
make deploy IMG=ghcr.io/danieloa/np4ns:v0.0.6-abc1234

# Monitor rollout
kubectl rollout status deployment/np4ns-controller-manager -n np4ns-system
```

### Method 3: GitOps (ArgoCD/Flux)

#### ArgoCD

```bash
# Update Application manifest
kubectl patch application np4ns-operator -n argocd \
  --type merge \
  -p '{"spec":{"source":{"targetRevision":"v0.0.6"}}}'

# Sync application
argocd app sync np4ns-operator

# Watch sync status
argocd app watch np4ns-operator
```

#### Flux

```yaml
# Update HelmRelease
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: np4ns
  namespace: np4ns-system
spec:
  chart:
    spec:
      version: 0.0.6  # Update this
  interval: 5m
  releaseName: np4ns
  values:
    image:
      tag: v0.0.6-abc1234  # Update this
```

---

## Version-Specific Notes

### Upgrading to v0.0.6 (Example)

**Changes:**
- New feature: X
- Breaking change: Y
- Configuration change: Z

**Required Actions:**
1. Update ConfigMap to include new field
2. Review RBAC changes
3. Test in staging first

**Example:**
```bash
# Update ConfigMap
kubectl edit configmap np4ns-config -n np4ns-system
# Add new field: NEW_CONFIG_OPTION

# Upgrade operator
helm upgrade np4ns charts/np4ns --version 0.0.6
```

### Upgrading to v0.0.5 (Example)

**Changes:**
- Improved reconciliation performance
- No breaking changes
- No configuration changes

**Actions:**
```bash
# Simple upgrade
helm upgrade np4ns charts/np4ns --version 0.0.5 --reuse-values
```

### Upgrading from v0.0.x to v0.1.0 (Future)

**Potential Breaking Changes:**
- May introduce CRDs
- May require RBAC updates
- May change default behavior

**Check CHANGELOG.md for specific instructions when available.**

---

## Rollback Procedure

### Automatic Rollback (Helm)

```bash
# Rollback to previous version
helm rollback np4ns -n np4ns-system

# Rollback to specific revision
helm rollback np4ns 2 -n np4ns-system

# Check rollback history
helm history np4ns -n np4ns-system
```

### Manual Rollback

#### Using kubectl

```bash
# Rollback deployment
kubectl rollout undo deployment/np4ns-controller-manager -n np4ns-system

# Verify rollback
kubectl rollout status deployment/np4ns-controller-manager -n np4ns-system

# Check previous revision
kubectl rollout history deployment/np4ns-controller-manager -n np4ns-system
```

#### Using Backup

```bash
# Restore from backup
kubectl apply -f np4ns-deployment-backup.yaml
kubectl apply -f np4ns-config-backup.yaml

# Restart deployment
kubectl rollout restart deployment/np4ns-controller-manager -n np4ns-system
```

### Verify Rollback

```bash
# Check current version
kubectl get deployment np4ns-controller-manager -n np4ns-system \
  -o jsonpath='{.spec.template.spec.containers[0].image}'

# Check operator is functional
kubectl logs -n np4ns-system deployment/np4ns-controller-manager -f

# Verify network policies are still managed
kubectl get networkpolicies --all-namespaces | grep enforced-network-policy
```

---

## Zero-Downtime Upgrades

### Using Multiple Replicas

```bash
# Before upgrade, ensure multiple replicas
helm upgrade np4ns charts/np4ns \
  --set replicaCount=3 \
  --set leaderElection.enabled=true

# Then perform upgrade
helm upgrade np4ns charts/np4ns --version 0.0.6

# Leader election ensures continuous operation
```

### Rolling Update Strategy

```yaml
# Ensure deployment uses RollingUpdate
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxUnavailable: 0
    maxSurge: 1
```

### PodDisruptionBudget

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: np4ns-pdb
  namespace: np4ns-system
spec:
  minAvailable: 1
  selector:
    matchLabels:
      control-plane: controller-manager
```

---

## Post-Upgrade Verification

### Verification Checklist

```bash
# 1. Check deployment status
kubectl get deployment -n np4ns-system
kubectl rollout status deployment/np4ns-controller-manager -n np4ns-system

# 2. Verify pods are running
kubectl get pods -n np4ns-system

# 3. Check operator logs for errors
kubectl logs -n np4ns-system deployment/np4ns-controller-manager --tail=100

# 4. Verify configuration
kubectl get configmap np4ns-config -n np4ns-system -o yaml

# 5. Test namespace creation
kubectl create namespace upgrade-test
sleep 10
kubectl get networkpolicy enforced-network-policy -n upgrade-test

# 6. Test policy recreation
kubectl delete networkpolicy enforced-network-policy -n upgrade-test
sleep 10
kubectl get networkpolicy enforced-network-policy -n upgrade-test

# 7. Cleanup test namespace
kubectl delete namespace upgrade-test

# 8. Check metrics endpoint
kubectl port-forward -n np4ns-system deployment/np4ns-controller-manager 8080:8080 &
curl http://localhost:8080/metrics | grep controller_runtime
kill %1

# 9. Verify health endpoints
kubectl port-forward -n np4ns-system deployment/np4ns-controller-manager 8081:8081 &
curl http://localhost:8081/healthz
curl http://localhost:8081/readyz
kill %1
```

### Monitoring After Upgrade

Monitor these metrics for 24-48 hours after upgrade:

```promql
# Reconciliation errors
rate(controller_runtime_reconcile_errors_total[5m])

# Reconciliation duration
histogram_quantile(0.95, rate(controller_runtime_reconcile_time_seconds_bucket[5m]))

# Workqueue depth
workqueue_depth{name="namespace"}

# Pod restarts
kube_pod_container_status_restarts_total{namespace="np4ns-system"}

# Memory usage
container_memory_usage_bytes{namespace="np4ns-system"}
```

---

## Troubleshooting Upgrades

### Upgrade Fails to Start

**Symptoms:**
- New pods fail to start
- ImagePullBackOff errors
- CrashLoopBackOff

**Investigation:**
```bash
# Check pod status
kubectl get pods -n np4ns-system

# Check events
kubectl get events -n np4ns-system --sort-by='.lastTimestamp'

# Check pod logs
kubectl logs -n np4ns-system deployment/np4ns-controller-manager

# Describe pod
kubectl describe pod -n np4ns-system -l control-plane=controller-manager
```

**Solutions:**
1. Verify image tag exists
2. Check image registry accessibility
3. Review resource limits
4. Check RBAC permissions

### Reconciliation Errors After Upgrade

**Symptoms:**
- High error rate in logs
- Policies not created/updated
- Increasing workqueue depth

**Investigation:**
```bash
# Check logs for errors
kubectl logs -n np4ns-system deployment/np4ns-controller-manager | grep ERROR

# Check RBAC
kubectl auth can-i --as=system:serviceaccount:np4ns-system:np4ns-controller-manager get namespaces
kubectl auth can-i --as=system:serviceaccount:np4ns-system:np4ns-controller-manager create networkpolicies

# Check configuration
kubectl get configmap np4ns-config -n np4ns-system -o yaml
```

**Solutions:**
1. Verify RBAC permissions updated
2. Check configuration changes
3. Review breaking changes in CHANGELOG
4. Rollback if necessary

### Performance Degradation

**Symptoms:**
- Slow reconciliation
- High CPU/memory usage
- API server errors

**Investigation:**
```bash
# Check resource usage
kubectl top pod -n np4ns-system

# Check metrics
kubectl port-forward -n np4ns-system deployment/np4ns-controller-manager 8080:8080
curl http://localhost:8080/metrics | grep -E '(reconcile|workqueue|memory)'

# Check for resource contention
kubectl describe node
```

**Solutions:**
1. Increase resource limits
2. Check for regression in new version
3. Review cluster capacity
4. Consider rollback

### Config Changes Not Applied

**Symptoms:**
- Operator using old configuration
- Changes to ConfigMap not taking effect

**Solution:**
```bash
# Restart operator to pick up new config
kubectl rollout restart deployment/np4ns-controller-manager -n np4ns-system

# Verify new config loaded
kubectl logs -n np4ns-system deployment/np4ns-controller-manager | head -50
```

---

## Best Practices

1. **Always test in staging first**: Never upgrade production directly
2. **Read the CHANGELOG**: Understand what's changing
3. **Monitor during upgrade**: Watch logs and metrics
4. **Use Helm**: Easier rollback and versioning
5. **Maintain backups**: Keep configuration backups
6. **Schedule appropriately**: Upgrade during low-traffic periods
7. **Document customizations**: Track any custom configurations
8. **Verify thoroughly**: Complete post-upgrade checklist
9. **Communicate**: Inform teams of upgrade schedule
10. **Have rollback plan**: Be prepared to rollback if needed

---

## Automation

### Automated Upgrade Testing

```bash
#!/bin/bash
# automated-upgrade-test.sh

set -e

NAMESPACE="np4ns-system"
NEW_VERSION="v0.0.6-abc1234"

echo "Starting upgrade test..."

# Backup current state
kubectl get deployment -n $NAMESPACE -o yaml > backup-deployment.yaml
kubectl get configmap -n $NAMESPACE -o yaml > backup-config.yaml

# Perform upgrade
helm upgrade np4ns charts/np4ns --set image.tag=$NEW_VERSION --wait

# Wait for rollout
kubectl rollout status deployment/np4ns-controller-manager -n $NAMESPACE

# Run tests
echo "Running verification tests..."
kubectl create namespace upgrade-test-$(date +%s)
sleep 15

# Verify policy created
if kubectl get networkpolicy enforced-network-policy -n upgrade-test-* > /dev/null 2>&1; then
  echo "✓ Network policy created successfully"
else
  echo "✗ Network policy creation failed"
  exit 1
fi

# Cleanup
kubectl delete namespace upgrade-test-*

echo "Upgrade test completed successfully"
```

### CI/CD Integration

```yaml
# .github/workflows/upgrade-test.yaml
name: Upgrade Test

on:
  push:
    tags:
      - 'v*'

jobs:
  upgrade-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: helm/kind-action@v1
      - name: Deploy previous version
        run: |
          helm install np4ns charts/np4ns --set image.tag=v0.0.5
          kubectl wait --for=condition=available deployment/np4ns-controller-manager -n np4ns-system
      - name: Upgrade to new version
        run: |
          helm upgrade np4ns charts/np4ns --set image.tag=${{ github.ref_name }}
          kubectl rollout status deployment/np4ns-controller-manager -n np4ns-system
      - name: Run tests
        run: ./tests/upgrade-verification.sh
```

---

## Additional Resources

- [CHANGELOG.md](../CHANGELOG.md) - Version history and changes
- [Helm Upgrade Documentation](https://helm.sh/docs/helm/helm_upgrade/)
- [Kubernetes Rolling Updates](https://kubernetes.io/docs/tutorials/kubernetes-basics/update/update-intro/)
- [GitOps Upgrade Patterns](https://www.weave.works/blog/gitops-upgrade-patterns)
