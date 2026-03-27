# CLAUDE.md

## Project Overview

Kost is a lightweight Kubernetes cost anomaly detector. It watches cluster resource usage, learns normal spending patterns via statistical baselines, and alerts when costs deviate — with human-readable explanations powered by pluggable LLM backends.

**Repo:** github.com/yashtandon019/kost
**Domain:** kost.dev
**Language:** Go
**Framework:** Kubebuilder (controller-runtime)

## Architecture

```
Collector → Detector → Explainer → Alerter
```

1. **Collector** (`pkg/collector/`) — pulls resource metrics from metrics-server or Prometheus
2. **Detector** (`pkg/detector/`) — statistical anomaly detection (rolling average + σ deviation, Welford's algorithm)
3. **Explainer** (`pkg/explainer/`) — pluggable LLM interface that generates human-readable explanations
4. **Alerter** (`pkg/alerter/`) — sends alerts to Slack, K8s Events, webhooks, or stdout
5. **Pricing** (`pkg/pricing/`) — configurable per-resource cost estimation

## Package Structure

```
api/v1alpha1/              # KostConfig CRD types (kubebuilder-managed)
internal/controller/       # KostConfig reconciliation loop
pkg/collector/             # Metrics collection interface + implementations
pkg/detector/              # Anomaly detection with rolling baselines
pkg/explainer/             # Pluggable LLM explainer (noop, Claude, OpenAI, Ollama)
pkg/alerter/               # Alert sink interface (Slack, K8s Events, webhook)
pkg/pricing/               # Resource cost estimation
config/                    # Kubebuilder-generated K8s manifests (CRDs, RBAC, etc.)
```

## Key Interfaces

All core components are defined as Go interfaces for pluggability:

- `collector.Collector` — `Collect(ctx) ([]NamespaceUsage, error)`
- `detector.Detector` — `Detect(usages) []Anomaly` + `UpdateBaselines(usages)`
- `explainer.Explainer` — `Explain(ctx, Anomaly) (Explanation, error)`
- `alerter.Alerter` — `Alert(ctx, Explanation) error`

## Explainer Backends

| Backend | File | Status | Notes |
|---------|------|--------|-------|
| Noop | `pkg/explainer/noop.go` | ✅ Implemented | Returns raw anomaly data, no LLM |
| Claude | `pkg/explainer/claude.go` | 🔲 Stub | Anthropic Messages API |
| OpenAI | `pkg/explainer/openai.go` | 🔲 Stub | Chat Completions API |
| Ollama | `pkg/explainer/ollama.go` | 🔲 Stub | Local models, zero cost |

## Build & Test Commands

```bash
make manifests          # Regenerate CRDs/RBAC from kubebuilder markers
make generate           # Regenerate DeepCopy methods
make lint               # Run golangci-lint
make lint-fix           # Auto-fix lint issues
make test               # Run unit tests (uses envtest)
go build ./...          # Compile all packages
go vet ./...            # Static analysis
```

## Development Rules

- **Never edit auto-generated files:** `config/crd/bases/`, `config/rbac/role.yaml`, `**/zz_generated.*.go`, `PROJECT`
- **Never remove scaffold markers:** `// +kubebuilder:scaffold:*` comments
- **Always run after editing types:** `make manifests && make generate`
- **Always run before committing:** `make lint-fix && make test`
- **Use kubebuilder CLI** to scaffold new APIs/webhooks, don't create files manually

## Git & CI Conventions

- **No `Co-Authored-By` lines** in commits for this repo
- **SSH remote** uses `github-personal` host alias (maps to personal SSH key)
- **CI:** lint + unit tests run on PRs to `main` only (GitHub Actions)
- **No e2e tests in CI yet** — run locally with kind

## Local Testing Setup

```bash
# 1. Create local cluster
kind create cluster --name kost-dev

# 2. Install metrics-server
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
kubectl patch deployment metrics-server -n kube-system \
  --type='json' -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'

# 3. Install CRDs and run operator locally
make install
make run
```

## Issue Tracking

GitHub Issues are used as the backlog. Current priorities:
- 🔴 High: #2 (collector), #3 (controller loop)
- 🟡 Medium: #4 (Slack alerter), #5 (Claude explainer), #9 (log alerter), #10 (Helm chart)
- 🟢 Low: #6 (OpenAI), #7 (Ollama), #8 (K8s Events), #11 (webhook alerter), #12 (docs)

## Cost Estimation Formula

```
cost/hr = (cpu_cores × cpu_price/hr) + (memory_gb × mem_price/hr) + (gpu_count × gpu_price/hr)
```

Default prices approximate AWS on-demand (m5.xlarge / g4dn.xlarge). User-configurable via `pkg/pricing/`.

## Anomaly Detection Logic

Uses Welford's online algorithm for running mean + standard deviation. An anomaly is flagged when:

```
current_cost > baseline_avg + (sigma_threshold × std_dev)
```

Default sigma threshold: 2.0. Minimum samples before detection activates: 10.

Severity classification:
- Low: < 100% deviation
- Medium: 100-200% deviation
- High: 200-500% deviation
- Critical: > 500% deviation
