package types

type Hypothesis struct {
	Rank       int      `json:"rank"`
	Confidence float64  `json:"confidence"`
	Component  string   `json:"component"`
	Issue      string   `json:"issue"`
	Evidence   []string `json:"evidence"`
	Suggestion string   `json:"suggestion"`
}
