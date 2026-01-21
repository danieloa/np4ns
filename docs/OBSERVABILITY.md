# Observability Guide

This guide covers monitoring, logging, and alerting for the np4ns operator.

## Table of Contents

- [Metrics](#metrics)
- [Logging](#logging)
- [Prometheus Integration](#prometheus-integration)
- [Grafana Dashboards](#grafana-dashboards)
- [Recommended Alerts](#recommended-alerts)
- [Troubleshooting with Observability](#troubleshooting-with-observability)

---

## Metrics

The np4ns operator exposes Prometheus-compatible metrics on port 8080 at the `/metrics` endpoint.

### Built-in Controller Runtime Metrics

The operator automatically exposes these standard controller-runtime metrics:

#### Reconciliation Metrics

```
# Number of reconciliations per controller
controller_runtime_reconcile_total{controller="namespace",result="success|error|requeue"}

# Reconciliation duration
controller_runtime_reconcile_time_seconds{controller="namespace"}

# Reconciliation errors
controller_runtime_reconcile_errors_total{controller="namespace"}
```

#### Workqueue Metrics

```
# Items in the workqueue
workqueue_depth{name="namespace"}

# Queue processing latency
workqueue_queue_duration_seconds{name="namespace"}

# Work duration
workqueue_work_duration_seconds{name="namespace"}

# Retries
workqueue_retries_total{name="namespace"}
```

#### Client Metrics

```
# API client requests
rest_client_requests_total{code="200|404|500",method="GET|POST|PATCH"}

# Client request duration
rest_client_request_duration_seconds{verb="GET|POST|PATCH"}
```

### Custom Metrics (Future Enhancement)

These custom metrics could be added:

```
# Network policies enforced
np4ns_network_policies_enforced_total{namespace="myapp"}

# Network policies recreated after deletion
np4ns_network_policies_recreated_total{namespace="myapp"}

# Non-compliant policies corrected
np4ns_network_policies_corrected_total{namespace="myapp"}

# Namespaces under management
np4ns_managed_namespaces_total

# Namespaces skipped (in exception list)
np4ns_skipped_namespaces_total
```

---

## Logging

### Log Levels

The operator uses structured logging with the following levels:

- **Error**: Serious issues requiring attention
- **Info**: Standard operational messages (default)
- **Debug**: Detailed information for troubleshooting (V=1)

### Viewing Logs

```bash
# Standard logs
kubectl logs -n np4ns-system deployment/np4ns-controller-manager

# Follow logs in real-time
kubectl logs -n np4ns-system deployment/np4ns-controller-manager -f

# Previous instance logs (if pod crashed)
kubectl logs -n np4ns-system deployment/np4ns-controller-manager --previous

# Logs for specific time range (requires logging backend)
kubectl logs -n np4ns-system deployment/np4ns-controller-manager --since=1h
```

### Key Log Messages

#### Successful Operations

```
# Namespace reconciliation started
INFO    Reconciling namespace    {"namespace": "myapp"}

# Network policy created
INFO    Creating enforced NetworkPolicy    {"namespace": "myapp"}
INFO    Successfully created enforced NetworkPolicy    {"namespace": "myapp", "policy": "enforced-network-policy"}

# Network policy updated to compliance
INFO    NetworkPolicy is not compliant, updating    {"namespace": "myapp", "policy": "enforced-network-policy"}
INFO    Successfully updated NetworkPolicy to be compliant    {"namespace": "myapp", "policy": "enforced-network-policy"}

# Namespace skipped
INFO    Skipping namespace (not in target list or in exception list)    {"namespace": "kube-system"}
```

#### Error Conditions

```
# Failed to get namespace
ERROR    failed to get namespace    {"error": "namespaces \"myapp\" not found"}

# Failed to create network policy
ERROR    failed to create NetworkPolicy    {"namespace": "myapp", "error": "..."}

# Failed to update network policy
ERROR    failed to update NetworkPolicy    {"namespace": "myapp", "error": "..."}

# Failed to update namespace annotation
ERROR    failed to update namespace annotation    {"namespace": "myapp", "error": "..."}
```

### Log Aggregation

For production deployments, consider using log aggregation:

#### ELK Stack Example

```yaml
# filebeat-config.yaml
filebeat.inputs:
- type: container
  paths:
    - /var/log/containers/np4ns-controller-manager-*.log
  processors:
    - add_kubernetes_metadata:
        host: ${NODE_NAME}
        matchers:
        - logs_path:
            logs_path: "/var/log/containers/"
```

#### Loki Example

```yaml
# promtail-config.yaml
scrape_configs:
  - job_name: kubernetes-pods
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_namespace]
        action: keep
        regex: np4ns-system
```

---

## Prometheus Integration

### ServiceMonitor (Prometheus Operator)

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: np4ns-metrics
  namespace: np4ns-system
  labels:
    app.kubernetes.io/name: np4ns
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
```

### Prometheus Scrape Config

```yaml
# prometheus-config.yaml
scrape_configs:
  - job_name: 'np4ns-operator'
    kubernetes_sd_configs:
      - role: endpoints
        namespaces:
          names:
            - np4ns-system
    relabel_configs:
      - source_labels: [__meta_kubernetes_service_label_control_plane]
        action: keep
        regex: controller-manager
      - source_labels: [__meta_kubernetes_endpoint_port_name]
        action: keep
        regex: metrics
```

### Exposing Metrics Service

The metrics are available through the manager pods, but you may want to create a Service:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: np4ns-metrics
  namespace: np4ns-system
  labels:
    control-plane: controller-manager
spec:
  selector:
    control-plane: controller-manager
  ports:
  - name: metrics
    port: 8080
    targetPort: 8080
    protocol: TCP
```

---

## Grafana Dashboards

### Example Dashboard JSON

Save this as `np4ns-dashboard.json`:

```json
{
  "dashboard": {
    "title": "np4ns Operator",
    "panels": [
      {
        "title": "Reconciliation Rate",
        "targets": [
          {
            "expr": "rate(controller_runtime_reconcile_total{controller=\"namespace\"}[5m])"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Reconciliation Errors",
        "targets": [
          {
            "expr": "rate(controller_runtime_reconcile_errors_total{controller=\"namespace\"}[5m])"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Reconciliation Duration (p95)",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(controller_runtime_reconcile_time_seconds_bucket{controller=\"namespace\"}[5m]))"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Workqueue Depth",
        "targets": [
          {
            "expr": "workqueue_depth{name=\"namespace\"}"
          }
        ],
        "type": "graph"
      },
      {
        "title": "API Client Requests",
        "targets": [
          {
            "expr": "rate(rest_client_requests_total[5m])"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Memory Usage",
        "targets": [
          {
            "expr": "container_memory_usage_bytes{namespace=\"np4ns-system\",container=\"manager\"}"
          }
        ],
        "type": "graph"
      },
      {
        "title": "CPU Usage",
        "targets": [
          {
            "expr": "rate(container_cpu_usage_seconds_total{namespace=\"np4ns-system\",container=\"manager\"}[5m])"
          }
        ],
        "type": "graph"
      }
    ]
  }
}
```

### Import Dashboard

```bash
# Using Grafana API
curl -X POST http://grafana:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d @np4ns-dashboard.json
```

---

## Recommended Alerts

### Prometheus Alert Rules

```yaml
# prometheus-alerts.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: np4ns-alerts
  namespace: np4ns-system
spec:
  groups:
  - name: np4ns
    interval: 30s
    rules:
    # High reconciliation error rate
    - alert: NP4NSHighReconciliationErrors
      expr: rate(controller_runtime_reconcile_errors_total{controller="namespace"}[5m]) > 0.1
      for: 5m
      labels:
        severity: warning
        component: np4ns
      annotations:
        summary: "High reconciliation error rate in np4ns operator"
        description: "np4ns operator is experiencing {{ $value }} errors per second"

    # Operator pod down
    - alert: NP4NSOperatorDown
      expr: up{job="np4ns-operator"} == 0
      for: 5m
      labels:
        severity: critical
        component: np4ns
      annotations:
        summary: "np4ns operator is down"
        description: "No metrics from np4ns operator for 5 minutes"

    # High reconciliation latency
    - alert: NP4NSHighReconciliationLatency
      expr: histogram_quantile(0.95, rate(controller_runtime_reconcile_time_seconds_bucket{controller="namespace"}[5m])) > 5
      for: 10m
      labels:
        severity: warning
        component: np4ns
      annotations:
        summary: "High reconciliation latency in np4ns"
        description: "95th percentile reconciliation time is {{ $value }}s"

    # Large workqueue depth
    - alert: NP4NSLargeWorkqueue
      expr: workqueue_depth{name="namespace"} > 100
      for: 10m
      labels:
        severity: warning
        component: np4ns
      annotations:
        summary: "Large workqueue in np4ns operator"
        description: "Workqueue has {{ $value }} items pending"

    # High memory usage
    - alert: NP4NSHighMemoryUsage
      expr: container_memory_usage_bytes{namespace="np4ns-system",container="manager"} > 100000000
      for: 5m
      labels:
        severity: warning
        component: np4ns
      annotations:
        summary: "np4ns operator high memory usage"
        description: "Memory usage is {{ $value | humanize }}B"

    # API client errors
    - alert: NP4NSAPIClientErrors
      expr: rate(rest_client_requests_total{code=~"5.."}[5m]) > 0.1
      for: 5m
      labels:
        severity: warning
        component: np4ns
      annotations:
        summary: "High API client error rate"
        description: "{{ $value }} API errors per second"
```

### Alert Manager Configuration

```yaml
# alertmanager-config.yaml
route:
  group_by: ['alertname', 'component']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  receiver: 'team-infra'
  routes:
  - match:
      component: np4ns
      severity: critical
    receiver: 'team-infra-pager'

receivers:
- name: 'team-infra'
  slack_configs:
  - api_url: 'YOUR_SLACK_WEBHOOK'
    channel: '#infrastructure'
    title: 'np4ns Alert'
    text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'

- name: 'team-infra-pager'
  pagerduty_configs:
  - service_key: 'YOUR_PAGERDUTY_KEY'
```

---

## Troubleshooting with Observability

### High Reconciliation Errors

**Query:**
```promql
rate(controller_runtime_reconcile_errors_total{controller="namespace"}[5m])
```

**Investigation:**
1. Check logs for error details
2. Verify RBAC permissions
3. Check API server connectivity
4. Review recent configuration changes

### Slow Reconciliation

**Query:**
```promql
histogram_quantile(0.95, rate(controller_runtime_reconcile_time_seconds_bucket{controller="namespace"}[5m]))
```

**Investigation:**
1. Check for large number of namespaces
2. Review network policy complexity
3. Check API server performance
4. Consider increasing operator resources

### Memory Leaks

**Query:**
```promql
container_memory_usage_bytes{namespace="np4ns-system",container="manager"}
```

**Investigation:**
1. Monitor memory over time
2. Check for growing workqueue
3. Review goroutine count
4. Enable pprof for profiling

### Network Policy Not Created

**Logs to Check:**
```bash
# Look for skipped namespaces
kubectl logs -n np4ns-system deployment/np4ns-controller-manager | grep "Skipping namespace"

# Look for creation failures
kubectl logs -n np4ns-system deployment/np4ns-controller-manager | grep "failed to create"

# Check reconciliation events
kubectl logs -n np4ns-system deployment/np4ns-controller-manager | grep "Reconciling namespace"
```

**Metrics to Check:**
```promql
# Reconciliation errors for specific namespace
controller_runtime_reconcile_errors_total{controller="namespace"}

# Recent reconciliations
increase(controller_runtime_reconcile_total{controller="namespace"}[1h])
```

---

## Performance Profiling

### Enable pprof

Add to deployment:
```yaml
env:
- name: ENABLE_PPROF
  value: "true"
```

### Capture Profiles

```bash
# CPU profile
kubectl exec -n np4ns-system deployment/np4ns-controller-manager -- \
  curl -o /tmp/cpu.prof http://localhost:8080/debug/pprof/profile?seconds=30

# Heap profile
kubectl exec -n np4ns-system deployment/np4ns-controller-manager -- \
  curl -o /tmp/heap.prof http://localhost:8080/debug/pprof/heap

# Goroutine profile
kubectl exec -n np4ns-system deployment/np4ns-controller-manager -- \
  curl -o /tmp/goroutine.prof http://localhost:8080/debug/pprof/goroutine
```

### Analyze with go tool

```bash
# Copy profiles locally
kubectl cp np4ns-system/np4ns-controller-manager-xxx:/tmp/cpu.prof ./cpu.prof

# Analyze
go tool pprof cpu.prof
```

---

## Best Practices

1. **Always monitor in production**: Set up alerts before deployment
2. **Baseline metrics**: Establish normal operating ranges
3. **Alert tuning**: Start with conservative thresholds, tune based on experience
4. **Log retention**: Keep logs for at least 30 days
5. **Dashboard reviews**: Regularly review dashboards for trends
6. **Incident runbooks**: Document common issues and resolutions
7. **Capacity planning**: Monitor resource usage trends

## Additional Resources

- [Prometheus Operator Documentation](https://prometheus-operator.dev/)
- [Grafana Dashboard Best Practices](https://grafana.com/docs/grafana/latest/best-practices/)
- [Kubernetes Monitoring Guide](https://kubernetes.io/docs/tasks/debug-application-cluster/resource-usage-monitoring/)
- [Controller Runtime Metrics](https://book.kubebuilder.io/reference/metrics-reference.html)
