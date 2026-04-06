/*
Package prayertexter is the main package for the prayertexter application. This package contains all of the main
application logic such as the sign up process, prayer request process, and prayer confirmation process. This package
is the starting point for all received text messages and decides what to do with the message based on message content
and sender member status.
*/
package prayertexter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"unicode"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/object"
	"github.com/4JesusApps/prayertexter/internal/utility"
)

const (
	preStage            = "PRE"
	blockedUserStage    = "BLOCKED USER"
	addBlockedUserStage = "ADD BLOCKED USER"
	helpStage           = "HELP"
	memberDeleteStage   = "MEMBER DELETE"
	signUpStage         = "SIGN UP"
	dropMessageStage    = "DROP MESSAGE"
	completePrayerStage = "COMPLETE PRAYER"
	prayerRequestStage  = "PRAYER REQUEST"
	postStage           = "POST"
	stageErrPrefix      = "failure during stage "
)

// profanityExceptions are words removed from the default profanity filter because it is too sensitive.
var profanityExceptions = []string{"jerk", "ass", "butt"} //nolint:gochecknoglobals // constant-like configuration

// MainFlow is the start of the prayertexter application. It receives a text message as a parameter and based on the
// message content and sender phone number, it decides what operations to perform.
func MainFlow(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, msg messaging.TextMessage) error {
	cfg := config.Load()

	profanityChecker := messaging.NewProfanityChecker(profanityExceptions)

	mem := object.Member{Phone: msg.Phone}
	if err := mem.Get(ctx, ddbClnt); err != nil {
		return utility.LogAndWrapError(ctx, err, stageErrPrefix+preStage, "phone", msg.Phone, "msg", msg.Body)
	}

	blockedPhones := object.BlockedPhones{}
	if err := blockedPhones.Get(ctx, ddbClnt); err != nil {
		return utility.LogAndWrapError(ctx, err, stageErrPrefix+preStage, "phone", msg.Phone, "msg", msg.Body)
	}
	var isBlocked bool
	if slices.Contains(blockedPhones.Phones, mem.Phone) {
		isBlocked = true
	}

	var stageErr error
	cleanMsg := cleanStr(msg.Body)

	switch {
	// BLOCKED USER STAGE
	// Drops messages of all blocked users.
	case isBlocked:
		slog.WarnContext(ctx, "blocked user dropping message", "phone", mem.Phone, "msg", msg.Body)
		stageErr = executeStage(ctx, mem, msg, blockedUserStage, func() error {
			return nil
		})

	// ADD BLOCKED USER STAGE
	// Adds a user to the blocked users list.
	case strings.Contains(strings.ToLower(msg.Body), "#block"):
		stageErr = executeStage(ctx, mem, msg, addBlockedUserStage, func() error {
			return blockUser(ctx, ddbClnt, smsClnt, msg, mem, blockedPhones)
		})

	// HELP STAGE
	// Responds with contact info and is a requirement for the 10DLC phone number provider to get sent to to anyone
	// regardless whether they are a member or not.
	case cleanMsg == "help":
		stageErr = executeStage(ctx, mem, msg, helpStage, func() error {
			return mem.SendMessage(ctx, smsClnt, messaging.MsgHelp)
		})

	// MEMBER DELETE STAGE
	// Removes member from prayertexter.
	case cleanMsg == "cancel" || cleanMsg == "stop":
		stageErr = executeStage(ctx, mem, msg, memberDeleteStage, func() error {
			return memberDelete(ctx, ddbClnt, smsClnt, mem)
		})

	// SIGN UP STAGE
	// Initial member sign up process.
	case cleanMsg == "pray" || mem.SetupStatus == object.MemberSetupInProgress:
		stageErr = executeStage(ctx, mem, msg, signUpStage, func() error {
			return signUp(ctx, ddbClnt, smsClnt, msg, mem, profanityChecker) //nolint:wrapcheck // wrapped by executeStage
		})

	// DROP MESSAGE STAGE
	// Drops all messages if they do not meet any of the previous criteria. This serves as a catch all to drop any
	// messages of non members (other than help and sign up messages).
	case mem.SetupStatus == "":
		slog.WarnContext(ctx, "non registered user dropping message", "phone", mem.Phone, "msg", msg.Body)
		stageErr = executeStage(ctx, mem, msg, dropMessageStage, func() error {
			return nil
		})

	// COMPLETE PRAYER STAGE
	// Intercessors confirm that they prayed for a prayer, a confirmations is sent out to the prayer requestor, and the
	// prayer is marked as completed.
	case cleanMsg == "prayed":
		stageErr = executeStage(ctx, mem, msg, completePrayerStage, func() error {
			return completePrayer(ctx, ddbClnt, smsClnt, mem)
		})

	// PRAYER REQUEST STAGE
	// Assigns a prayer request to intercessors.
	case mem.SetupStatus == object.MemberSetupComplete:
		stageErr = executeStage(ctx, mem, msg, prayerRequestStage, func() error {
			return prayerRequest(ctx, ddbClnt, smsClnt, msg, mem, profanityChecker, cfg.Prayer.IntercessorsPerPrayer)
		})

	// This should never happen and if it does then it is a bug.
	default:
		err := errors.New("unexpected text message input/member status")
		return utility.LogAndWrapError(ctx, err, "could not satisfy any required conditions", "phone", mem.Phone, "msg",
			msg.Body)
	}

	if stageErr != nil {
		return stageErr
	}

	return nil
}

func cleanStr(str string) string {
	var sb strings.Builder
	sb.Grow(len(str))
	for _, ch := range str {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			sb.WriteRune(unicode.ToLower(ch))
		}
	}

	return sb.String()
}

func executeStage(ctx context.Context, mem object.Member, msg messaging.TextMessage, stageName string, stageFunc func() error) error {
	slog.InfoContext(ctx, fmt.Sprintf("Starting stage: %s", stageName), "phone", mem.Phone, "message", msg.Body)
	if stageErr := stageFunc(); stageErr != nil {
		return utility.LogAndWrapError(ctx, stageErr, stageErrPrefix+stageName, "phone", mem.Phone, "msg", msg.Body)
	}

	return nil
}
