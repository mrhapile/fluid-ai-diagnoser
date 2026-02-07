package engine

import (
	"sort"
	"time"

	"github.com/mrhapile/fluid-ai-diagnoser/pkg/rules"
	"github.com/mrhapile/fluid-ai-diagnoser/pkg/types"
)

// Analyze performs deterministic reasoning on the provided DiagnosticContext.
// It is a pure function that:
//   - Never mutates the input
//   - Never performs I/O
//   - Produces deterministic, repeatable output
func Analyze(ctx types.DiagnosticContext) (types.DiagnosisResult, error) {
	// Initialize all rules
	allRules := []Rule{
		&rules.FuseUnschedulableRule{},
		&rules.WorkerPendingMemoryRule{},
		&rules.RuntimePartiallyReadyRule{},
		&rules.PVCUnboundRule{},
		&rules.DatasetNotBoundRule{},
	}

	var hypotheses []types.Hypothesis

	// Apply each rule
	for _, rule := range allRules {
		if rule.Match(ctx) {
			h := rule.Hypothesis(ctx)
			hypotheses = append(hypotheses, h)
		}
	}

	// Sort by confidence (descending) for deterministic ordering
	sort.SliceStable(hypotheses, func(i, j int) bool {
		// Primary: higher confidence first
		if hypotheses[i].Confidence != hypotheses[j].Confidence {
			return hypotheses[i].Confidence > hypotheses[j].Confidence
		}
		// Secondary: alphabetical by component for stability
		if hypotheses[i].Component != hypotheses[j].Component {
			return hypotheses[i].Component < hypotheses[j].Component
		}
		// Tertiary: alphabetical by issue
		return hypotheses[i].Issue < hypotheses[j].Issue
	})

	// Assign ranks after sorting
	for i := range hypotheses {
		hypotheses[i].Rank = i + 1
	}

	return types.DiagnosisResult{
		Hypotheses:  hypotheses,
		GeneratedAt: time.Now().UTC(),
		Engine:      "rule-based",
	}, nil
}
