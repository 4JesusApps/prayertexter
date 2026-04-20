package messaging

import (
	"slices"

	goaway "github.com/TwiN/go-away"
)

var profanityDetector *goaway.ProfanityDetector

func init() {
	goaway.DefaultProfanities = slices.DeleteFunc(goaway.DefaultProfanities, func(s string) bool {
		return s == "jerk" || s == "ass" || s == "butt"
	})
	profanityDetector = goaway.NewProfanityDetector().WithSanitizeSpaces(false)
}

func CheckProfanity(text string) string {
	return profanityDetector.ExtractProfanity(text)
}
