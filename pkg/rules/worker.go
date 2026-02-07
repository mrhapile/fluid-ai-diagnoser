package rules

import (
	"fmt"
	"strings"

	"github.com/mrhapile/fluid-ai-diagnoser/pkg/types"
)

// WorkerPendingMemoryRule detects worker pods pending due to insufficient memory.
type WorkerPendingMemoryRule struct{}

func (r *WorkerPendingMemoryRule) ID() string {
	return "worker-pending-memory"
}

func (r *WorkerPendingMemoryRule) Match(ctx types.DiagnosticContext) bool {
	// Check for worker pods in pending state
	for _, pod := range ctx.Graph.Pods {
		if !isWorkerPod(pod) {
			continue
		}
		if pod.Status == "Pending" {
			// Check for memory-related scheduling failures
			for _, cond := range pod.Conditions {
				if cond.Type == "PodScheduled" && cond.Status == "False" {
					if strings.Contains(strings.ToLower(cond.Message), "memory") ||
						strings.Contains(strings.ToLower(cond.Message), "insufficient") {
						return true
					}
				}
			}
		}
	}

	// Check events for memory issues
	for _, event := range ctx.Events {
		if event.Type == "Warning" && event.Reason == "FailedScheduling" {
			if isWorkerRelatedEvent(event) &&
				strings.Contains(strings.ToLower(event.Message), "memory") {
				return true
			}
		}
	}

	return false
}

func (r *WorkerPendingMemoryRule) Hypothesis(ctx types.DiagnosticContext) types.Hypothesis {
	var evidence []string
	confidence := types.ConfidenceConditionOnly

	// Gather evidence from pods
	for name, pod := range ctx.Graph.Pods {
		if !isWorkerPod(pod) {
			continue
		}
		if pod.Status == "Pending" {
			for _, cond := range pod.Conditions {
				if cond.Type == "PodScheduled" && cond.Status == "False" {
					if strings.Contains(strings.ToLower(cond.Message), "memory") ||
						strings.Contains(strings.ToLower(cond.Message), "insufficient") {
						evidence = append(evidence, fmt.Sprintf("Pod %s/%s: %s",
							pod.Namespace, name, cond.Message))
						confidence = types.ConfidenceEventAndStatus
					}
				}
			}
		}
	}

	// Gather evidence from events
	for _, event := range ctx.Events {
		if event.Type == "Warning" && event.Reason == "FailedScheduling" {
			if isWorkerRelatedEvent(event) &&
				strings.Contains(strings.ToLower(event.Message), "memory") {
				evidence = append(evidence, fmt.Sprintf("Event: %s - %s", event.Reason, event.Message))
				confidence = types.ConfidenceEventAndStatus
			}
		}
	}

	return types.Hypothesis{
		Confidence: confidence,
		Component:  "Worker",
		Issue:      "Worker pod cannot be scheduled due to insufficient memory",
		Evidence:   evidence,
		Suggestion: "Reduce worker memory requests, add nodes with more memory, or scale down other workloads to free resources.",
	}
}

func isWorkerPod(pod types.PodInfo) bool {
	// Check labels for worker identification
	if role, ok := pod.Labels["role"]; ok && role == "worker" {
		return true
	}
	if _, ok := pod.Labels["fluid.io/worker"]; ok {
		return true
	}
	// Check owner references
	for _, owner := range pod.OwnerReferences {
		if strings.Contains(strings.ToLower(owner.Name), "worker") {
			return true
		}
	}
	return strings.Contains(strings.ToLower(pod.Name), "worker")
}

func isWorkerRelatedEvent(event types.Event) bool {
	return strings.Contains(strings.ToLower(event.InvolvedObject.Name), "worker")
}
