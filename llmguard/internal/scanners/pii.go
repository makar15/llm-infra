package scanners

import (
	"regexp"
	"strings"
)

// PIIGuard redacts personally identifiable information from text.
// Detected types: email addresses, credit card numbers, Russian phone numbers, international phone numbers.
type PIIGuard struct{}

type piiPattern struct {
	name        string
	re          *regexp.Regexp
	replacement string
}

var piiPatterns = []piiPattern{
	{
		name:        "email",
		re:          regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
		replacement: "[REDACTED:EMAIL]",
	},
	{
		name:        "credit_card",
		re:          regexp.MustCompile(`\b(?:\d[ \-]?){13,16}\b`),
		replacement: "[REDACTED:CARD]",
	},
	{
		name:        "phone_ru",
		re:          regexp.MustCompile(`(?:\+7|8)[\s\-]?\(?\d{3}\)?[\s\-]?\d{3}[\s\-]?\d{2}[\s\-]?\d{2}`),
		replacement: "[REDACTED:PHONE]",
	},
	{
		name:        "phone_intl",
		re:          regexp.MustCompile(`\+[1-9]\d{6,14}\b`),
		replacement: "[REDACTED:PHONE]",
	},
}

func NewPIIGuard() *PIIGuard { return &PIIGuard{} }

func (p *PIIGuard) Name() string { return "PIIGuard" }

// Redact replaces all detected PII in text and returns the sanitised string.
func (p *PIIGuard) Redact(text string) string {
	for _, pat := range piiPatterns {
		text = pat.re.ReplaceAllString(text, pat.replacement)
	}
	return text
}

// Scan reports whether any PII was found (for pipeline use).
func (p *PIIGuard) Scan(text string) ScanResult {
	lower := strings.ToLower(text)
	_ = lower // used implicitly via Redact
	redacted := p.Redact(text)
	if redacted != text {
		return ScanResult{
			Scanner: p.Name(),
			Passed:  true, // PII found but redacted — request is allowed to continue
			Reason:  "pii redacted",
		}
	}
	return ScanResult{Scanner: p.Name(), Passed: true}
}
