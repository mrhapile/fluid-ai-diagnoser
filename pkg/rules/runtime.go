package rules

import (
	"fmt"

	"github.com/mrhapile/fluid-ai-diagnoser/pkg/types"
)

// RuntimePartiallyReadyRule detects runtimes that are only partially ready due to dependency failures.
type RuntimePartiallyReadyRule struct{}

func (r *RuntimePartiallyReadyRule) ID() string {
	return "runtime-partially-ready"
}

func (r *RuntimePartiallyReadyRule) Match(ctx types.DiagnosticContext) bool {
	for _, runtime := range ctx.Graph.Runtimes {
		// Check if master or worker replicas are not fully ready
		if runtime.MasterReplicas > 0 && runtime.MasterReady < runtime.MasterReplicas {
			return true
		}
		if runtime.WorkerReplicas > 0 && runtime.WorkerReady < runtime.WorkerReplicas {
			return true
		}
		// Check for non-ready conditions
		for _, cond := range runtime.Conditions {
			if cond.Status == "False" && cond.Type == "Ready" {
				return true
			}
		}
	}
	return false
}

func (r *RuntimePartiallyReadyRule) Hypothesis(ctx types.DiagnosticContext) types.Hypothesis {
	var evidence []string
	confidence := types.ConfidenceConditionOnly

	for name, runtime := range ctx.Graph.Runtimes {
		if runtime.MasterReplicas > 0 && runtime.MasterReady < runtime.MasterReplicas {
			evidence = append(evidence, fmt.Sprintf("Runtime %s/%s: Master %d/%d ready",
				runtime.Namespace, name, runtime.MasterReady, runtime.MasterReplicas))
			confidence = types.ConfidencePodStatusOnly
		}
		if runtime.WorkerReplicas > 0 && runtime.WorkerReady < runtime.WorkerReplicas {
			evidence = append(evidence, fmt.Sprintf("Runtime %s/%s: Worker %d/%d ready",
				runtime.Namespace, name, runtime.WorkerReady, runtime.WorkerReplicas))
			confidence = types.ConfidencePodStatusOnly
		}
		for _, cond := range runtime.Conditions {
			if cond.Status == "False" {
				evidence = append(evidence, fmt.Sprintf("Runtime %s/%s: Condition %s=%s, reason=%s",
					runtime.Namespace, name, cond.Type, cond.Status, cond.Reason))
				if cond.Reason != "" {
					confidence = types.ConfidenceEventAndStatus
				}
			}
		}
	}

	return types.Hypothesis{
		Confidence: confidence,
		Component:  "Runtime",
		Issue:      "Runtime is only partially ready, indicating dependency or configuration failure",
		Evidence:   evidence,
		Suggestion: "Check runtime pod logs for errors. Verify storage backend connectivity and credentials. Ensure all required dependencies are available.",
	}
}
