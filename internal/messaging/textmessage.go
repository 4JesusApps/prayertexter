/*
Package messaging implements sms operations (sending text messages). It is also used to organize the different messages
that are sent out by the prayertexter application.
*/
package messaging

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/4JesusApps/prayertexter/internal/utility"
	goaway "github.com/TwiN/go-away"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2/types"
	"github.com/aws/smithy-go"
	"github.com/spf13/viper"
)

// A TextMessage represents a received text message from a user.
type TextMessage struct {
	// Body is the text message content.
	Body string `json:"messageBody"`
	// Phone is the phone number of the text message sender.
	Phone string `json:"originationNumber"`
}

type TextSender interface {
	SendTextMessage(ctx context.Context, params *pinpointsmsvoicev2.SendTextMessageInput,
		optFns ...func(*pinpointsmsvoicev2.Options)) (*pinpointsmsvoicev2.SendTextMessageOutput, error)
}

// GetSmsClient returns a pinpoint sms client that can be used for sending text messages.
func GetSmsClient(ctx context.Context) (*pinpointsmsvoicev2.Client, error) {
	cfg, err := utility.GetAwsConfig(ctx)
	if err != nil {
		return nil, utility.WrapError(err, "failed to get sms client")
	}

	smsClnt := pinpointsmsvoicev2.NewFromConfig(cfg)

	return smsClnt, nil
}

// SendText sends a text message via pinpoint sms.
func SendText(ctx context.Context, smsClnt TextSender, msg TextMessage) error {
	body := MsgPre + msg.Body + "\n\n" + MsgPost

	// This helps with SAM local testing. We don't want to actually send a SMS when doing SAM local tests (for now).
	// However when unit testing, we can't skip this part since this is mocked and receives inputs.
	if utility.IsAwsLocal() {
		slog.InfoContext(ctx, "sent text message (local)", "phone", msg.Phone, "body", body)
		return nil
	}

	phonepool := viper.GetString(PhonePoolConfigPath)
	input := &pinpointsmsvoicev2.SendTextMessageInput{
		DestinationPhoneNumber: aws.String(msg.Phone),
		MessageBody:            aws.String(body),
		MessageType:            types.MessageTypeTransactional,
		OriginationIdentity:    aws.String(phonepool),
	}

	timeoutSec := viper.GetInt(TimeoutConfigPath)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	const maxAttempts = 3
	const sleepDuration = 500
	var lastErr error

	// Checks if there is a ThrottlingException error and if so, perform small sleep to wait out the AWS throttle
	// threshold limit. This happens when too many SMS messages are sent per second. AWS will throttle if we go over the
	// specified amount which could lead to failed SMS message delivery.
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_, err := smsClnt.SendTextMessage(ctx, input)
		if err == nil {
			return nil
		}
		lastErr = err

		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "ThrottlingException" && attempt < maxAttempts {
			slog.WarnContext(ctx, "throttled by Pinpoint, retrying", "attempt", attempt, "phone", msg.Phone)
			time.Sleep(sleepDuration * time.Millisecond)
			continue
		}

		// not retryable or out of attempts
		break
	}

	return utility.LogAndWrapError(ctx, lastErr, "failed to send text message", "phone", msg.Phone, "msg", msg.Body)
}

// CheckProfanity returns any detected profanity found inside a string. This will return an empty string if no profanity
// is detected.
func (t TextMessage) CheckProfanity() string {
	// We need to remove some words from the profanity filter because it is too sensitive.
	removedWords := []string{"jerk", "ass"}
	profanities := &goaway.DefaultProfanities

	for _, word := range removedWords {
		utility.RemoveItem(profanities, word)
	}

	return goaway.ExtractProfanity(t.Body)
}
