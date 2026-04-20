package messaging

import (
	"bytes"
	"text/template"
)

var (
	PrayerIntroTmpl = template.Must(template.New("prayerIntro").Parse(
		"Hello! Please pray for {{.Name}}:\n\n"))

	ProfanityDetectedTmpl = template.Must(template.New("profanity").Parse(
		"There was profanity found in your message:\n\n{{.Word}}\n\nPlease try again"))

	PrayerConfirmationTmpl = template.Must(template.New("prayerConfirmation").Parse(
		"You're prayer request has been prayed for by {{.Name}}."))

	PrayerReminderTmpl = template.Must(template.New("prayerReminder").Parse(
		"This is a friendly reminder to pray for {{.Name}}:\n\n"))
)

func Render(tmpl *template.Template, data any) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
