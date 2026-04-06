// Package app provides application bootstrap wiring.
package app

import (
	"context"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/service"
)

// App holds the initialized application.
type App struct {
	Service *service.Service
}

// New creates a fully-initialized App from config.
func New(ctx context.Context, cfg *config.Config) (*App, error) {
	ddbClnt, err := db.GetDdbClient(ctx, &cfg.AWS)
	if err != nil {
		return nil, err
	}

	smsClnt, err := messaging.GetSmsClient(ctx, &cfg.AWS)
	if err != nil {
		return nil, err
	}

	svc := service.NewService(cfg, ddbClnt, smsClnt)

	return &App{Service: svc}, nil
}
