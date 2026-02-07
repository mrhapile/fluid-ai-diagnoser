package rules

import (
	"fmt"
	"strings"

	"github.com/mrhapile/fluid-ai-diagnoser/pkg/types"
)

// FuseUnschedulableRule detects Fuse pods that are unschedulable due to node taints/tolerations.
type FuseUnschedulableRule struct{}

func (r *FuseUnschedulableRule) ID() string {
	return "fuse-unschedulable"
}

func (r *FuseUnschedulableRule) Match(ctx types.DiagnosticContext) bool {
	// Check for fuse pods in pending state with scheduling issues
	for _, pod := range ctx.Graph.Pods {
		if !isFusePod(pod) {
			continue
		}
		if pod.Status == "Pending" {
			for _, cond := range pod.Conditions {
				if cond.Type == "PodScheduled" && cond.Status == "False" {
					return true
				}
			}
		}
	}

	// Check for related events
	for _, event := range ctx.Events {
		if event.Type == "Warning" && strings.Contains(event.Reason, "FailedScheduling") {
			if isFuseRelatedEvent(event) {
				return true
			}
		}
	}

	return false
}

func (r *FuseUnschedulableRule) Hypothesis(ctx types.DiagnosticContext) types.Hypothesis {
	var evidence []string
	confidence := types.ConfidenceConditionOnly

	// Gather evidence from pods
	for name, pod := range ctx.Graph.Pods {
		if !isFusePod(pod) {
			continue
		}
		if pod.Status == "Pending" {
			for _, cond := range pod.Conditions {
				if cond.Type == "PodScheduled" && cond.Status == "False" {
					evidence = append(evidence, fmt.Sprintf("Pod %s/%s: PodScheduled=False, reason=%s",
						pod.Namespace, name, cond.Reason))
					if cond.Reason != "" {
						confidence = types.ConfidenceEventAndStatus
					}
				}
			}
		}
	}

	// Gather evidence from events
	for _, event := range ctx.Events {
		if event.Type == "Warning" && strings.Contains(event.Reason, "FailedScheduling") {
			if isFuseRelatedEvent(event) {
				evidence = append(evidence, fmt.Sprintf("Event: %s - %s", event.Reason, event.Message))
				confidence = types.ConfidenceEventAndStatus
			}
		}
	}

	return types.Hypothesis{
		Confidence: confidence,
		Component:  "Fuse",
		Issue:      "Fuse pod cannot be scheduled due to node taints or missing tolerations",
		Evidence:   evidence,
		Suggestion: "Check node taints and ensure Fuse pods have appropriate tolerations. Verify node selectors match available nodes.",
	}
}

func isFusePod(pod types.PodInfo) bool {
	// Check labels for fuse identification
	if role, ok := pod.Labels["role"]; ok && role == "fuse" {
		return true
	}
	if _, ok := pod.Labels["fluid.io/fuse"]; ok {
		return true
	}
	// Check owner references
	for _, owner := range pod.OwnerReferences {
		if strings.Contains(strings.ToLower(owner.Name), "fuse") {
			return true
		}
	}
	return strings.Contains(strings.ToLower(pod.Name), "fuse")
}

func isFuseRelatedEvent(event types.Event) bool {
	return strings.Contains(strings.ToLower(event.InvolvedObject.Name), "fuse")
}
