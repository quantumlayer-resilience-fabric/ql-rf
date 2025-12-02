# ADR-002: Agentless by Default

## Status
Accepted

## Context
Fleet inventory and drift detection require visibility into thousands of compute instances across multiple clouds and data centers. Two primary approaches exist:

1. **Agent-based**: Deploy agents on each instance to report status
2. **Agentless**: Query cloud APIs and hypervisor management planes directly

Agent-based approaches provide richer data but introduce:
- Deployment complexity (agent rollout, updates, failures)
- Security concerns (privileged processes, credential management)
- Operational overhead (monitoring agent health)
- Platform support limitations (serverless, containers)

## Decision
QL-RF adopts an **agentless-by-default** architecture:

1. **Cloud platforms**: Use native APIs (AWS EC2, Azure Resource Manager, GCP Compute)
2. **VMware vSphere**: Use vCenter APIs via govmomi
3. **Kubernetes**: Use Kubernetes API for node/pod metadata

For enhanced data (runtime configuration, package versions), we support **optional agent integration** via:
- AWS SSM Run Command
- Azure Run Command
- Ansible/SSH for on-prem

```
┌─────────────────────────────────────────────────┐
│                  QL-RF Connectors               │
├──────────┬──────────┬──────────┬───────────────┤
│   AWS    │  Azure   │   GCP    │   vSphere     │
│  (APIs)  │  (APIs)  │  (APIs)  │   (vCenter)   │
└────┬─────┴────┬─────┴────┬─────┴───────┬───────┘
     │          │          │             │
     ▼          ▼          ▼             ▼
┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐
│ EC2 API │ │ ARM API │ │ GCE API │ │ vCenter │
└─────────┘ └─────────┘ └─────────┘ └─────────┘
```

## Consequences

### Positive
- Zero footprint on target systems
- No agent deployment or maintenance
- Works with existing cloud credentials (STS, MSI, Workload Identity)
- Consistent approach across platforms
- Faster time-to-value

### Negative
- Limited visibility (no runtime process info, package versions)
- Dependent on API quotas and rate limits
- Cannot detect local configuration drift

### Mitigations
- Implement robust rate limiting and backoff
- Cache inventory data with configurable TTL
- Provide optional SSM/Ansible integration for deep inspection
- Use image tagging conventions to infer version information
