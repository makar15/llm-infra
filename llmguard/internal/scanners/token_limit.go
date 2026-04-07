package scanners

import (
	"fmt"
	"strings"
)

// TokenLimit blocks input that exceeds a token budget.
// Uses a simple whitespace-split approximation (1 token ≈ 1 word).
// Swap for a real tokenizer (e.g. tiktoken-go) if you need exact counts.
type TokenLimit struct {
	maxTokens int
}

func NewTokenLimit(max int) *TokenLimit { return &TokenLimit{maxTokens: max} }

func (t *TokenLimit) Name() string { return "TokenLimit" }

func (t *TokenLimit) Scan(text string) ScanResult {
	count := len(strings.Fields(text))
	if count > t.maxTokens {
		return ScanResult{
			Scanner: t.Name(),
			Passed:  false,
			Reason:  fmt.Sprintf("token count %d exceeds limit %d", count, t.maxTokens),
		}
	}
	return ScanResult{Scanner: t.Name(), Passed: true}
}
