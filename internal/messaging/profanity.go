package messaging

import (
	goaway "github.com/TwiN/go-away"
)

// ProfanityChecker performs profanity detection using a pre-filtered word list.
// It is safe for concurrent use because the filtered list is computed once at construction
// and the detector is read-only after initialization.
type ProfanityChecker struct {
	detector *goaway.ProfanityDetector
}

// NewProfanityChecker creates a ProfanityChecker with the default profanity list,
// minus the supplied exceptions. The global DefaultProfanities is never mutated.
func NewProfanityChecker(exceptions []string) *ProfanityChecker {
	skip := make(map[string]struct{}, len(exceptions))
	for _, w := range exceptions {
		skip[w] = struct{}{}
	}

	filtered := make([]string, 0, len(goaway.DefaultProfanities))
	for _, w := range goaway.DefaultProfanities {
		if _, ok := skip[w]; !ok {
			filtered = append(filtered, w)
		}
	}

	detector := goaway.NewProfanityDetector().
		WithSanitizeSpaces(false).
		WithCustomDictionary(filtered, goaway.DefaultFalsePositives, goaway.DefaultFalseNegatives)

	return &ProfanityChecker{detector: detector}
}

// Check returns any detected profanity inside str, or an empty string if clean.
func (pc *ProfanityChecker) Check(str string) string {
	return pc.detector.ExtractProfanity(str)
}
