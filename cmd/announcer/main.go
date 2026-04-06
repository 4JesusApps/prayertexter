/*
Announcer is a helper application for the main application prayertexter. Announcer can be used to send announcements
to all prayertexter members. This could be for general updates, taking down or turning up prayertexter, or alerts for
things such as outages or service restoration.
*/
package main

import (
	"log/slog"

	"github.com/4JesusApps/prayertexter/internal/handler"
	"github.com/aws/aws-lambda-go/lambda"
)

// MUST BE SET by go build -ldflags "-X main.version=999" like 0.6.14-0-g26fe727 or 0.6.14-2-g9118702-dirty.
var version string // do not remove or modify

func main() {
	slog.Info("starting announcer", "version", version)
	lambda.Start(handler.NewAnnouncerHandler(version))
}
