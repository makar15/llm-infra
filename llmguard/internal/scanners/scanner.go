package scanners

import (
	"fmt"

	"github.com/local/llmguard/internal/config"
)

// ScanResult holds the outcome of a single scanner.
type ScanResult struct {
	Scanner string
	Passed  bool
	Reason  string
}

// Scanner is the interface every scanner must implement.
type Scanner interface {
	Name() string
	Scan(text string) ScanResult
}

// BuildInputScanners instantiates input scanners from config.
func BuildInputScanners(cfgs []config.ScannerConfig) ([]Scanner, error) {
	var out []Scanner
	for _, c := range cfgs {
		s, err := buildScanner(c)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// BuildOutputScanners instantiates output scanners from config.
func BuildOutputScanners(cfgs []config.ScannerConfig) ([]Scanner, error) {
	return BuildInputScanners(cfgs) // same registry, different placement
}

func buildScanner(c config.ScannerConfig) (Scanner, error) {
	switch c.Name {
	case "TokenLimit":
		max := c.MaxTokens
		if max == 0 {
			max = 4096
		}
		return NewTokenLimit(max), nil
	case "PromptInjection":
		thr := c.Threshold
		if thr == 0 {
			thr = 0.85
		}
		return NewPromptInjection(thr), nil
	case "Toxicity":
		thr := c.Threshold
		if thr == 0 {
			thr = 0.85
		}
		return NewToxicity(thr), nil
	case "Relevance":
		thr := c.Threshold
		if thr == 0 {
			thr = 0.1
		}
		return NewRelevance(thr), nil
	case "PIIGuard":
		return NewPIIGuard(), nil
	default:
		return nil, fmt.Errorf("unknown scanner %q", c.Name)
	}
}

// RunAll runs all scanners against text; returns the first failure or a pass.
func RunAll(scanners []Scanner, text string) *ScanResult {
	for _, s := range scanners {
		r := s.Scan(text)
		if !r.Passed {
			return &r
		}
	}
	return nil
}
