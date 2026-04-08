package scanners

import (
	"strings"
)

// PromptInjection detects common prompt-injection patterns.
// This is a heuristic baseline — replace with a model-based classifier for production.
type PromptInjection struct {
	threshold float64
}

// injectionPatterns is a curated list of known injection signatures.
var injectionPatterns = []string{
	// English patterns
	"ignore previous instructions",
	"ignore all previous",
	"disregard previous",
	"forget your instructions",
	"you are now",
	"act as",
	"new instructions:",
	"override instructions",
	"system prompt:",
	"</s>",               // token boundary abuse
	"[system]",           // role hijacking
	"[/system]",
	"[inst]",
	"[/inst]",
	"###instruction",
	"<|im_start|>system",
	// Russian patterns
	"игнорируй все предыдущие",
	"игнорируй предыдущие",
	"забудь предыдущие инструкции",
	"забудь все инструкции",
	"ты теперь",
	"притворись что ты",
	"притворись, что ты",
	"новые инструкции:",
	"системный промпт",
	"выведи system prompt",
	"покажи system prompt",
	"раскрой system prompt",
	"игнорируй ограничения",
	"обойди ограничения",
	"действуй как",
}

func NewPromptInjection(threshold float64) *PromptInjection {
	return &PromptInjection{threshold: threshold}
}

func (p *PromptInjection) Name() string { return "PromptInjection" }

func (p *PromptInjection) Scan(text string) ScanResult {
	lower := strings.ToLower(text)
	hits := 0
	for _, pattern := range injectionPatterns {
		if strings.Contains(lower, pattern) {
			hits++
		}
	}

	// Simple scoring: each hit contributes equally.
	score := float64(hits) / float64(len(injectionPatterns))
	if score >= p.threshold || hits >= 1 {
		return ScanResult{
			Scanner: p.Name(),
			Passed:  false,
			Reason:  "prompt injection pattern detected",
		}
	}
	return ScanResult{Scanner: p.Name(), Passed: true}
}
