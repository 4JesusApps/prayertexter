package messaging

import (
	goaway "github.com/TwiN/go-away"
)

func CheckProfanity(text string) string {
	profanityDetector := goaway.NewProfanityDetector().WithSanitizeSpaces(false)
	removedWords := []string{"jerk", "ass", "butt"}
	profanities := &goaway.DefaultProfanities

	for _, word := range removedWords {
		removeFromSlice(profanities, word)
	}

	return profanityDetector.ExtractProfanity(text)
}

func removeFromSlice(items *[]string, target string) {
	var result []string
	for _, v := range *items {
		if v != target {
			result = append(result, v)
		}
	}
	*items = result
}
