package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/4JesusApps/prayertexter/internal/awscfg"
	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/repository"
	"github.com/4JesusApps/prayertexter/internal/service"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
)

var version string // do not remove or modify

func handler(ctx context.Context, snsEvent events.SNSEvent) error {
	slog.InfoContext(ctx, "running prayertexter", "version", version)

	if len(snsEvent.Records) == 0 {
		return errors.New("lambda handler: sns event contained no records")
	}
	if len(snsEvent.Records) > 1 {
		slog.WarnContext(ctx, "processing batched SNS event", "records", len(snsEvent.Records))
	}

	cfg := config.Load()

	awsCfg, err := awscfg.GetAwsConfig(ctx, cfg.AWS.Region, cfg.AWS.Retry, cfg.AWS.Backoff)
	if err != nil {
		return fmt.Errorf("lambda handler: failed to get aws config: %w", err)
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
	blocked := repository.NewBlockedPhonesRepository(
		ddbClnt, cfg.AWS.DB.BlockedPhonesTable, cfg.AWS.DB.Timeout,
	)
	intercessors := repository.NewIntercessorPhonesRepository(
		ddbClnt, cfg.AWS.DB.IntercessorPhonesTable, cfg.AWS.DB.Timeout,
	)

	sender := messaging.NewPinpointSender(smsClnt, cfg.AWS.SMS.PhonePool, cfg.AWS.SMS.Timeout)

	memberSvc := service.NewMemberService(members, intercessors, prayers, sender, cfg)
	prayerSvc := service.NewPrayerService(members, intercessors, prayers, sender, cfg)
	adminSvc := service.NewAdminService(members, blocked, sender, memberSvc)
	router := service.NewRouter(members, blocked, memberSvc, prayerSvc, adminSvc)

	var recordErrs []error
	for idx, record := range snsEvent.Records {
		var msg domain.TextMessage
		if err = json.Unmarshal([]byte(record.SNS.Message), &msg); err != nil {
			recordErrs = append(recordErrs, fmt.Errorf(
				"lambda handler: failed to unmarshal sns record %d (%s): %w",
				idx, record.SNS.MessageID, err,
			))
			continue
		}
		if err = router.Handle(ctx, msg); err != nil {
			recordErrs = append(recordErrs, fmt.Errorf(
				"lambda handler: failed to process sns record %d (%s): %w",
				idx, record.SNS.MessageID, err,
			))
		}
	}

	return errors.Join(recordErrs...)
}

func main() {
	lambda.Start(handler)
}
