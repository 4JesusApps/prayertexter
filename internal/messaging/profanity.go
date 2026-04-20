package messaging

import (
	"slices"

	goaway "github.com/TwiN/go-away"
)

func CheckProfanity(text string) string {
	profanities := &goaway.DefaultProfanities
	*profanities = slices.DeleteFunc(*profanities, func(s string) bool {
		return s == "jerk" || s == "ass" || s == "butt"
	})
	detector := goaway.NewProfanityDetector().WithSanitizeSpaces(false)
	return detector.ExtractProfanity(text)
}
