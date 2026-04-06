/*
This file is used for sam local testing only. It uses an api gateway trigger which is easier to set and up with sam
local.
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

	lambda.Start(handler.NewLocalHandler(a.Service, version))
}
