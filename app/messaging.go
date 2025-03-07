package prayertexter

import (
	"context"
	"fmt"
	"log/slog"

	goaway "github.com/TwiN/go-away"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2/types"
)

const (
	// sign up messages
	msgNameRequest             = "Reply your name, or 2 to stay anonymous"
	msgMemberTypeRequest       = "Reply 1 to send prayer request, or 2 to be added to the intercessors list (to pray for others). 2 will also allow you to send in prayer requests."
	msgPrayerInstructions      = "You are now signed up to send prayer requests! You can send them directly to this number at any time. You will be alerted when someone has prayed for your request."
	msgPrayerNumRequest        = "Reply with the number of maximum prayer texts you are willing to receive and pray for each week"
	msgIntercessorInstructions = "You are now signed up to receive prayer requests. Please try to pray for the requests ASAP. Once you are done praying, send 'prayed' back to this number for confirmation."
	msgWrongInput              = "Wrong input received during sign up process, please try again"
	msgSignUpConfirmation      = "You have opted in to PrayerTexter. Msg & data rates may apply."
	msgRemoveUser              = "You have been removed from PrayerTexter. To sign back up, text the word pray to this number."

	// prayer request messages
	msgProfanityFound = "There was profanity found in your prayer request:\n\nPLACEHOLDER\n\nPlease try the request again without this word or words."
	msgPrayerIntro    = "Hello! Please pray for PLACEHOLDER:\n"
	msgPrayerQueued   = "We could not find any available intercessors. Your prayer has been added to the queue and will get sent out as soon as someone is available."
	msgPrayerSentOut  = "Your prayer request has been sent out!"

	// prayer completion messages
	msgNoActivePrayer     = "You have no more active prayers to mark as prayed"
	msgPrayerThankYou     = "Thank you for praying!"
	msgPrayerConfirmation = "You're prayer request has been prayed for by PLACEHOLDER"

	// other
	msgPre            = "PrayerTexter: "
	msgPost           = "Reply HELP for help or STOP to cancel."
	msgHelp           = "To receive support, please email info@4jesusministries.com or call/text (657) 217-1678. Thank you!"
	prayerTexterPhone = "+12762908579"
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
	cfg, err := getAwsConfig()
	if err != nil {
		return nil, fmt.Errorf("GetSmsClient: %w", err)
	}

	smsClnt := pinpointsmsvoicev2.NewFromConfig(cfg)

	return smsClnt, nil
}

func sendText(smsClnt TextSender, msg TextMessage) error {
	body := msgPre + msg.Body + "\n\n" + msgPost

	input := &pinpointsmsvoicev2.SendTextMessageInput{
		DestinationPhoneNumber: aws.String(msg.Phone),
		MessageBody:            aws.String(body),
		MessageType:            types.MessageTypeTransactional,
		OriginationIdentity:    aws.String(prayerTexterPhone),
	}

	if _, err := smsClnt.SendTextMessage(context.TODO(), input); err != nil {
		return err
	}

	// this helps with unit testing and sam local testing so you can view the text message flow from the logs
	if isAwsLocal() {
		slog.Info("sent text message", "phone", msg.Phone, "body", msg.Body)
	}

	return nil
}

func (t TextMessage) checkProfanity() string {
	// We need to remove some words from the profanity filter because it is too sensitive
	removedWords := []string{"jerk"}
	profanities := &goaway.DefaultProfanities

	for _, word := range removedWords {
		removeItem(profanities, word)
	}

	return goaway.ExtractProfanity(t.Body)
}
