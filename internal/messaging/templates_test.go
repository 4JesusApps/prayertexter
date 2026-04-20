package messaging_test

import (
	"testing"
	"text/template"

	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name     string
		tmpl     *template.Template
		data     any
		contains string
	}{
		{"prayer intro", messaging.PrayerIntroTmpl, struct{ Name string }{"John"}, "Hello! Please pray for John:\n\n"},
		{"profanity detected", messaging.ProfanityDetectedTmpl, struct{ Word string }{"badword"}, "badword"},
		{"prayer confirmation", messaging.PrayerConfirmationTmpl, struct{ Name string }{"Jane"}, "Jane"},
		{"prayer reminder", messaging.PrayerReminderTmpl, struct{ Name string }{"Bob"}, "Bob"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := messaging.Render(tt.tmpl, tt.data)
			require.NoError(t, err)
			assert.Contains(t, result, tt.contains)
		})
	}
}
