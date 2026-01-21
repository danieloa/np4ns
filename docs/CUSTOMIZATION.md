# Customizing Network Policies

This guide explains how to customize the network policy that np4ns enforces across your namespaces.

## Current Behavior

By default, np4ns enforces a network policy with the following specification:

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

## Customization Options

There are several ways to customize the enforced network policy:

### Option 1: Modify the Source Code (Recommended)

The network policy specification is defined in the `buildCompliantNetworkPolicySpec()` function in `internal/controller/namespace_controller.go`.

**Steps:**

1. **Fork the repository**
   ```bash
   # Fork on GitHub, then clone your fork
   git clone https://github.com/YOUR-USERNAME/np4ns.git
   cd np4ns
   ```

2. **Modify the network policy spec**

   Edit `internal/controller/namespace_controller.go` around line 206:

   ```go
   func buildCompliantNetworkPolicySpec() networkingv1.NetworkPolicySpec {
       protocolTCP := corev1.ProtocolTCP
       port := intstr.FromInt(80)

       return networkingv1.NetworkPolicySpec{
           PodSelector: metav1.LabelSelector{}, // Selects all pods
           PolicyTypes: []networkingv1.PolicyType{
               networkingv1.PolicyTypeEgress,
           },
           Egress: []networkingv1.NetworkPolicyEgressRule{
               {
                   To: []networkingv1.NetworkPolicyPeer{
                       {
                           NamespaceSelector: &metav1.LabelSelector{
                               MatchLabels: map[string]string{
                                   "name": "target-namespace",
                               },
                           },
                           PodSelector: &metav1.LabelSelector{
                               MatchLabels: map[string]string{
                                   "app": "myapp",
                               },
                           },
                       },
                   },
                   Ports: []networkingv1.NetworkPolicyPort{
                       {
                           Protocol: &protocolTCP,
                           Port:     &port,
                       },
                   },
               },
           },
       }
   }
   ```

3. **Build and deploy your custom version**
   ```bash
   # Build custom image
   make docker-build IMG=ghcr.io/YOUR-USERNAME/np4ns:custom

   # Push to your registry
   make docker-push IMG=ghcr.io/YOUR-USERNAME/np4ns:custom

   # Deploy
   make deploy IMG=ghcr.io/YOUR-USERNAME/np4ns:custom
   ```

### Option 2: Use Multiple Instances with Different Configs

Deploy multiple instances of the operator, each managing different namespaces with different policies:

```bash
# Instance 1: Manages production namespaces with strict policy
helm install np4ns-prod charts/np4ns \
  --set image.tag=custom-strict \
  --set config.nsTargetForNp="prod-*"

# Instance 2: Manages dev namespaces with permissive policy
helm install np4ns-dev charts/np4ns \
  --set image.tag=custom-permissive \
  --set config.nsTargetForNp="dev-*"
```

### Option 3: Extend with Additional Policies

The operator creates a policy named `enforced-network-policy`. You can create additional policies manually that complement the enforced one:

```yaml
# Additional policy for ingress rules
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ingress-policy
  namespace: myapp
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: frontend
```

The operator will not touch policies with different names.

## Common Customization Examples

### Example 1: Allow All DNS Traffic

```go
func buildCompliantNetworkPolicySpec() networkingv1.NetworkPolicySpec {
    protocolTCP := corev1.ProtocolTCP
    protocolUDP := corev1.ProtocolUDP
    dnsPort := intstr.FromInt(53)

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
                    },
                },
                Ports: []networkingv1.NetworkPolicyPort{
                    {Protocol: &protocolUDP, Port: &dnsPort},
                    {Protocol: &protocolTCP, Port: &dnsPort},
                },
            },
        },
    }
}
```

### Example 2: Allow Egress to Specific CIDR

```go
func buildCompliantNetworkPolicySpec() networkingv1.NetworkPolicySpec {
    return networkingv1.NetworkPolicySpec{
        PodSelector: metav1.LabelSelector{},
        PolicyTypes: []networkingv1.PolicyType{
            networkingv1.PolicyTypeEgress,
        },
        Egress: []networkingv1.NetworkPolicyEgressRule{
            // Allow egress to specific external IP range
            {
                To: []networkingv1.NetworkPolicyPeer{
                    {
                        IPBlock: &networkingv1.IPBlock{
                            CIDR: "10.0.0.0/8",
                            Except: []string{
                                "10.0.1.0/24", // Except this subnet
                            },
                        },
                    },
                },
            },
        },
    }
}
```

### Example 3: Combined Ingress and Egress Policy

