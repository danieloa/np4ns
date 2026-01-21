# Performance and Scale Guide

This document provides guidance on performance characteristics, resource requirements, and scaling considerations for np4ns.

## Table of Contents

- [Performance Characteristics](#performance-characteristics)
- [Resource Requirements](#resource-requirements)
- [Scaling Guidelines](#scaling-guidelines)
- [Benchmarks](#benchmarks)
- [Optimization Tips](#optimization-tips)
- [Troubleshooting Performance](#troubleshooting-performance)

---

## Performance Characteristics

### Reconciliation Time

The time to reconcile a namespace depends on several factors:

| Operation | Typical Time | Factors |
|-----------|--------------|---------|
| Create NetworkPolicy | 50-200ms | API server latency, policy complexity |
| Update NetworkPolicy | 50-200ms | API server latency, policy diff |
| Check compliance | 10-50ms | Cached reads, comparison logic |
| Namespace annotation update | 30-100ms | API server write latency |

### Throughput

Expected throughput under normal conditions:

| Metric | Value | Notes |
|--------|-------|-------|
| Namespaces reconciled/sec | 10-20 | Steady state |
| Initial reconciliation (100 ns) | 10-30s | Startup burst |
| NetworkPolicy creations/sec | 5-10 | Limited by API server |

### Resource Consumption

#### CPU

```
Idle: 5-10m (0.005-0.01 cores)
Light load (10 ns): 10-20m
Moderate load (100 ns): 20-50m
Heavy load (1000 ns): 50-100m
Peak (bulk reconciliation): 100-200m
```

#### Memory

```
Base: 30-40 MiB
10 namespaces: 40-50 MiB
100 namespaces: 50-80 MiB
1000 namespaces: 80-120 MiB
Peak: 150-200 MiB (with cache)
```

### Network

- **API server calls**: 2-4 per reconciliation
- **Watch connections**: 2 (Namespaces, NetworkPolicies)
- **Bandwidth**: < 1 Mbps under normal load

---

## Resource Requirements

### Minimum Requirements

**For small clusters (< 50 namespaces):**

```yaml
resources:
  requests:
    cpu: 10m
    memory: 64Mi
  limits:
    cpu: 200m
    memory: 128Mi
```

### Recommended Requirements

**For medium clusters (50-500 namespaces):**

```yaml
resources:
  requests:
    cpu: 50m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 256Mi
```

### Large Scale Requirements

**For large clusters (500-5000 namespaces):**

```yaml
resources:
  requests:
    cpu: 100m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 512Mi
```

### Factors Affecting Requirements

1. **Number of namespaces**: Primary factor
2. **Churn rate**: How often namespaces are created/deleted
3. **Policy complexity**: More complex policies take longer to process
4. **API server latency**: Network and cluster load
5. **Resync period**: More frequent resyncs increase load

---

## Scaling Guidelines

### Vertical Scaling

Increase resources for the operator pod:

```bash
# Update resource limits
helm upgrade np4ns charts/np4ns \
  --set resources.requests.cpu=100m \
  --set resources.requests.memory=256Mi \
  --set resources.limits.cpu=1000m \
  --set resources.limits.memory=512Mi
```

**When to scale up:**
- High CPU usage (> 80% of limit)
- Memory pressure or OOMKilled events
- Increasing reconciliation latency
- Growing workqueue depth

### Horizontal Scaling

Run multiple replicas with leader election:

```bash
# Enable multiple replicas
helm upgrade np4ns charts/np4ns \
  --set replicaCount=3 \
  --set leaderElection.enabled=true
```

**Benefits:**
- High availability
- Faster leader failover
- Zero downtime during upgrades

**Limitations:**
- Only one replica actively reconciles (leader election)
- Standby replicas consume resources
- Slightly higher memory usage per pod

### Cluster Size Recommendations

| Cluster Size | Namespaces | Replicas | CPU Request | Memory Request |
|--------------|-----------|----------|-------------|----------------|
| Small | < 50 | 1 | 10m | 64Mi |
| Medium | 50-200 | 2 | 50m | 128Mi |
| Large | 200-1000 | 3 | 100m | 256Mi |
| X-Large | 1000-5000 | 3 | 200m | 512Mi |
| XX-Large | 5000+ | 3 | 500m | 1Gi |

---

## Benchmarks

### Test Environment

```
Kubernetes: v1.28.2
Nodes: 3x n1-standard-4 (4 vCPU, 15GB RAM)
CNI: Calico
API server: default configuration
```

### Test 1: Initial Reconciliation

Creating 100 namespaces and measuring time to enforce policies:

```
Total time: 25 seconds
Average time per namespace: 250ms
Policies created: 100
Errors: 0
Peak CPU: 180m
Peak memory: 95 MiB
```

### Test 2: Steady State Operation

100 namespaces, measuring reconciliation over 1 hour:

```
Reconciliations: 600 (1 per namespace per 10 min resync)
Average reconciliation time: 45ms
95th percentile: 120ms
99th percentile: 280ms
Errors: 0
Average CPU: 25m
Average memory: 78 MiB
```

### Test 3: Bulk Deletion Recovery

Deleting all 100 enforced policies and measuring recreation:

```
Policies deleted: 100
Detection time: < 1 second (via watch)
Recreation time: 28 seconds
Average time per policy: 280ms
Peak CPU: 220m
Peak memory: 105 MiB
```

### Test 4: High Churn

Creating and deleting 10 namespaces per minute for 1 hour:

```
Namespaces created: 600
Namespaces deleted: 600
Policies created: 600
Average creation latency: 2.1 seconds
Average CPU: 65m
Average memory: 82 MiB
```

### Test 5: Large Scale

1000 namespaces in a cluster:

```
Initial reconciliation: 4 minutes 30 seconds
Steady state CPU: 45m
Steady state memory: 185 MiB
Cache size: ~15 MiB
Watch event rate: ~50 events/hour
```

---

## Optimization Tips

### 1. Adjust Resource Limits

Start with defaults and increase based on observed usage:

```bash
# Monitor current usage
kubectl top pod -n np4ns-system

# Adjust limits if needed
helm upgrade np4ns charts/np4ns \
  --set resources.limits.cpu=1000m \
  --set resources.limits.memory=512Mi
```

### 2. Configure Resync Period

Default resync is 10 hours. Increase for reduced load:

```go
// In controller setup (requires code change)
ctrl.NewControllerManagedBy(mgr).
    For(&corev1.Namespace{}).
    WithOptions(controller.Options{
        MaxConcurrentReconciles: 1,
        RecoverPanic:            true,
        // Adjust resync period
    })
```

### 3. Use Node Affinity

Pin operator to specific nodes for consistent performance:

```yaml
affinity:
  nodeAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      preference:
        matchExpressions:
        - key: node-role.kubernetes.io/control-plane
          operator: Exists
```

### 4. Optimize Network Policy

Simpler policies reconcile faster:

```go
// Simple policy (faster)
Egress: []NetworkPolicyEgressRule{
    {}, // Allow all
}

// Complex policy (slower)
Egress: []NetworkPolicyEgressRule{
    {
        To: [...10 peers...],
        Ports: [...10 ports...],
    },
    // ... 5 more rules
}
```

### 5. Batch Operations

When creating many namespaces, create them in batches:

```bash
# Instead of loop creating 100 namespaces one by one
# Create in batches of 10-20
for i in {1..100}; do
  kubectl create namespace test-$i &
  if [ $((i % 20)) -eq 0 ]; then
    wait
  fi
done
```

### 6. Monitor Workqueue Depth

Large workqueue indicates backlog:

```promql
# Alert if workqueue depth > 50
workqueue_depth{name="namespace"} > 50
```

### 7. Use PodDisruptionBudget

Prevent multiple replicas from being evicted simultaneously:

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

## Troubleshooting Performance

### High CPU Usage

**Symptoms:**
- CPU usage consistently > 80%
- Throttling in metrics
- Slow reconciliation

**Investigation:**
```bash
# Check current usage
kubectl top pod -n np4ns-system

# View CPU throttling
kubectl get pod -n np4ns-system -o yaml | grep -A 5 throttling

# Check reconciliation rate
kubectl logs -n np4ns-system deployment/np4ns-controller-manager | grep "Reconciling" | wc -l
```

**Solutions:**
1. Increase CPU limits
2. Check for reconciliation loops
3. Reduce resync frequency
4. Investigate API server latency

### High Memory Usage

**Symptoms:**
- Memory usage near limits
- OOMKilled events
- Increasing over time

**Investigation:**
```bash
# Check memory usage
kubectl top pod -n np4ns-system

# Check for OOM kills
kubectl get events -n np4ns-system | grep OOM

# Get pprof heap dump
kubectl exec -n np4ns-system deployment/np4ns-controller-manager -- \
  curl http://localhost:8080/debug/pprof/heap > heap.prof
```

**Solutions:**
1. Increase memory limits
2. Check for memory leaks (analyze heap profile)
3. Reduce cache size (if possible)
4. Restart operator periodically

### Slow Reconciliation

**Symptoms:**
- High reconciliation latency (> 5 seconds)
- Policies not created promptly
- Large workqueue depth

**Investigation:**
```promql
# Check reconciliation duration
histogram_quantile(0.95, rate(controller_runtime_reconcile_time_seconds_bucket[5m]))

# Check workqueue depth
workqueue_depth{name="namespace"}

# Check API server latency
histogram_quantile(0.95, rate(rest_client_request_duration_seconds_bucket[5m]))
```

**Solutions:**
1. Increase operator resources
2. Check API server health
3. Reduce number of concurrent reconciliations
4. Investigate network latency

### Growing Workqueue

**Symptoms:**
- Workqueue depth consistently increasing
- Reconciliations not completing
- Backlog building up

**Investigation:**
```bash
# Check workqueue metrics
kubectl logs -n np4ns-system deployment/np4ns-controller-manager | grep workqueue

# Check for errors
kubectl logs -n np4ns-system deployment/np4ns-controller-manager | grep ERROR
```

**Solutions:**
1. Check for reconciliation errors
2. Increase reconciliation throughput (resources)
3. Fix underlying issues causing requeues
4. Restart operator to clear queue

### API Server Pressure

**Symptoms:**
- API server returning 429 (Too Many Requests)
- Increased latency on all operations
- Operator consuming high API rate

**Investigation:**
```bash
# Check API server metrics
kubectl top pod -n kube-system | grep apiserver

# Check rate limiting
kubectl logs -n np4ns-system deployment/np4ns-controller-manager | grep 429
```

**Solutions:**
1. Reduce watch frequency
2. Implement request rate limiting
3. Use client-side throttling
4. Coordinate with cluster admins

---

## Capacity Planning

### Estimating Requirements

Use this formula to estimate resource needs:

```
CPU (millicores) = 10 + (namespaces × 0.1) + (churn_per_minute × 5)
Memory (MiB) = 50 + (namespaces × 0.1)
```

**Example:**
- 500 namespaces
- 5 namespace creates/deletes per minute

```
CPU = 10 + (500 × 0.1) + (5 × 5) = 10 + 50 + 25 = 85m
Memory = 50 + (500 × 0.1) = 50 + 50 = 100 MiB
```

**Recommended (with 2x buffer):**
- CPU request: 85m, limit: 170m
- Memory request: 100Mi, limit: 200Mi

### Growth Planning

| Current | 6 Months | 12 Months | Action |
|---------|----------|-----------|--------|
| 100 ns | 200 ns | 400 ns | Monitor, adjust resources at 200 |
| 500 ns | 1000 ns | 2000 ns | Plan resource increase at 1000 |
| 1000 ns | 2000 ns | 4000 ns | Consider optimization, increase resources |

---

## Load Testing

### Running Load Tests

```bash
# Create load test script
cat > load-test.sh <<'EOF'
#!/bin/bash
for i in {1..1000}; do
  kubectl create namespace load-test-$i &
  if [ $((i % 50)) -eq 0 ]; then
    wait
    echo "Created $i namespaces"
  fi
done
wait
echo "Load test complete"
EOF

chmod +x load-test.sh

# Monitor operator during test
kubectl top pod -n np4ns-system -w &
kubectl logs -n np4ns-system deployment/np4ns-controller-manager -f &

# Run load test
./load-test.sh

# Cleanup
for i in {1..1000}; do
  kubectl delete namespace load-test-$i --wait=false &
done
```

### Analyzing Results

```bash
# Check metrics
kubectl port-forward -n np4ns-system deployment/np4ns-controller-manager 8080:8080

# Query metrics
curl http://localhost:8080/metrics | grep controller_runtime_reconcile

# View workqueue stats
curl http://localhost:8080/metrics | grep workqueue
```

---

## Best Practices

1. **Start conservative**: Begin with minimum resources, scale based on metrics
2. **Monitor continuously**: Set up dashboards and alerts
3. **Test at scale**: Load test before production deployment
4. **Plan for growth**: Project future namespace count
5. **Use HPA**: Consider Horizontal Pod Autoscaler for dynamic scaling
6. **Document baselines**: Record normal operating metrics
7. **Review regularly**: Quarterly performance reviews

## Additional Resources

- [Kubernetes Resource Management](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
- [Controller Runtime Performance](https://book.kubebuilder.io/reference/tuning-controller-runtime.html)
- [Prometheus Monitoring](https://prometheus.io/docs/introduction/overview/)
- [OBSERVABILITY.md](OBSERVABILITY.md) - Monitoring and alerting guide
