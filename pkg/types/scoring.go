package types

// Confidence score constants based on evidence strength.
// These are heuristic values, not probabilistic.
const (
	// ConfidenceEventAndStatus is assigned when both an event and pod/resource status confirm the issue.
	ConfidenceEventAndStatus = 0.8

	// ConfidencePodStatusOnly is assigned when only pod status is available.
	ConfidencePodStatusOnly = 0.6

	// ConfidenceConditionOnly is assigned when only a condition is available, without event correlation.
	ConfidenceConditionOnly = 0.5

	// ConfidenceLogMatch is assigned when the evidence comes from log analysis.
	ConfidenceLogMatch = 0.55

	// ConfidenceEventOnly is assigned when only an event matches, without status confirmation.
	ConfidenceEventOnly = 0.5

	// ConfidenceLow is a fallback for weak signals.
	ConfidenceLow = 0.3
)

// Severity levels for sorting hypotheses.
const (
	SeverityCritical = 1
	SeverityHigh     = 2
	SeverityMedium   = 3
	SeverityLow      = 4
)
