package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	goaway "github.com/TwiN/go-away"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2/types"
	"github.com/mshort55/prayertexter/internal/utility"
	"github.com/spf13/viper"
)

const (
	DefaultPhone    = "+12762908579"
	phoneConfigPath = "conf.aws.sms.phone"

	DefaultTimeout    = 60
	timeoutConfigPath = "conf.aws.sms.timeout"

	// Sign up messages.
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

	// Prayer request messages.
	MsgProfanityFound = "There was profanity found in your prayer request:\n\nPLACEHOLDER\n\nPlease try the request " +
		"again without this word or words."
	MsgPrayerIntro  = "Hello! Please pray for PLACEHOLDER:\n"
	MsgPrayerQueued = "We could not find any available intercessors. Your prayer has been added to the queue and " +
		"will get sent out as soon as someone is available."
	MsgPrayerSentOut = "Your prayer request has been sent out!"

	// Prayer completion messages.
	MsgNoActivePrayer     = "You have no more active prayers to mark as prayed"
	MsgPrayerThankYou     = "Thank you for praying!"
	MsgPrayerConfirmation = "You're prayer request has been prayed for by PLACEHOLDER"

	// Other.
	MsgHelp = "To receive support, please email info@4jesusministries.com or call/text (657) 217-1678. " +
		"Thank you!"
	MsgPre  = "PrayerTexter: "
	MsgPost = "Reply HELP for help or STOP to cancel."
)

type TextMessage struct {
	Body  string `json:"body"`
	Phone string `json:"phone-number"`
}

type TextSender interface {
	SendTextMessage(ctx context.Context,
		params *pinpointsmsvoicev2.SendTextMessageInput,
		optFns ...func(*pinpointsmsvoicev2.Options)) (*pinpointsmsvoicev2.SendTextMessageOutput, error)
}

func GetSmsClient() (*pinpointsmsvoicev2.Client, error) {
	cfg, err := utility.GetAwsConfig()
	if err != nil {
		return nil, utility.WrapError(err, "failed to get sms client")
	}

	smsClnt := pinpointsmsvoicev2.NewFromConfig(cfg)

	return smsClnt, nil
}

func SendText(smsClnt TextSender, msg TextMessage) error {
	body := MsgPre + msg.Body + "\n\n" + MsgPost

	// This helps with SAM local testing. We don't want to actually send a SMS when doing SAM local tests (for now).
	// However when unit testing, we can't skip this part since this is mocked and receives inputs.
	if !utility.IsAwsLocal() {
		timeout := viper.GetInt(timeoutConfigPath)
		ctx, cancel := context.WithTimeout(context.Background(),
			time.Duration(timeout)*time.Second)
		defer cancel()

		phone := viper.GetString(phoneConfigPath)
		input := &pinpointsmsvoicev2.SendTextMessageInput{
			DestinationPhoneNumber: aws.String(msg.Phone),
			MessageBody:            aws.String(body),
			MessageType:            types.MessageTypeTransactional,
			OriginationIdentity:    aws.String(phone),
		}

		_, err := smsClnt.SendTextMessage(ctx, input)

		return utility.WrapError(err, fmt.Sprintf("failed to send text message to %s", msg.Phone))
	}

	slog.Info("sent text message", "phone", msg.Phone, "body", body)
	return nil
}

func (t TextMessage) CheckProfanity() string {
	// We need to remove some words from the profanity filter because it is too sensitive.
	removedWords := []string{"jerk"}
	profanities := &goaway.DefaultProfanities

	for _, word := range removedWords {
		utility.RemoveItem(profanities, word)
	}

	return goaway.ExtractProfanity(t.Body)
}
