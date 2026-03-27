# Kost

Lightweight Kubernetes cost anomaly detector. Detects spending deviations and explains them in plain English.

## What Is This?

Kost is a Kubernetes operator that watches your cluster's resource usage, learns normal spending patterns, and alerts you when something looks off — with a human-readable explanation of what happened and what to do about it.

**Instead of this:**
> CPU usage in namespace `staging` is 287% above baseline.

**You get this:**
> 🚨 **Cost Anomaly — namespace: staging**
>
> Current hourly burn rate: $12.40/hr (normal: $3.20/hr, +287%)
>
> **What happened:** 6 hours ago, deployment `ml-inference` was scaled from 2 → 12 replicas. Each replica requests 1 GPU + 4 CPU + 16Gi memory, accounting for $9.10/hr of the increase.
>
> **Suggested actions:**
> 1. Scale `ml-inference` down to 2-3 replicas if this was a test
> 2. Add a ResourceQuota to `staging` (currently unlimited)
> 3. Consider spot/preemptible nodes for this workload

## Why Not Kubecost?

Kubecost is great but heavy — it requires Prometheus, Grafana, and its own stack. Kost is a single binary that works with metrics-server (already in most clusters) and focuses on **alerting + explanation**, not dashboards.

| | Kubecost | Kost |
|---|---|---|
| Footprint | Heavy (Prometheus + Grafana + own stack) | Single binary/operator |
| Focus | Dashboards + reporting | Alerting + explanation |
| Intelligence | Static thresholds | Statistical anomaly detection + LLM reasoning |
| Output | Charts you interpret | Actionable Slack alerts |

## Architecture

```
┌─────────────────────────────────────────────────┐
│                 Kost Operator                    │
│                                                  │
│  ┌───────────┐  ┌───────────┐  ┌─────────────┐  │
│  │ Collector  │→ │ Detector  │→ │  Explainer  │  │
│  └───────────┘  └───────────┘  └─────────────┘  │
│       │               │              │           │
│  metrics-server    statistical     LLM API      │
│  or Prometheus     baselines     (pluggable)    │
│                                                  │
│  ┌──────────────────────────────────────────┐    │
│  │            Alert Sink                     │    │
│  │    Slack / Webhook / K8s Events           │    │
│  └──────────────────────────────────────────┘    │
└─────────────────────────────────────────────────┘
```

### Collector
Pulls resource usage data from metrics-server API or Prometheus (if available). Tracks CPU, memory, pod count, and GPU usage per namespace and deployment. No heavy dependencies required.

### Detector
Statistical anomaly detection — no ML framework needed. Uses rolling averages + standard deviation with day-of-week awareness. If current cost exceeds baseline + 2σ, it flags it.

### Explainer
Feeds the anomaly data + recent Kubernetes events (deployments, scaling, config changes) to a pluggable LLM backend (Claude, GPT, or local models) to produce human-readable explanations and suggestions.

### Alert Sink
Sends alerts to Slack, webhooks, PagerDuty, or emits Kubernetes Events. Supports acknowledge/suppress workflows.

## Data Collected

Per namespace, every 5 minutes:
- Total CPU requests vs actual usage
- Total memory requests vs actual usage
- Pod and container count
- GPU allocations (if applicable)
- Recent events (deployments, scaling events, OOM kills)

Cost is estimated from resource usage:
```
cost ≈ (cpu_cores × cpu_price) + (memory_gb × mem_price) + (gpu × gpu_price)
```
Resource prices are user-configurable.

## Local Demo

With a Kind cluster and `metrics-server` installed:

```bash
make install
kubectl apply -f config/samples/kost_v1alpha1_kostconfig.yaml
make run
```

Then inspect the resource:

```bash
kubectl describe kostconfig kostconfig-sample
```

Example output:

```text
Name:         kostconfig-sample
Namespace:    default
Labels:       app.kubernetes.io/managed-by=kustomize
              app.kubernetes.io/name=kost
Annotations:  <none>
API Version:  kost.kost.dev/v1alpha1
Kind:         KostConfig
Metadata:
  Creation Timestamp:  2026-03-27T18:55:30Z
  Generation:          1
  Resource Version:    1460
  UID:                 2b24ac9a-11e4-45ce-b56d-3ec9cc80c0c3
Spec:
  Explainer:
    Provider:        noop
  Min Samples:       10
  Polling Interval:  5m
  Pricing:
    Cpu Per Core Hour:   0.034
    Gpu Per Hour:        0.526
    Memory Per GB Hour:  0.0043
  Sigma Threshold:       2
Status:
  Active Anomalies:  0
  Conditions:
    Last Transition Time:  2026-03-27T18:57:15Z
    Message:               Checked 5 namespaces, found 0 anomalies
    Reason:                ReconcileSucceeded
    Status:                True
    Type:                  Available
  Last Check Time:         2026-03-27T18:57:15Z
Events:                    <none>
```

This shows the current happy-path behavior:
- the controller is running
- metrics collection succeeded
- anomaly detection ran
- no anomalies were found yet

For a faster demo, lower `minSamples` and `pollingInterval` in `config/samples/kost_v1alpha1_kostconfig.yaml`.

## Roadmap

- [ ] Operator scaffold with kubebuilder
- [ ] Metrics collector (metrics-server integration)
- [ ] Statistical anomaly detector (rolling average + σ deviation)
- [ ] LLM explainer with pluggable backends
- [ ] Slack alert integration
- [ ] Cost estimation with configurable pricing
- [ ] Acknowledge/suppress workflow
- [ ] Multi-cluster support
- [ ] Historical trend reporting

## Tech Stack

- **Language:** Go
- **Operator framework:** Kubebuilder
- **Anomaly detection:** Statistical (no ML framework needed)
- **LLM integration:** Pluggable — Claude, OpenAI, or local models
- **Storage:** In-memory with periodic ConfigMap snapshots
- **Alerting:** Slack webhooks, K8s Events, extensible sinks

## License

MIT
