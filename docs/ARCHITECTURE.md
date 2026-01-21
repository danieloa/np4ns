# Architecture Documentation

This document describes the architecture and design decisions of the np4ns operator.

## Table of Contents

- [Overview](#overview)
- [Architecture Diagram](#architecture-diagram)
- [Components](#components)
- [Reconciliation Loop](#reconciliation-loop)
- [Design Decisions](#design-decisions)
- [Data Flow](#data-flow)
- [RBAC Model](#rbac-model)
- [Future Enhancements](#future-enhancements)

---

## Overview

np4ns is a Kubernetes operator built using the Operator SDK and controller-runtime framework. It watches Namespace resources and ensures that each target namespace has a compliant NetworkPolicy enforced.

### Key Characteristics

- **Declarative**: Continuously reconciles desired state
- **Self-healing**: Automatically recreates deleted policies
- **Non-intrusive**: Works with native Kubernetes resources, no CRDs
- **Configurable**: Environment variable-based configuration
- **Reliable**: Leader election for high availability

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Kubernetes Cluster                          │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │                    Kubernetes API Server                      │ │
│  │                                                               │ │
│  │  Resources:                                                   │ │
│  │  - Namespaces (core/v1)                                      │ │
│  │  - NetworkPolicies (networking.k8s.io/v1)                    │ │
│  └─────────────▲──────────────────────────────▲─────────────────┘ │
│                │                               │                   │
│                │ Watch/List                    │ Create/Update     │
│                │ Namespaces                    │ NetworkPolicies   │
│                │                               │                   │
│  ┌─────────────┴───────────────────────────────┴─────────────────┐ │
│  │              np4ns-system Namespace                           │ │
│  │                                                               │ │
│  │  ┌─────────────────────────────────────────────────────────┐ │ │
│  │  │           np4ns-controller-manager Pod                  │ │ │
│  │  │                                                         │ │ │
│  │  │  ┌──────────────────────────────────────────────────┐ │ │ │
│  │  │  │          Manager Container                        │ │ │ │
│  │  │  │                                                   │ │ │ │
│  │  │  │  ┌─────────────────────────────────────────────┐ │ │ │ │
│  │  │  │  │      Namespace Controller                   │ │ │ │ │
│  │  │  │  │                                             │ │ │ │ │
│  │  │  │  │  - Watch Namespaces                         │ │ │ │ │
│  │  │  │  │  - Watch NetworkPolicies                    │ │ │ │ │
│  │  │  │  │  - Reconcile Loop                           │ │ │ │ │
│  │  │  │  │  - Enforce Compliance                       │ │ │ │ │
│  │  │  │  └─────────────────────────────────────────────┘ │ │ │ │
│  │  │  │                                                   │ │ │ │
│  │  │  │  ┌─────────────────────────────────────────────┐ │ │ │ │
│  │  │  │  │       Leader Election                       │ │ │ │ │
│  │  │  │  │  (coordination.k8s.io/v1 Lease)            │ │ │ │ │
│  │  │  │  └─────────────────────────────────────────────┘ │ │ │ │
│  │  │  │                                                   │ │ │ │
│  │  │  │  ┌─────────────────────────────────────────────┐ │ │ │ │
│  │  │  │  │      Health & Metrics                       │ │ │ │ │
│  │  │  │  │  - /healthz (port 8081)                     │ │ │ │ │
│  │  │  │  │  - /readyz (port 8081)                      │ │ │ │ │
│  │  │  │  │  - /metrics (port 8080)                     │ │ │ │ │
│  │  │  │  └─────────────────────────────────────────────┘ │ │ │ │
│  │  │  │                                                   │ │ │ │
│  │  │  └───────────────────────────────────────────────────┘ │ │ │
│  │  │                                                         │ │ │
│  │  │  Environment (from ConfigMap):                          │ │ │
│  │  │  - NS_EXCEPTION_LIST                                    │ │ │
│  │  │  - NS_TARGET_FOR_NP                                     │ │ │
│  │  └─────────────────────────────────────────────────────────┘ │ │
│  │                                                               │ │
│  │  ┌─────────────────────────────────────────────────────────┐ │ │
│  │  │              np4ns-config ConfigMap                     │ │ │
│  │  │  - NS_EXCEPTION_LIST                                    │ │ │
│  │  │  - NS_TARGET_FOR_NP                                     │ │ │
│  │  └─────────────────────────────────────────────────────────┘ │ │
│  │                                                               │ │
│  └───────────────────────────────────────────────────────────────┘ │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │                    Target Namespaces                         │ │
│  │                                                              │ │
│  │  Namespace: myapp                                            │ │
│  │  ├── NetworkPolicy: enforced-network-policy (managed)       │ │
│  │  ├── NetworkPolicy: custom-ingress-policy (user-managed)    │ │
│  │  └── Annotation: network-policy/enforced=<timestamp>        │ │
│  │                                                              │ │
│  │  Namespace: team-a                                           │ │
│  │  ├── NetworkPolicy: enforced-network-policy (managed)       │ │
│  │  └── Annotation: network-policy/enforced=<timestamp>        │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Components

### 1. Namespace Controller

The core component that implements the reconciliation logic.

**Location**: `internal/controller/namespace_controller.go`

**Responsibilities**:
- Watch Namespace resources
- Watch NetworkPolicy resources (specifically `enforced-network-policy`)
- Determine which namespaces need enforcement
- Create, update, or ensure NetworkPolicy compliance
- Update namespace annotations

**Key Functions**:
- `Reconcile()`: Main reconciliation loop
- `shouldEnforceNetworkPolicy()`: Determines if namespace needs enforcement
- `buildCompliantNetworkPolicySpec()`: Defines the desired NetworkPolicy spec
- `updateNamespaceAnnotation()`: Tracks enforcement with timestamps

### 2. Configuration System

**Location**: Environment variables loaded from ConfigMap

**Parameters**:
- `NS_EXCEPTION_LIST`: Comma-separated list of namespaces to exclude
- `NS_TARGET_FOR_NP`: Comma-separated list of namespaces to include (empty = all)

**Functions**:
- `getNamespaceExceptionList()`: Parses exception list
- `getNamespaceTargetList()`: Parses target list

### 3. RBAC Components

**ClusterRole**: Grants permissions to:
- Read/watch/update Namespaces
- Full CRUD on NetworkPolicies

**Role** (Leader Election): Grants permissions to:
- Manage ConfigMaps (for leader election)
- Manage Leases (coordination.k8s.io)
- Create Events

### 4. Workqueue & Caching

Provided by controller-runtime:
- **Cache**: In-memory cache of watched resources
- **Workqueue**: Rate-limited queue for reconciliation requests
- **Informers**: Watch API server for resource changes

---

## Reconciliation Loop

### Trigger Events

Reconciliation is triggered by:
1. Namespace creation
2. Namespace update (labels, annotations)
3. NetworkPolicy creation (named `enforced-network-policy`)
4. NetworkPolicy update (spec changes)
5. NetworkPolicy deletion
6. Periodic resync (default: 10 hours)

### Reconciliation Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    Reconciliation Triggered                     │
│                   (Namespace or NetworkPolicy)                  │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
                    ┌──────────────────────┐
                    │  Get Namespace Name  │
                    └──────────┬───────────┘
                               │
                               ▼
              ┌────────────────────────────────────┐
              │  Check shouldEnforceNetworkPolicy() │
              └─────────┬──────────────────────────┘
                        │
           ┌────────────┴────────────┐
           │                         │
           ▼ No                      ▼ Yes
    ┌────────────┐          ┌──────────────────┐
    │  Skip &    │          │  Fetch Namespace │
    │  Return    │          │   from API       │
    └────────────┘          └────────┬─────────┘
                                     │
                                     ▼
                    ┌────────────────────────────────────┐
                    │  Check if enforced-network-policy  │
                    │            exists                  │
                    └─────────┬──────────────────────────┘
                              │
                 ┌────────────┴───────────┐
                 │                        │
                 ▼ Not Found              ▼ Exists
        ┌────────────────────┐   ┌──────────────────────┐
        │  Create            │   │  Compare actual spec │
        │  NetworkPolicy     │   │  with desired spec   │
        └────────┬───────────┘   └──────────┬───────────┘
                 │                           │
                 │              ┌────────────┴───────────┐
                 │              │                        │
                 │              ▼ Compliant              ▼ Non-compliant
                 │       ┌────────────┐         ┌───────────────┐
                 │       │  Log &     │         │  Update       │
                 │       │  Return    │         │  NetworkPolicy│
                 │       └────────────┘         └───────┬───────┘
                 │                                      │
                 └──────────────┬───────────────────────┘
                                │
                                ▼
                  ┌──────────────────────────────┐
                  │  Update Namespace Annotation │
                  │  (network-policy/enforced)   │
                  └──────────────┬───────────────┘
                                 │
                                 ▼
                          ┌────────────┐
                          │  Success   │
                          │   Return   │
                          └────────────┘
```

### Error Handling

- **Transient errors** (network issues, API server busy): Requeue with exponential backoff
- **Not found errors**: Ignored (resource may have been deleted)
- **Permanent errors**: Logged and not requeued (requires operator restart or manual intervention)

---

## Design Decisions

### 1. No Custom Resource Definitions (CRDs)

**Decision**: Use native Kubernetes resources only.

**Rationale**:
- Simpler deployment (no CRD installation required)
- Lower complexity for users
- Works with standard RBAC
- Easier to understand and debug

**Trade-offs**:
- Less flexibility for complex policy definitions
- Customization requires code changes
- No declarative API for policy templates

### 2. ConfigMap-Based Configuration

**Decision**: Use environment variables from ConfigMap for operator configuration.

**Rationale**:
- Standard Kubernetes pattern
- Easy to update without code changes
- Works with GitOps workflows
- Clear separation of config and code

**Trade-offs**:
- Requires operator restart for config changes
- Limited validation of configuration
- No per-namespace customization

### 3. Single NetworkPolicy per Namespace

**Decision**: Manage one NetworkPolicy named `enforced-network-policy` per namespace.

**Rationale**:
- Clear ownership and responsibility
- No conflicts with user-defined policies
- Simple reconciliation logic
- Easy to identify managed policies

**Trade-offs**:
- Cannot manage multiple policies per namespace
- Users must create additional policies manually
- Single policy must cover all enforcement requirements

### 4. Namespace Annotations for Tracking

**Decision**: Add `network-policy/enforced` annotation with timestamp.

**Rationale**:
- Audit trail of when enforcement occurred
- Easy to query namespaces under management
- Non-intrusive (doesn't affect functionality)
- Standard Kubernetes practice

### 5. Owner References

**Decision**: Set namespace as owner of NetworkPolicy.

**Rationale**:
- Automatic cleanup when namespace is deleted
- Clear relationship between resources
- Garbage collection handled by Kubernetes
- Prevents orphaned policies

**Trade-offs**:
- Policy cannot outlive namespace
- Policy cannot be adopted by other controllers

### 6. Continuous Reconciliation

**Decision**: Continuously reconcile and self-heal.

**Rationale**:
- Ensures compliance even after manual changes
- Recovers from deletions automatically
- Handles configuration drift
- Standard operator pattern

### 7. Leader Election

**Decision**: Support leader election for HA deployments.

**Rationale**:
- Enables running multiple replicas
- Automatic failover on pod failure
- Standard HA pattern for operators
- Prevents conflicting updates

---

## Data Flow

### Startup Sequence

```
1. Operator pod starts
2. Load configuration from environment (ConfigMap)
3. Establish connection to API server
4. Start leader election (if enabled)
5. Start cache and wait for sync
6. Start Namespace controller
7. Start watching Namespaces
8. Start watching NetworkPolicies
9. Trigger initial reconciliation for all namespaces
10. Enter event loop
```

### Watch Flow

```
API Server → Informer → Cache → Workqueue → Reconciler → API Server
     │                                            │
     └────────────────────────────────────────────┘
            (Watch events and updates)
```

### Create NetworkPolicy Flow

```
1. Reconciler determines NetworkPolicy needed
2. Build desired NetworkPolicy spec
3. Set owner reference to Namespace
4. Call API server to create NetworkPolicy
5. API server validates and persists
6. Update namespace annotation with timestamp
7. Cache is updated by informer
8. Reconciliation completes
```

---

## RBAC Model

### Cluster-Level Permissions

```yaml
# What the operator can do cluster-wide
ClusterRole:
  - Namespaces: get, list, watch, update, patch
  - NetworkPolicies: get, list, watch, create, update, patch, delete
```

### Namespace-Level Permissions

```yaml
# What the operator can do in its own namespace (np4ns-system)
Role:
  - ConfigMaps: get, list, watch, create, update, patch, delete
  - Leases: get, list, watch, create, update, patch, delete
  - Events: create, patch
```

### Security Considerations

- **Least privilege**: Operator has minimal permissions required
- **No escalation**: Cannot modify RBAC or cluster config
- **Audit trail**: All actions logged and traceable
- **Non-root**: Runs as non-root user (UID 65532)
- **No privileged containers**: Security context drops all capabilities

---

## Future Enhancements

### Under Consideration

1. **Custom Policy Templates**
   - ConfigMap or CRD-based policy definitions
   - Per-namespace policy customization
   - Policy template library

2. **Enhanced Observability**
   - Custom metrics for policy enforcement
   - Policy compliance dashboard
   - Alert integration

3. **Policy Validation**
   - Validate policies before enforcement
   - Dry-run mode
   - Impact analysis

4. **Webhook Integration**
   - Validating webhook to prevent policy deletion
   - Mutating webhook to auto-label namespaces
   - Admission control integration

5. **Multi-Policy Support**
   - Manage multiple policies per namespace
   - Policy priority and ordering
   - Policy composition

6. **Advanced Configuration**
   - Namespace selectors (label-based targeting)
   - Policy overrides per namespace
   - Time-based policy application

### Architectural Implications

These enhancements may require:
- Introduction of CRDs
- Additional controllers
- Webhook server component
- More complex reconciliation logic
- Enhanced RBAC permissions

---

## Decision Records

### Why Operator SDK vs Custom Controller?

**Decision**: Use Operator SDK framework.

**Reasons**:
- Reduces boilerplate code
- Provides best practices out of the box
- Built-in leader election, metrics, health checks
- Active community and support
- Easier to maintain and extend

### Why Watch NetworkPolicies?

**Decision**: Watch NetworkPolicy resources in addition to Namespaces.

**Reasons**:
- Detect deletions immediately
- Respond to manual modifications
- Faster reconciliation than polling
- Standard controller pattern

### Why Build from Scratch vs Fork?

**Decision**: Built as new operator rather than extending existing tools.

**Reasons**:
- Specific use case (namespace-level enforcement)
- Simpler implementation
- No dependencies on larger frameworks
- Easier to customize and maintain

---

## Additional Resources

- [Operator Pattern Documentation](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Operator SDK](https://sdk.operatorframework.io/)
- [Kubernetes Network Policies](https://kubernetes.io/docs/concepts/services-networking/network-policies/)
