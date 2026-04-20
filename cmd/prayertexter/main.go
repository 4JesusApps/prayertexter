package main

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/repository"
	"github.com/4JesusApps/prayertexter/internal/service"
	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
)

var version string // do not remove or modify

func handler(ctx context.Context, snsEvent events.SNSEvent) {
	slog.InfoContext(ctx, "running prayertexter", "version", version)

	if len(snsEvent.Records) > 1 {
		for _, record := range snsEvent.Records {
			slog.ErrorContext(ctx, "lambda handler: there are more than 1 SNS records! This is unexpected and only "+
				"the first record will be handled", "message", record.SNS.Message, "messageid", record.SNS.MessageID)
		}
	}

	var msg domain.TextMessage
	if err := json.Unmarshal([]byte(snsEvent.Records[0].SNS.Message), &msg); err != nil {
		slog.ErrorContext(ctx, "lambda handler: failed to unmarshal api gateway request", "error", err)
		return
	}

	cfg := config.Load()

	awsCfg, err := utility.GetAwsConfig(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "lambda handler: failed to get aws config", "error", err)
		return
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
	blocked := repository.NewBlockedPhonesRepository(ddbClnt, cfg.AWS.DB.BlockedPhonesTable, cfg.AWS.DB.Timeout)
	intercessors := repository.NewIntercessorPhonesRepository(ddbClnt, cfg.AWS.DB.IntercessorPhonesTable, cfg.AWS.DB.Timeout)

	sender := messaging.NewPinpointSender(smsClnt, cfg.AWS.SMS.PhonePool, cfg.AWS.SMS.Timeout)

	memberSvc := service.NewMemberService(members, intercessors, prayers, sender, cfg)
	prayerSvc := service.NewPrayerService(members, intercessors, prayers, sender, cfg)
	adminSvc := service.NewAdminService(members, blocked, sender, memberSvc)
	router := service.NewRouter(members, blocked, memberSvc, prayerSvc, adminSvc)

	if err = router.Handle(ctx, msg); err != nil {
		return
	}
}

func main() {
	lambda.Start(handler)
}
