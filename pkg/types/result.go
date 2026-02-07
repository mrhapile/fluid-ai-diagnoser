package types

import "time"

type DiagnosisResult struct {
	Hypotheses  []Hypothesis `json:"hypotheses"`
	GeneratedAt time.Time    `json:"generatedAt"`
	Engine      string       `json:"engine"` // "rule-based"
}
