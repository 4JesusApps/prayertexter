package main

import (
	"context"
	"log/slog"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/repository"
	"github.com/4JesusApps/prayertexter/internal/service"
	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
)

var version string // do not remove or modify

func handler(ctx context.Context) {
	slog.InfoContext(ctx, "running statecontroller", "version", version)

	cfg := config.Load()

	awsCfg, err := utility.GetAwsConfig(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "lambda handler: failed to get aws config", "error", err)
		return
	}

	ddbClnt := dynamodb.NewFromConfig(awsCfg)
	smsClnt := pinpointsmsvoicev2.NewFromConfig(awsCfg)

	members := repository.NewMemberRepository(ddbClnt, cfg.AWS.DB.MemberTable, cfg.AWS.DB.Timeout)
	prayers := repository.NewPrayerRepository(ddbClnt, cfg.AWS.DB.ActivePrayerTable, cfg.AWS.DB.QueuedPrayerTable, cfg.AWS.DB.Timeout)
	intercessors := repository.NewIntercessorPhonesRepository(ddbClnt, cfg.AWS.DB.IntercessorPhonesTable, cfg.AWS.DB.Timeout)

	sender := messaging.NewPinpointSender(smsClnt, cfg.AWS.SMS.PhonePool, cfg.AWS.SMS.Timeout)

	prayerSvc := service.NewPrayerService(members, intercessors, prayers, sender, cfg)
	prayerSvc.RunScheduledJobs(ctx)
}

func main() {
	lambda.Start(handler)
}
