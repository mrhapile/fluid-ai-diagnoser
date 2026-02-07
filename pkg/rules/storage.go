package rules

import (
	"fmt"
	"strings"

	"github.com/mrhapile/fluid-ai-diagnoser/pkg/types"
)

// PVCUnboundRule detects PVCs that are not bound due to storage provisioning issues.
type PVCUnboundRule struct{}

func (r *PVCUnboundRule) ID() string {
	return "pvc-unbound"
}

func (r *PVCUnboundRule) Match(ctx types.DiagnosticContext) bool {
	for _, pvc := range ctx.Graph.PVCs {
		if pvc.Status == "Pending" || pvc.Status == "Lost" {
			return true
		}
	}

	// Check for provisioning failure events
	for _, event := range ctx.Events {
		if event.Type == "Warning" && event.InvolvedObject.Kind == "PersistentVolumeClaim" {
			if strings.Contains(event.Reason, "ProvisioningFailed") ||
				strings.Contains(event.Reason, "FailedBinding") {
				return true
			}
		}
	}

	return false
}

func (r *PVCUnboundRule) Hypothesis(ctx types.DiagnosticContext) types.Hypothesis {
	var evidence []string
	confidence := types.ConfidenceConditionOnly

	for name, pvc := range ctx.Graph.PVCs {
		if pvc.Status == "Pending" {
			evidence = append(evidence, fmt.Sprintf("PVC %s/%s: Status=Pending",
				pvc.Namespace, name))
			confidence = types.ConfidencePodStatusOnly
		}
		if pvc.Status == "Lost" {
			evidence = append(evidence, fmt.Sprintf("PVC %s/%s: Status=Lost",
				pvc.Namespace, name))
			confidence = types.ConfidenceEventAndStatus
		}
	}

	// Gather evidence from events
	for _, event := range ctx.Events {
		if event.Type == "Warning" && event.InvolvedObject.Kind == "PersistentVolumeClaim" {
			if strings.Contains(event.Reason, "ProvisioningFailed") ||
				strings.Contains(event.Reason, "FailedBinding") {
				evidence = append(evidence, fmt.Sprintf("Event on PVC %s: %s - %s",
					event.InvolvedObject.Name, event.Reason, event.Message))
				confidence = types.ConfidenceEventAndStatus
			}
		}
	}

	return types.Hypothesis{
		Confidence: confidence,
		Component:  "Storage",
		Issue:      "PVC is not bound due to storage provisioning failure",
		Evidence:   evidence,
		Suggestion: "Check storage class configuration and provisioner status. Verify storage backend has available capacity.",
	}
}

// DatasetNotBoundRule detects Datasets that are not bound due to missing Runtime.
type DatasetNotBoundRule struct{}

func (r *DatasetNotBoundRule) ID() string {
	return "dataset-not-bound"
}

func (r *DatasetNotBoundRule) Match(ctx types.DiagnosticContext) bool {
	for _, dataset := range ctx.Graph.Datasets {
		if dataset.Status == "NotBound" || dataset.Status == "" {
			return true
		}
		for _, cond := range dataset.Conditions {
			if cond.Type == "Ready" && cond.Status == "False" {
				return true
			}
		}
	}
	return false
}

func (r *DatasetNotBoundRule) Hypothesis(ctx types.DiagnosticContext) types.Hypothesis {
	var evidence []string
	confidence := types.ConfidenceConditionOnly

	for name, dataset := range ctx.Graph.Datasets {
		if dataset.Status == "NotBound" || dataset.Status == "" {
			statusStr := dataset.Status
			if statusStr == "" {
				statusStr = "<empty>"
			}
			evidence = append(evidence, fmt.Sprintf("Dataset %s/%s: Status=%s",
				dataset.Namespace, name, statusStr))
			confidence = types.ConfidencePodStatusOnly
		}
		for _, cond := range dataset.Conditions {
			if cond.Type == "Ready" && cond.Status == "False" {
				evidence = append(evidence, fmt.Sprintf("Dataset %s/%s: Condition Ready=%s, reason=%s",
					dataset.Namespace, name, cond.Status, cond.Reason))
				if cond.Reason != "" {
					confidence = types.ConfidenceEventAndStatus
				}
			}
		}
	}

	return types.Hypothesis{
		Confidence: confidence,
		Component:  "Dataset",
		Issue:      "Dataset is not bound, likely due to missing or failed Runtime",
		Evidence:   evidence,
		Suggestion: "Ensure a Runtime (e.g., AlluxioRuntime, JuiceFSRuntime) is created for this Dataset. Check Runtime status for failures.",
	}
}
