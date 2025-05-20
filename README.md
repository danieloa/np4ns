# np4ns

Network Policies For Namespaces (k8s operator in go)

// TODO(user): Add simple overview of use/purpose
ensure that for each namespace there is a network policy enforced
even in the event of deletion, this policy will be recreated

## Description
k8s oerator to ensure that for each namespace there is a network policy enforced, even in the event of np deletion, this policy will be recreated

## Getting Started

**Docs i followed to get here:**
- https://www.codereliant.io/build-kubernetes-operator-kubebuilder/
- https://www.codereliant.io/hands-on-kubernetes-operator-development-part-2/
- https://kubernetes.io/blog/2021/06/21/writing-a-controller-for-pod-labels/
- https://github.com/busser/label-operator


**Commands i run to get here:**


```bash
mkdir k8s-operator && cd k8s-operator
operator-sdk init --domain=danieloa.io --repo=github.com/danieloa/np4ns
operator-sdk create api --group="" --version=v1 --kind=Namespace --controller=true --resource=false
```

Note that specifying `--resource=true` when creating the API will auto-generate the code for the resource inside `internal/controller` folder
Also note that when creating controllers to manage kube-system objects (like namespaces,pods, ie `core` components) we need to set the `--group=""`
For this particular case, we do not need to create any CRD, hence when creating the api we will specify `--resource=false`

### Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Run on the cluster

Note: ensure kubectx is set to kind cluster:

```sh
go run cmd/main.go
```

it will iterate through all namespaces (except the ones from the nsExceptionList) and where no network policy is found, will create a compliant one.
if you create a new namespace, a compliant network policy will be enforced
if you edit an existing network policy, it will validate it and if not compliant, it will make it compliant
if you delete an existing network policy it will recreate it


### TODO Next:

- `NS_EXCEPTION_LIST` (namespaces where network policies are not enforced)
- `NS_TARGET_FOR_NP` (namespaces where network policies are enforced)

These environment variables can be set via a ConfigMap and mounted into the operator's pod at deployment runtime. Below are examples of how to configure them:

#### ConfigMap Example
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: np4ns-config
  namespace: default
data:
  NS_EXCEPTION_LIST: "kube-system,default"
  NS_TARGET_FOR_NP: "team-a,team-b"
