package main

import (
	"context"
	"log/slog"

	"github.com/4JesusApps/prayertexter/internal/awscfg"
	"github.com/4JesusApps/prayertexter/internal/buildinfo"
	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/repository"
	"github.com/4JesusApps/prayertexter/internal/service"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
)

func handler(ctx context.Context) error {
	slog.InfoContext(ctx, "running statecontroller", "version", buildinfo.Version())

	cfg := config.Load()

	awsCfg, err := awscfg.GetAwsConfig(ctx, cfg.AWS.Region, cfg.AWS.Retry, cfg.AWS.Backoff)
	if err != nil {
		return err
	}

	ddbClnt := dynamodb.NewFromConfig(awsCfg)
	smsClnt := pinpointsmsvoicev2.NewFromConfig(awsCfg)

	members := repository.NewMemberRepository(ddbClnt, cfg.AWS.DB.MemberTable, cfg.AWS.DB.Timeout)
	prayers := repository.NewPrayerRepository(
		ddbClnt,
		cfg.AWS.DB.ActivePrayerTable,
		cfg.AWS.DB.QueuedPrayerTable,
		cfg.AWS.DB.Timeout,
	)
	intercessors := repository.NewIntercessorPhonesRepository(
		ddbClnt, cfg.AWS.DB.IntercessorPhonesTable, cfg.AWS.DB.Timeout,
	)

	sender := messaging.NewPinpointSender(smsClnt, cfg.AWS.SMS.PhonePool, cfg.AWS.SMS.Timeout)

	prayerSvc := service.NewPrayerService(members, intercessors, prayers, sender, cfg)
	return prayerSvc.RunScheduledJobs(ctx)
}

func main() {
	lambda.Start(handler)
}
