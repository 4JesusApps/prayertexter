package messaging_test

import (
	"testing"

	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/stretchr/testify/require"

	goaway "github.com/TwiN/go-away"
)

func TestCheckProfanity_CleanText(t *testing.T) {
	require.Empty(t, messaging.CheckProfanity("please pray for my family today"))
}

func TestCheckProfanity_AllowlistedWords(t *testing.T) {
	for _, word := range []string{"jerk", "ass", "butt"} {
		require.Empty(t, messaging.CheckProfanity("please pray about my "+word+" today"),
			"allowlisted word %q must not be flagged", word)
	}
}

func TestCheckProfanity_DoesNotMutateGlobalDictionary(t *testing.T) {
	before := append([]string(nil), goaway.DefaultProfanities...)
	_ = messaging.CheckProfanity("please pray about my jerk today")
	_ = messaging.CheckProfanity("hello world")
	require.Equal(t, before, goaway.DefaultProfanities,
		"CheckProfanity must not mutate goaway.DefaultProfanities")
}
