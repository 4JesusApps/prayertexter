package messaging_test

import (
	"testing"

	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderPrayerIntro(t *testing.T) {
	result, err := messaging.Render(messaging.PrayerIntroTmpl, struct{ Name string }{"John"})
	require.NoError(t, err)
	assert.Equal(t, "Hello! Please pray for John:\n\n", result)
}

func TestRenderProfanityDetected(t *testing.T) {
	result, err := messaging.Render(messaging.ProfanityDetectedTmpl, struct{ Word string }{"badword"})
	require.NoError(t, err)
	assert.Contains(t, result, "badword")
}

func TestRenderPrayerConfirmation(t *testing.T) {
	result, err := messaging.Render(messaging.PrayerConfirmationTmpl, struct{ Name string }{"Jane"})
	require.NoError(t, err)
	assert.Contains(t, result, "Jane")
}

func TestRenderPrayerReminder(t *testing.T) {
	result, err := messaging.Render(messaging.PrayerReminderTmpl, struct{ Name string }{"Bob"})
	require.NoError(t, err)
	assert.Contains(t, result, "Bob")
}
