# fluid-ai-diagnoser

A standalone Go library for performing **deterministic, rule-based reasoning** on Fluid diagnostic data.

## Purpose

Fluid diagnostics are inherently complex—operators must correlate state across Kubernetes resources, Fluid CRDs (Datasets, Runtimes), and underlying storage systems. This library bridges the gap between raw diagnostic data and actionable insights.

Given a structured `DiagnosticContext`, the engine produces ranked hypotheses explaining potential root causes, backed by explicit evidence.

## Non-Goals

This project explicitly does **NOT**:

- **Interact with Kubernetes APIs** — No `client-go`, no live cluster access
- **Collect logs or resources** — Data gathering is the responsibility of upstream tools
- **Mutate cluster state** — Read-only analysis only
- **Provide "auto-fix" capabilities** — Suggestions are advisory
- **Include a CLI** — This is a pure library; CLI belongs in `fluidctl`

## Installation

```bash
go get github.com/mrhapile/fluid-ai-diagnoser
```

## Usage

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/mrhapile/fluid-ai-diagnoser/pkg/engine"
    "github.com/mrhapile/fluid-ai-diagnoser/pkg/types"
)

func main() {
    // Load diagnostic context (from file, bundler, or fluidctl)
    data, _ := os.ReadFile("diagnostic_context.json")
    
    var ctx types.DiagnosticContext
    json.Unmarshal(data, &ctx)

    // Run analysis
    result, err := engine.Analyze(ctx)
    if err != nil {
        panic(err)
    }

    // Output results
    output, _ := json.MarshalIndent(result, "", "  ")
    fmt.Println(string(output))
}
```

## Sample Input

```json
{
  "summary": {
    "clusterVersion": "v1.28.0",
    "namespace": "fluid-system"
  },
  "graph": {
    "pods": {
      "mydata-fuse-abc123": {
        "name": "mydata-fuse-abc123",
        "namespace": "default",
        "status": "Pending",
        "conditions": [
          {
            "type": "PodScheduled",
            "status": "False",
            "reason": "Unschedulable",
            "message": "0/1 nodes are available: 1 node(s) had taints that the pod didn't tolerate."
          }
        ],
        "labels": {"role": "fuse"}
      }
    },
    "runtimes": {
      "mydata": {
        "name": "mydata",
        "namespace": "default",
        "type": "Alluxio",
        "masterReplicas": 1,
        "workerReplicas": 2,
        "masterReady": 1,
        "workerReady": 0,
        "phase": "NotReady"
      }
    }
  },
  "events": [
    {
      "reason": "FailedScheduling",
      "message": "0/1 nodes are available: 1 node(s) had taints that the pod didn't tolerate.",
      "type": "Warning",
      "involvedObject": {"kind": "Pod", "name": "mydata-fuse-abc123"}
    }
  ],
  "findings": [],
  "logs": {},
  "metadata": {
    "creationTimestamp": "2026-02-08T04:35:00Z",
    "collectorVersion": "v0.1.0"
  }
}
```

## Sample Output

```json
{
  "hypotheses": [
    {
      "rank": 1,
      "confidence": 0.8,
      "component": "Fuse",
      "issue": "Fuse pod cannot be scheduled due to node taints or missing tolerations",
      "evidence": [
        "Pod default/mydata-fuse-abc123: PodScheduled=False, reason=Unschedulable",
        "Event: FailedScheduling - 0/1 nodes are available: 1 node(s) had taints that the pod didn't tolerate."
      ],
      "suggestion": "Check node taints and ensure Fuse pods have appropriate tolerations. Verify node selectors match available nodes."
    },
    {
      "rank": 2,
      "confidence": 0.6,
      "component": "Runtime",
      "issue": "Runtime is only partially ready, indicating dependency or configuration failure",
      "evidence": [
        "Runtime default/mydata: Worker 0/2 ready"
      ],
      "suggestion": "Check runtime pod logs for errors. Verify storage backend connectivity and credentials."
    }
  ],
  "generatedAt": "2026-02-08T04:36:00Z",
  "engine": "rule-based"
}
```

## Rules

The engine includes the following deterministic rules:

| Rule ID | Component | Detects |
|---------|-----------|---------|
| `fuse-unschedulable` | Fuse | Fuse pods pending due to node taints/tolerations |
| `worker-pending-memory` | Worker | Worker pods pending due to insufficient memory |
| `runtime-partially-ready` | Runtime | Runtime not fully ready (workers/masters missing) |
| `pvc-unbound` | Storage | PVCs not bound due to provisioning issues |
| `dataset-not-bound` | Dataset | Datasets not bound due to missing Runtime |

## Confidence Scoring

Confidence is assigned based on evidence strength (heuristic, not probabilistic):

| Evidence Type | Confidence |
|---------------|------------|
| Event + Pod Status | 0.8 |
| Pod Status Only | 0.6 |
| Log Match | 0.55 |
| Condition Only | 0.5 |
| Event Only | 0.5 |
| Weak Signal | 0.3 |

## Integration

This library is designed to be used **after** diagnostic data collection:

- **`fluid-diagnose-bundler`** — Produces the `DiagnosticContext` as a portable archive
- **`fluidctl diagnose --output json`** — Outputs the context directly

```
┌─────────────────────┐     ┌─────────────────────────┐     ┌──────────────┐
│ fluidctl diagnose   │ ──▶ │ fluid-diagnose-bundler  │ ──▶ │ This Library │
│ (data collection)   │     │ (archiving)             │     │ (reasoning)  │
└─────────────────────┘     └─────────────────────────┘     └──────────────┘
```

## License

Apache 2.0
