package scanners

import (
	"strings"
)

// Toxicity flags clearly toxic or harmful content using a keyword heuristic.
// For production, wire this to a classifier API (e.g. OpenAI Moderation, Perspective API).
type Toxicity struct {
	threshold float64
}

var toxicKeywords = []string{
	// Violence
	"kill", "murder", "bomb", "shoot", "attack", "stab", "explode",
	// Hate speech signals (very coarse — production should use a classifier)
	"hate", "racist", "slur",
	// Self-harm
	"suicide", "self-harm", "self harm",
	// CSAM / exploitation
	"child abuse", "csam",
}

func NewToxicity(threshold float64) *Toxicity {
	return &Toxicity{threshold: threshold}
}

func (t *Toxicity) Name() string { return "Toxicity" }

func (t *Toxicity) Scan(text string) ScanResult {
	lower := strings.ToLower(text)
	hits := 0
	for _, kw := range toxicKeywords {
		if strings.Contains(lower, kw) {
			hits++
		}
	}

	score := float64(hits) / float64(len(toxicKeywords))
	if score >= t.threshold || hits >= 3 {
		return ScanResult{
			Scanner: t.Name(),
			Passed:  false,
			Reason:  "potentially toxic content detected",
		}
	}
	return ScanResult{Scanner: t.Name(), Passed: true}
}
