# Real-World Examples

This document provides practical, real-world examples of using np4ns in different scenarios.

## Table of Contents

- [Multi-Tenant SaaS Platform](#multi-tenant-saas-platform)
- [Microservices Architecture](#microservices-architecture)
- [CI/CD Pipeline Integration](#cicd-pipeline-integration)
- [Service Mesh Integration](#service-mesh-integration)
- [Migration from Manual Policies](#migration-from-manual-policies)
- [Development vs Production](#development-vs-production)

---

## Multi-Tenant SaaS Platform

### Scenario

A SaaS platform where each customer has their own namespace. Network policies must be enforced to ensure tenant isolation.

### Configuration

```yaml
# values-saas.yaml
config:
  # Exclude system and monitoring namespaces
  nsExceptionList: "kube-system,kube-public,kube-node-lease,monitoring,logging,istio-system"

  # Enforce on all customer namespaces (prefix pattern)
  # Empty means all except exceptions - rely on namespace naming convention
  nsTargetForNp: ""

# Label customer namespaces for easy identification
```

### Deployment

```bash
# Deploy operator
helm install np4ns charts/np4ns -f values-saas.yaml

# Create customer namespaces with labels
kubectl create namespace customer-acme
kubectl label namespace customer-acme tenant=acme environment=production

kubectl create namespace customer-globex
kubectl label namespace customer-globex tenant=globex environment=production
```

### Verification

```bash
# Check that policies are created for customer namespaces
kubectl get networkpolicies --all-namespaces | grep enforced

# Verify tenant isolation
kubectl run test-pod -n customer-acme --image=busybox --command -- sleep 3600
kubectl exec -n customer-acme test-pod -- wget -O- http://service.customer-globex.svc.cluster.local
# Should fail due to network policy
```

### Benefits

- Automatic network policy enforcement for new tenants
- Consistent security baseline across all customer namespaces
- Reduced risk of misconfiguration
- Audit trail through namespace annotations

---

## Microservices Architecture

### Scenario

A microservices application where services need controlled communication between layers (frontend, backend, database).

### Architecture

```
┌─────────────┐
│  Frontend   │ (namespace: frontend)
│  Namespace  │
└──────┬──────┘
       │ ✓ Can call backend on port 8080
       │
┌──────▼──────┐
│   Backend   │ (namespace: backend)
│  Namespace  │
└──────┬──────┘
       │ ✓ Can call database on port 5432
       │
┌──────▼──────┐
│  Database   │ (namespace: database)
│  Namespace  │
└─────────────┘
```

### Configuration

```bash
# Deploy operator with selective enforcement
helm install np4ns charts/np4ns \
  --set config.nsTargetForNp="frontend,backend,database" \
  --set image.tag=custom-microservices
```

### Custom Network Policy (in code)

```go
// internal/controller/namespace_controller.go
func buildCompliantNetworkPolicySpec() networkingv1.NetworkPolicySpec {
    protocolTCP := corev1.ProtocolTCP
    port8080 := intstr.FromInt(8080)
    port5432 := intstr.FromInt(5432)
    dnsPort := intstr.FromInt(53)
    protocolUDP := corev1.ProtocolUDP

    return networkingv1.NetworkPolicySpec{
        PodSelector: metav1.LabelSelector{},
        PolicyTypes: []networkingv1.PolicyType{
            networkingv1.PolicyTypeEgress,
        },
        Egress: []networkingv1.NetworkPolicyEgressRule{
            // Allow DNS
            {
                To: []networkingv1.NetworkPolicyPeer{
                    {
                        NamespaceSelector: &metav1.LabelSelector{
                            MatchLabels: map[string]string{
                                "kubernetes.io/metadata.name": "kube-system",
                            },
                        },
                        PodSelector: &metav1.LabelSelector{
                            MatchLabels: map[string]string{
                                "k8s-app": "kube-dns",
                            },
                        },
                    },
                },
                Ports: []networkingv1.NetworkPolicyPort{
                    {Protocol: &protocolUDP, Port: &dnsPort},
                    {Protocol: &protocolTCP, Port: &dnsPort},
                },
            },
            // Allow backend communication
            {
                To: []networkingv1.NetworkPolicyPeer{
                    {
                        NamespaceSelector: &metav1.LabelSelector{
                            MatchLabels: map[string]string{
                                "tier": "backend",
                            },
                        },
                    },
                },
                Ports: []networkingv1.NetworkPolicyPort{
                    {Protocol: &protocolTCP, Port: &port8080},
                },
            },
            // Allow database communication
            {
                To: []networkingv1.NetworkPolicyPeer{
                    {
                        NamespaceSelector: &metav1.LabelSelector{
                            MatchLabels: map[string]string{
                                "tier": "database",
                            },
                        },
                    },
                },
                Ports: []networkingv1.NetworkPolicyPort{
                    {Protocol: &protocolTCP, Port: &port5432},
                },
            },
        },
    }
}
```

### Setup

```bash
# Create namespaces with appropriate labels
kubectl create namespace frontend
kubectl label namespace frontend tier=frontend

kubectl create namespace backend
kubectl label namespace backend tier=backend

kubectl create namespace database
kubectl label namespace database tier=database
```

---

## CI/CD Pipeline Integration

### Scenario

Integrate np4ns enforcement into a GitOps workflow where namespace creation and policy enforcement are automated.

### GitOps Structure

```
infra-repo/
├── namespaces/
│   ├── production/
│   │   ├── api-namespace.yaml
│   │   └── frontend-namespace.yaml
│   └── staging/
│       ├── api-namespace.yaml
│       └── frontend-namespace.yaml
└── np4ns/
    ├── np4ns-operator.yaml
    └── np4ns-config.yaml
```

### ArgoCD Application

```yaml
# argocd/np4ns-app.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: np4ns-operator
  namespace: argocd
spec:
  project: infrastructure
  source:
    repoURL: https://github.com/yourorg/infra-repo
    targetRevision: main
    path: np4ns
  destination:
    server: https://kubernetes.default.svc
    namespace: np4ns-system
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

### Automated Verification

```yaml
# .github/workflows/verify-policies.yaml
name: Verify Network Policies

on:
  pull_request:
    paths:
      - 'namespaces/**'

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup kubectl
        uses: azure/setup-kubectl@v3

      - name: Create Kind cluster
        uses: helm/kind-action@v1

      - name: Deploy np4ns
        run: |
          helm install np4ns charts/np4ns
          kubectl wait --for=condition=available --timeout=60s deployment/np4ns-controller-manager -n np4ns-system

      - name: Apply namespace manifests
        run: |
          kubectl apply -f namespaces/

      - name: Verify policies created
        run: |
          # Wait for reconciliation
          sleep 10

          # Check that all namespaces have policies
          NAMESPACES=$(kubectl get ns -o jsonpath='{.items[*].metadata.name}')
          for ns in $NAMESPACES; do
            if [[ ! "$ns" =~ ^(kube-|default) ]]; then
              kubectl get networkpolicy enforced-network-policy -n $ns || exit 1
            fi
          done

      - name: Run compliance tests
        run: |
          # Add your compliance tests here
          ./tests/verify-policies.sh
```

---

## Service Mesh Integration

### Scenario

Using np4ns alongside a service mesh (Istio/Linkerd) for defense-in-depth.

### Architecture

Network policies provide layer 3/4 security, while service mesh provides layer 7 security and observability.

### Configuration

```yaml
# values-with-mesh.yaml
config:
  # Exclude mesh system namespaces
  nsExceptionList: "kube-system,kube-public,kube-node-lease,istio-system,linkerd,linkerd-viz"

  # Enforce on application namespaces
  nsTargetForNp: ""

# Allow service mesh sidecar communication
```

### Custom Policy for Istio

```go
func buildCompliantNetworkPolicySpec() networkingv1.NetworkPolicySpec {
    protocolTCP := corev1.ProtocolTCP

    return networkingv1.NetworkPolicySpec{
        PodSelector: metav1.LabelSelector{},
        PolicyTypes: []networkingv1.PolicyType{
            networkingv1.PolicyTypeIngress,
            networkingv1.PolicyTypeEgress,
        },
        Ingress: []networkingv1.NetworkPolicyIngressRule{
            // Allow from same namespace
            {
                From: []networkingv1.NetworkPolicyPeer{
                    {PodSelector: &metav1.LabelSelector{}},
                },
            },
            // Allow from Istio ingress gateway
            {
                From: []networkingv1.NetworkPolicyPeer{
                    {
                        NamespaceSelector: &metav1.LabelSelector{
                            MatchLabels: map[string]string{
                                "kubernetes.io/metadata.name": "istio-system",
                            },
                        },
                    },
                },
            },
        },
        Egress: []networkingv1.NetworkPolicyEgressRule{
            // Allow to same namespace
            {
                To: []networkingv1.NetworkPolicyPeer{
                    {PodSelector: &metav1.LabelSelector{}},
                },
            },
            // Allow to Istio control plane
            {
                To: []networkingv1.NetworkPolicyPeer{
                    {
                        NamespaceSelector: &metav1.LabelSelector{
                            MatchLabels: map[string]string{
                                "kubernetes.io/metadata.name": "istio-system",
                            },
                        },
                    },
                },
            },
            // Allow DNS
            {
                To: []networkingv1.NetworkPolicyPeer{
                    {
                        NamespaceSelector: &metav1.LabelSelector{
                            MatchLabels: map[string]string{
                                "kubernetes.io/metadata.name": "kube-system",
                            },
                        },
                    },
                },
                Ports: []networkingv1.NetworkPolicyPort{
                    {
                        Protocol: &protocolTCP,
                        Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 53},
                    },
                },
            },
        },
    }
}
```

---

## Migration from Manual Policies

### Scenario

You have existing network policies and want to migrate to automated enforcement with np4ns.

### Migration Steps

#### Phase 1: Assessment

```bash
# List all existing network policies
kubectl get networkpolicies --all-namespaces

# Export existing policies for backup
kubectl get networkpolicies --all-namespaces -o yaml > existing-policies-backup.yaml

# Analyze policy coverage
for ns in $(kubectl get ns -o jsonpath='{.items[*].metadata.name}'); do
  count=$(kubectl get networkpolicies -n $ns --no-headers 2>/dev/null | wc -l)
  echo "$ns: $count policies"
done
```

#### Phase 2: Test in Non-Production

```bash
# Deploy np4ns to staging cluster
helm install np4ns charts/np4ns \
  --set config.nsTargetForNp="staging-*" \
  --namespace np4ns-system \
  --create-namespace

# Monitor for 24-48 hours
kubectl logs -n np4ns-system deployment/np4ns-controller-manager -f

# Check for application issues
# Review monitoring dashboards
```

#### Phase 3: Gradual Rollout

```bash
# Week 1: Enable for development namespaces
helm upgrade np4ns charts/np4ns \
  --set config.nsTargetForNp="dev-*"

# Week 2: Add staging namespaces
helm upgrade np4ns charts/np4ns \
  --set config.nsTargetForNp="dev-*,staging-*"

# Week 3: Add production namespaces (one at a time)
helm upgrade np4ns charts/np4ns \
  --set config.nsTargetForNp="dev-*,staging-*,prod-api"

# Week 4: Full rollout
helm upgrade np4ns charts/np4ns \
  --set config.nsTargetForNp=""
```

#### Phase 4: Cleanup

```bash
# Remove old policies that are now managed by np4ns
# Keep complementary policies (ingress, additional egress rules, etc.)

# Example: Remove old egress-only policies
kubectl delete networkpolicy old-egress-policy -n myapp
```

---

## Development vs Production

### Scenario

Different network policy requirements for dev and prod environments.

### Setup

#### Production Cluster - Strict Policies

```bash
# Deploy with strict enforcement
helm install np4ns charts/np4ns \
  --set image.tag=custom-strict \
  --set config.nsExceptionList="kube-system,kube-public,kube-node-lease" \
  --set config.nsTargetForNp="" \
  --namespace np4ns-system
```

#### Development Cluster - Permissive Policies

```bash
# Deploy with permissive enforcement
helm install np4ns charts/np4ns \
  --set image.tag=custom-permissive \
  --set config.nsExceptionList="kube-system,kube-public,kube-node-lease" \
  --set config.nsTargetForNp="" \
  --namespace np4ns-system
```

### Custom Builds

**Strict Policy (Production):**
```go
// Deny all by default, explicit allows only
func buildCompliantNetworkPolicySpec() networkingv1.NetworkPolicySpec {
    // Very restrictive - only allow specific internal communication
    // See CUSTOMIZATION.md for examples
}
```

**Permissive Policy (Development):**
```go
// Allow most traffic, block only external
func buildCompliantNetworkPolicySpec() networkingv1.NetworkPolicySpec {
    return networkingv1.NetworkPolicySpec{
        PodSelector: metav1.LabelSelector{},
        PolicyTypes: []networkingv1.PolicyType{
            networkingv1.PolicyTypeEgress,
        },
        Egress: []networkingv1.NetworkPolicyEgressRule{
            // Allow all internal cluster communication
            {
                To: []networkingv1.NetworkPolicyPeer{
                    {
                        PodSelector: &metav1.LabelSelector{},
                    },
                },
            },
            // Block external internet (optional)
        },
    }
}
```

---

## Tips for All Scenarios

1. **Start with observation**: Deploy with permissive policies first, monitor traffic patterns
2. **Use labels**: Leverage namespace and pod labels for flexible policies
3. **Test thoroughly**: Use network policy visualizers and testing tools
4. **Monitor logs**: Watch operator logs during rollout
5. **Have rollback plan**: Keep operator easily upgradable/downgradable
6. **Document exceptions**: Maintain a list of why certain namespaces are excluded
7. **Regular audits**: Periodically review exception lists and policy effectiveness

## Additional Resources

- [Kubernetes Network Policy Documentation](https://kubernetes.io/docs/concepts/services-networking/network-policies/)
- [Network Policy Editor](https://editor.networkpolicy.io/)
- [Cilium Network Policy Guide](https://docs.cilium.io/en/stable/policy/)
- [CUSTOMIZATION.md](CUSTOMIZATION.md) - How to customize policies
- [DEPLOYMENT.md](../DEPLOYMENT.md) - Deployment scenarios
