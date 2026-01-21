# Security Policy

## Supported Versions

We release patches for security vulnerabilities in the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.0.x   | :white_check_mark: |

## Reporting a Vulnerability

We take the security of np4ns seriously. If you believe you have found a security vulnerability, please report it to us as described below.

### Please Do Not

- Open a public GitHub issue for security vulnerabilities
- Discuss the vulnerability in public forums, chat, or social media

### Please Do

**Report security vulnerabilities privately** using one of these methods:

1. **GitHub Security Advisories (Preferred)**
   - Go to https://github.com/danieloa/np4ns/security/advisories/new
   - Click "Report a vulnerability"
   - Fill out the form with details about the vulnerability

2. **Email**
   - Send details to the repository maintainers through GitHub
   - Include "SECURITY" in the subject line

### What to Include

Please include the following information in your report:

- Type of vulnerability (e.g., privilege escalation, information disclosure)
- Full paths of source file(s) related to the vulnerability
- Location of the affected source code (tag/branch/commit or direct URL)
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the vulnerability, including how an attacker might exploit it
- Any potential mitigations you've identified

### What to Expect

- **Acknowledgment**: We will acknowledge receipt of your vulnerability report within 48 hours
- **Updates**: We will send you regular updates about our progress
- **Timeline**: We aim to resolve critical vulnerabilities within 30 days
- **Credit**: We will credit you for the discovery (unless you prefer to remain anonymous)

### Disclosure Policy

- We will work with you to understand and validate the vulnerability
- We will develop and test a fix
- We will prepare security advisories and updates
- We will coordinate the disclosure timeline with you
- After the fix is released, we will publish a security advisory

**Coordinated Disclosure**: We ask that you do not publicly disclose the vulnerability until we have had a chance to address it and release a fix.

## Security Best Practices

When deploying np4ns, we recommend:

### RBAC Configuration

- The operator requires ClusterRole permissions to manage namespaces and network policies
- Review and audit RBAC permissions regularly
- Use the principle of least privilege

```yaml
# The operator requires these permissions:
- namespaces: get, list, watch, update, patch
- networkpolicies: get, list, watch, create, update, patch, delete
```

### Network Policies

- Review the default network policy specification before deployment
- Ensure the enforced policy meets your security requirements
- Customize the policy template if needed (see `buildCompliantNetworkPolicySpec()`)
- Test policies in a non-production environment first

### Container Security

- Always use specific image tags (not `latest`)
- Verify image signatures when available
- Use private registries with image scanning enabled
- Keep the operator updated to the latest version

```bash
# Use specific versions
helm install np4ns charts/np4ns --set image.tag=v0.0.5-abc1234
```

### Resource Limits

- Set appropriate resource limits to prevent resource exhaustion
- Monitor operator resource usage

```yaml
resources:
  limits:
    cpu: 500m
    memory: 128Mi
  requests:
    cpu: 10m
    memory: 64Mi
```

### Monitoring and Auditing

- Enable Kubernetes audit logging
- Monitor operator logs for suspicious activity
- Set up alerts for unexpected behavior
- Regularly review network policy changes

```bash
# Monitor operator logs
kubectl logs -n np4ns-system deployment/np4ns-controller-manager -f

# Audit network policy changes
kubectl get events --all-namespaces --field-selector involvedObject.kind=NetworkPolicy
```

### Configuration Security

- Protect the ConfigMap with appropriate RBAC
- Validate namespace exception and target lists
- Use version control for configuration changes
- Document the rationale for exceptions

### Pod Security

The operator runs with secure defaults:
- Non-root user (UID 65532)
- No privilege escalation
- Dropped capabilities
- Read-only root filesystem (where possible)
- Security context constraints

### Supply Chain Security

- Verify image provenance using attestations
- Review the Dockerfile and build process
- Use dependabot for dependency updates
- Regularly update Go modules

## Known Security Considerations

### Owner References

Network policies created by the operator have owner references to their namespaces. This means:
- Policies are automatically deleted when namespaces are deleted
- This is intended behavior for cleanup
- Policies cannot be adopted by other controllers

### Namespace Annotations

The operator adds annotations to namespaces to track enforcement:
- Annotations include timestamps
- Annotations are used for auditing
- Ensure RBAC prevents unauthorized annotation modifications

### Leader Election

When multiple replicas are configured:
- Only one instance actively reconciles at a time
- Leader election uses leases in the operator's namespace
- This prevents conflicting updates

## Security Updates

We will publish security updates through:
- GitHub Security Advisories
- Release notes with `[SECURITY]` prefix
- CHANGELOG.md with security highlights

To stay informed:
- Watch the repository for security advisories
- Subscribe to release notifications
- Follow semantic versioning for security patches (0.0.x)

## Vulnerability Disclosure Process

1. **Report received** - Acknowledged within 48 hours
2. **Investigation** - Team assesses severity and impact
3. **Fix development** - Patch created and tested
4. **Security advisory** - Draft advisory prepared
5. **Release** - Fix released with security update
6. **Public disclosure** - Advisory published after fix is available

## Attribution

We appreciate the security research community and will credit researchers who report vulnerabilities responsibly (unless they prefer anonymity).

## Questions?

If you have questions about this security policy, please open a discussion in GitHub Discussions or contact the maintainers.

---

Thank you for helping keep np4ns and its users safe!
