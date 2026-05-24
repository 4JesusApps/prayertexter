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

// messageHandler is the minimal contract processRecords needs from a router.
// *service.Router satisfies it in production; tests substitute a stub so the
// SNS parsing loop can be exercised without spinning up the full service graph.
type messageHandler interface {
	Handle(ctx context.Context, msg domain.TextMessage) error
}

func handler(ctx context.Context, snsEvent events.SNSEvent) error {
	slog.InfoContext(ctx, "running prayertexter", "version", version)

	cfg := config.Load()
	router, err := newRouter(ctx, cfg)
	if err != nil {
		return err
	}
	return processRecords(ctx, router, snsEvent.Records)
}

func newRouter(ctx context.Context, cfg config.Config) (*service.Router, error) {
	awsCfg, err := awscfg.GetAwsConfig(ctx, cfg.AWS.Region, cfg.AWS.Retry, cfg.AWS.Backoff)
	if err != nil {
		return nil, fmt.Errorf("lambda handler: failed to get aws config: %w", err)
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
	return service.NewRouter(members, blocked, memberSvc, prayerSvc, adminSvc), nil
}

func processRecords(ctx context.Context, h messageHandler, records []events.SNSEventRecord) error {
	if len(records) == 0 {
		return errors.New("lambda handler: sns event contained no records")
	}
	if len(records) > 1 {
		slog.WarnContext(ctx, "processing batched SNS event", "records", len(records))
	}

	var recordErrs []error
	for idx, record := range records {
		var msg domain.TextMessage
		if err := json.Unmarshal([]byte(record.SNS.Message), &msg); err != nil {
			recordErrs = append(recordErrs, fmt.Errorf(
				"lambda handler: failed to unmarshal sns record %d (%s): %w",
				idx, record.SNS.MessageID, err,
			))
			continue
		}
		if err := h.Handle(ctx, msg); err != nil {
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
