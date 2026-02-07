package engine

import "github.com/mrhapile/fluid-ai-diagnoser/pkg/types"

// Rule defines the interface for deterministic reasoning rules.
// Each rule inspects the DiagnosticContext and produces a Hypothesis if matched.
type Rule interface {
	// ID returns a unique identifier for this rule.
	ID() string

	// Match returns true if this rule applies to the given context.
	Match(ctx types.DiagnosticContext) bool

	// Hypothesis generates a hypothesis based on the context.
	// This should only be called if Match returns true.
	Hypothesis(ctx types.DiagnosticContext) types.Hypothesis
}
