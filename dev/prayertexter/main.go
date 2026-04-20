/*
This file is used for sam local testing only. It uses an api gateway trigger which is easier to set and up with sam
local.
*/
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

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

// MUST BE SET by go build -ldflags "-X main.version=999" like 0.6.14-0-g26fe727 or 0.6.14-2-g9118702-dirty.
var version string // do not remove or modify

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	slog.InfoContext(ctx, "running prayertexter", "version", version)

	var msg domain.TextMessage
	if err := json.Unmarshal([]byte(req.Body), &msg); err != nil {
		slog.ErrorContext(ctx, "lambda handler: failed to unmarshal api gateway request", "error", err)
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	cfg := config.Load()

	awsCfg, err := awscfg.GetAwsConfig(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "lambda handler: failed to get aws config", "error", err)
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
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

	if err = router.Handle(ctx, msg); err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: "Success\n"}, nil
}

func main() {
	lambda.Start(handler)
}
