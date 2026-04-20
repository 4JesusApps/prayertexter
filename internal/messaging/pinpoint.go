package messaging

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/4JesusApps/prayertexter/internal/apperr"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2/types"
	"github.com/aws/smithy-go"
)

type PinpointClient interface {
	SendTextMessage(ctx context.Context, params *pinpointsmsvoicev2.SendTextMessageInput,
		optFns ...func(*pinpointsmsvoicev2.Options)) (*pinpointsmsvoicev2.SendTextMessageOutput, error)
}

type PinpointSender struct {
	client    PinpointClient
	phonePool string
	timeout   int
}

func NewPinpointSender(client PinpointClient, phonePool string, timeout int) *PinpointSender {
	return &PinpointSender{
		client:    client,
		phonePool: phonePool,
		timeout:   timeout,
	}
}

func (s *PinpointSender) SendMessage(ctx context.Context, to string, body string) error {
	wrappedBody := MsgPre + body + "\n\n" + MsgPost

	if os.Getenv("AWS_SAM_LOCAL") == "true" {
		slog.InfoContext(ctx, "sent text message (local)", "phone", to, "body", wrappedBody)
		return nil
	}

	input := &pinpointsmsvoicev2.SendTextMessageInput{
		DestinationPhoneNumber: aws.String(to),
		MessageBody:            aws.String(wrappedBody),
		MessageType:            types.MessageTypeTransactional,
		OriginationIdentity:    aws.String(s.phonePool),
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.timeout)*time.Second)
	defer cancel()

	const maxAttempts = 3
	const sleepDuration = 500
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_, err := s.client.SendTextMessage(ctx, input)
		if err == nil {
			return nil
		}
		lastErr = err

		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "ThrottlingException" && attempt < maxAttempts {
			slog.WarnContext(ctx, "throttled by Pinpoint, retrying", "attempt", attempt, "phone", to)
			time.Sleep(sleepDuration * time.Millisecond)
			continue
		}

		break
	}

	return apperr.LogAndWrapError(ctx, lastErr, "failed to send text message", "phone", to, "msg", body)
}