```go
func buildCompliantNetworkPolicySpec() networkingv1.NetworkPolicySpec {
    protocolTCP := corev1.ProtocolTCP
    httpPort := intstr.FromInt(80)
    httpsPort := intstr.FromInt(443)

    return networkingv1.NetworkPolicySpec{
        PodSelector: metav1.LabelSelector{},
        PolicyTypes: []networkingv1.PolicyType{
            networkingv1.PolicyTypeIngress,
            networkingv1.PolicyTypeEgress,
        },
        Ingress: []networkingv1.NetworkPolicyIngressRule{
            {
                From: []networkingv1.NetworkPolicyPeer{
                    {
                        NamespaceSelector: &metav1.LabelSelector{
                            MatchLabels: map[string]string{
                                "environment": "production",
                            },
                        },
                    },
                },
                Ports: []networkingv1.NetworkPolicyPort{
                    {Protocol: &protocolTCP, Port: &httpPort},
                },
            },
        },
        Egress: []networkingv1.NetworkPolicyEgressRule{
            {
                To: []networkingv1.NetworkPolicyPeer{
                    {
                        NamespaceSelector: &metav1.LabelSelector{
                            MatchLabels: map[string]string{
                                "environment": "production",
                            },
                        },
                    },
                },
                Ports: []networkingv1.NetworkPolicyPort{
                    {Protocol: &protocolTCP, Port: &httpsPort},
                },
            },
        },
    }
}
```

### Example 4: Deny All (Most Restrictive)

```go
func buildCompliantNetworkPolicySpec() networkingv1.NetworkPolicySpec {
    return networkingv1.NetworkPolicySpec{
        PodSelector: metav1.LabelSelector{}, // Applies to all pods
        PolicyTypes: []networkingv1.PolicyType{
            networkingv1.PolicyTypeIngress,
            networkingv1.PolicyTypeEgress,
        },
        // No ingress or egress rules = deny all
    }
}
```

### Example 5: Allow All (Most Permissive)

```go
func buildCompliantNetworkPolicySpec() networkingv1.NetworkPolicySpec {
    return networkingv1.NetworkPolicySpec{
        PodSelector: metav1.LabelSelector{},
        PolicyTypes: []networkingv1.PolicyType{
            networkingv1.PolicyTypeIngress,
            networkingv1.PolicyTypeEgress,
        },
        Ingress: []networkingv1.NetworkPolicyIngressRule{
            {}, // Empty rule allows all ingress
        },
        Egress: []networkingv1.NetworkPolicyEgressRule{
            {}, // Empty rule allows all egress
        },
    }
}
```

## Testing Your Custom Policy

After modifying the policy:

1. **Build and deploy** your custom operator
2. **Create a test namespace**
   ```bash
   kubectl create namespace test-custom-policy
   ```

3. **Verify the policy is created**
   ```bash
   kubectl get networkpolicy -n test-custom-policy
   kubectl describe networkpolicy enforced-network-policy -n test-custom-policy
   ```

4. **Test connectivity**
   ```bash
   # Deploy a test pod
   kubectl run test-pod -n test-custom-policy --image=busybox --command -- sleep 3600

   # Test DNS
   kubectl exec -n test-custom-policy test-pod -- nslookup kubernetes.default

   # Test external connectivity
   kubectl exec -n test-custom-policy test-pod -- wget -O- https://example.com
   ```

5. **Check operator logs**
   ```bash
   kubectl logs -n np4ns-system deployment/np4ns-controller-manager -f
   ```

## Best Practices

1. **Start Restrictive**: Begin with a deny-all policy and incrementally add allowed traffic
2. **Test Thoroughly**: Test in non-production environments first
3. **Document Changes**: Keep a record of why you made specific customizations
4. **Version Control**: Store your custom operator code in version control
5. **Monitor Impact**: Watch for application errors after policy changes
6. **Use Labels**: Leverage namespace and pod labels for flexible policies
7. **Consider Multiple Policies**: Use the operator's enforced policy for base security, and add additional policies for specific needs

## Troubleshooting

### Policy Not Being Enforced

Check operator logs:
```bash
kubectl logs -n np4ns-system deployment/np4ns-controller-manager | grep "NetworkPolicy"
```

### Application Connectivity Issues

1. Verify the policy spec:
   ```bash
   kubectl get networkpolicy enforced-network-policy -n <namespace> -o yaml
   ```

2. Check if the application's required traffic is allowed
3. Use network policy visualizers (Cilium Editor, NetworkPolicy.io)
4. Test with a permissive policy first, then restrict

### Policy Changes Not Taking Effect

The operator checks for compliance on every reconciliation. To force an immediate check:
```bash
kubectl annotate namespace <namespace> reconcile=true --overwrite
```

## Future Enhancements

We're considering these features for future releases:

- ConfigMap-based policy templates (without code changes)
- CRD for policy definitions
- Per-namespace policy customization
- Policy templates library

See [open issues](https://github.com/danieloa/np4ns/issues) for discussion and progress.

## Contributing

If you develop a useful policy template, please consider contributing it back:
1. Add your example to this documentation
2. Submit a pull request
3. Include your use case and testing results

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.
