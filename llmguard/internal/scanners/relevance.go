package scanners

import (
	"strings"
)

// Relevance checks that the output is not trivially empty or a refusal boilerplate.
// A real implementation would use embedding cosine similarity between input and output.
type Relevance struct {
	threshold float64
}

// refusalPhrases are LLM boilerplate non-answers that indicate a failed response.
var refusalPhrases = []string{
	"i cannot assist",
	"i can't assist",
	"i am unable to",
	"i'm unable to",
	"as an ai language model, i",
	"i don't have the ability",
	"that is outside my",
	"i'm just an ai",
}

func NewRelevance(threshold float64) *Relevance {
	return &Relevance{threshold: threshold}
}

func (r *Relevance) Name() string { return "Relevance" }

func (r *Relevance) Scan(text string) ScanResult {
	trimmed := strings.TrimSpace(text)

	// Empty output is never relevant.
	if len(trimmed) < 3 {
		return ScanResult{
			Scanner: r.Name(),
			Passed:  false,
			Reason:  "output too short or empty",
		}
	}

	lower := strings.ToLower(trimmed)
	for _, phrase := range refusalPhrases {
		if strings.Contains(lower, phrase) {
			return ScanResult{
				Scanner: r.Name(),
				Passed:  false,
				Reason:  "output appears to be a refusal",
			}
		}
	}

	return ScanResult{Scanner: r.Name(), Passed: true}
}
