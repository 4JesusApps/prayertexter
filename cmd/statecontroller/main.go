/*
Statecontroller is a helper application for the main application prayertexter. Statecontroller is used to check and
correct various stateful parts of the prayertexter application. Since prayertexter is event driven and only runs when
a text message is received, and since prayertexter is only running in response to a specific text message,
statecontroller fills the gaps that prayertexter is unable to cover. Statecontroller is designed to be ran on a
continuous scheduled basis, and designed to contain multiple individual jobs that run against some part of prayertexter.
*/
package main

import (
	"context"
	"log/slog"

	"github.com/4JesusApps/prayertexter/internal/app"
	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/handler"
	"github.com/aws/aws-lambda-go/lambda"
)

// MUST BE SET by go build -ldflags "-X main.version=999" like 0.6.14-0-g26fe727 or 0.6.14-2-g9118702-dirty.
var version string // do not remove or modify

func main() {
	cfg := config.Load()

	a, err := app.New(context.Background(), cfg)
	if err != nil {
		slog.Error("failed to initialize app", "error", err)
		return
	}

	lambda.Start(handler.NewScheduleHandler(a.Service, version))
}
