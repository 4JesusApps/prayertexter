package messaging

import (
	"slices"

	goaway "github.com/TwiN/go-away"
)

var profanityDetector = newProfanityDetector()

func newProfanityDetector() *goaway.ProfanityDetector {
	filtered := slices.DeleteFunc(slices.Clone(goaway.DefaultProfanities), func(s string) bool {
		return s == "jerk" || s == "ass" || s == "butt"
	})
	return goaway.NewProfanityDetector().
		WithCustomDictionary(filtered, goaway.DefaultFalsePositives, goaway.DefaultFalseNegatives).
		WithSanitizeSpaces(false)
}

func CheckProfanity(text string) string {
	return profanityDetector.ExtractProfanity(text)
}
