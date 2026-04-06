// Package handler contains Lambda-specific request/response parsing, delegating to the service layer.
package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/service"
	"github.com/aws/aws-lambda-go/events"
)

// NewSMSHandler returns a Lambda handler for SNS-triggered SMS processing.
func NewSMSHandler(svc *service.Service, version string) func(ctx context.Context, snsEvent events.SNSEvent) {
	return func(ctx context.Context, snsEvent events.SNSEvent) {
		slog.InfoContext(ctx, "running prayertexter", "version", version)
		if len(snsEvent.Records) > 1 {
			for _, record := range snsEvent.Records {
				slog.ErrorContext(
					ctx,
					"lambda handler: more than 1 SNS record, only the first will be handled",
					"message", record.SNS.Message,
					"messageid", record.SNS.MessageID,
				)
			}
		}

		msg := messaging.TextMessage{}
		if err := json.Unmarshal([]byte(snsEvent.Records[0].SNS.Message), &msg); err != nil {
			slog.ErrorContext(ctx, "lambda handler: failed to unmarshal SNS message", "error", err)
			return
		}

		if err := svc.MainFlow(ctx, msg); err != nil {
			return
		}
	}
}

// NewScheduleHandler returns a Lambda handler for EventBridge-triggered scheduled jobs.
func NewScheduleHandler(svc *service.Service, version string) func(ctx context.Context) {
	return func(ctx context.Context) {
		slog.InfoContext(ctx, "running statecontroller", "version", version)
		svc.RunJobs(ctx)
	}
}

// NewLocalHandler returns a Lambda handler for local development via API Gateway.
func NewLocalHandler(svc *service.Service, version string) func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		slog.InfoContext(ctx, "running prayertexter", "version", version)
		msg := messaging.TextMessage{}

		if err := json.Unmarshal([]byte(req.Body), &msg); err != nil {
			slog.ErrorContext(ctx, "lambda handler: failed to unmarshal api gateway request", "error", err)
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
		}

		if err := svc.MainFlow(ctx, msg); err != nil {
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
		}

		return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: "Success\n"}, nil
	}
}

// NewAnnouncerHandler returns a Lambda handler for API Gateway-triggered announcements.
func NewAnnouncerHandler(version string) func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	//nolint:revive // IGNORING UNUSED CTX AND REQ VARIABLES FOR NOW; REMOVE ONCE THIS FUNCTION IS IMPLEMENTED
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		slog.InfoContext(ctx, "running announcer", "version", version)
		return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: "Success"}, nil
	}
}
