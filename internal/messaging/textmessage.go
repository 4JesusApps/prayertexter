/*
Package messaging implements sms operations (sending text messages). It is also used to organize the different messages
that are sent out by the prayertexter application.
*/
package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/4JesusApps/prayertexter/internal/utility"
	goaway "github.com/TwiN/go-away"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2/types"
	"github.com/spf13/viper"
)

// Default values for configuration that has been exposed to be used with the config package.
const (
	DefaultPhonePool    = "dummy"
	PhonePoolConfigPath = "conf.aws.sms.phonepool"

	DefaultTimeout    = 60
	TimeoutConfigPath = "conf.aws.sms.timeout"
)

// Sign up stage text message content sent by prayertexter.
const (
	MsgNameRequest       = "Reply your name, or 2 to stay anonymous"
	MsgMemberTypeRequest = "Reply 1 to send prayer request, or 2 to be added to the intercessors list (to pray for " +
		"others). 2 will also allow you to send in prayer requests."
	MsgPrayerInstructions = "You are now signed up to send prayer requests! You can send them directly to this number" +
		" at any time. You will be alerted when someone has prayed for your request."
	MsgPrayerNumRequest = "Reply with the number of maximum prayer texts you are willing to receive and pray for " +
		"each week"
	MsgIntercessorInstructions = "You are now signed up to receive prayer requests. Please try to pray for the " +
		"requests ASAP. Once you are done praying, send 'prayed' back to this number for confirmation."
	MsgWrongInput         = "Wrong input received during sign up process, please try again"
	MsgSignUpConfirmation = "You have opted in to PrayerTexter. Msg & data rates may apply."
	MsgRemoveUser         = "You have been removed from PrayerTexter. To sign back up, text the word pray to this " +
		"number."
)

// Prayer request stage message content sent by prayertexter.
const (
	MsgProfanityFound = "There was profanity found in your prayer request:\n\nPLACEHOLDER\n\nPlease try the request " +
		"again without this word or words."
	MsgPrayerIntro  = "Hello! Please pray for PLACEHOLDER:\n"
	MsgPrayerQueued = "We could not find any available intercessors. Your prayer has been added to the queue and " +
		"will get sent out as soon as someone is available."
	MsgPrayerSentOut = "Your prayer request has been sent out!"
)

// Prayer completion stage message content sent by prayertexter.
const (
	MsgNoActivePrayer     = "You have no more active prayers to mark as prayed"
	MsgPrayerThankYou     = "Thank you for praying!"
	MsgPrayerConfirmation = "You're prayer request has been prayed for by PLACEHOLDER"
)

// Other (general) message content sent by prayertexter.
const (
	MsgHelp = "To receive support, please email info@4jesusministries.com or call/text (657) 217-1678. " +
		"Thank you!"
	MsgPre  = "PrayerTexter: "
	MsgPost = "Reply HELP for help or STOP to cancel."
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
	if !utility.IsAwsLocal() {
		timeout := viper.GetInt(TimeoutConfigPath)
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()

		phonepool := viper.GetString(PhonePoolConfigPath)
		input := &pinpointsmsvoicev2.SendTextMessageInput{
			DestinationPhoneNumber: aws.String(msg.Phone),
			MessageBody:            aws.String(body),
			MessageType:            types.MessageTypeTransactional,
			OriginationIdentity:    aws.String(phonepool),
		}

		_, err := smsClnt.SendTextMessage(ctx, input)

		return utility.WrapError(err, fmt.Sprintf("failed to send text message to %s", msg.Phone))
	}

	slog.InfoContext(ctx, "sent text message", "phone", msg.Phone, "body", body)
	return nil
}

// CheckProfanity returns any detected profanity found inside a string. This will return an empty string if no profanity
// is detected.
func (t TextMessage) CheckProfanity() string {
	// We need to remove some words from the profanity filter because it is too sensitive.
	removedWords := []string{"jerk"}
	profanities := &goaway.DefaultProfanities

	for _, word := range removedWords {
		utility.RemoveItem(profanities, word)
	}

	return goaway.ExtractProfanity(t.Body)
}
