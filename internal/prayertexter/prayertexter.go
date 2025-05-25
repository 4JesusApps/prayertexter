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
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/object"
	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/spf13/viper"
)

const (
	preStage            = "PRE"
	helpStage           = "HELP"
	memberDeleteStage   = "MEMBER DELETE"
	signUpStage         = "SIGN UP"
	dropMessageStage    = "DROP MESSAGE"
	completePrayerStage = "COMPLETE PRAYER"
	prayerRequestStage  = "PRAYER REQUEST"
	postStage           = "POST"
	stageErrPrefix      = "failure during stage "
)

// MainFlow is the start of the prayertexter application. It receives a text message as a parameter and based on the
// message content and sender phone number, it decides what operations to perform.
func MainFlow(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, msg messaging.TextMessage) error {
	config.InitConfig()

	currTime := time.Now().Format(time.RFC3339)
	id, err := utility.GenerateID()
	if err != nil {
		return utility.LogAndWrapError(ctx, err, stageErrPrefix+preStage, "phone", msg.Phone, "msg", msg.Body)
	}

	state := object.State{}
	state.Status = object.StateInProgress
	state.TimeStart = currTime
	state.ID = id
	state.Message = msg

	if err = state.Update(ctx, ddbClnt, false); err != nil {
		return utility.LogAndWrapError(ctx, err, stageErrPrefix+preStage, "phone", msg.Phone, "msg", msg.Body)
	}

	mem := object.Member{Phone: msg.Phone}
	if err = mem.Get(ctx, ddbClnt); err != nil {
		return utility.LogAndWrapError(ctx, err, stageErrPrefix+preStage, "phone", msg.Phone, "msg", msg.Body)
	}

	var stageErr error
	cleanMsg := cleanStr(msg.Body)

	switch {
	// HELP STAGE
	// Responds with contact info and is a requirement for the 10DLC phone number provider to get sent to to anyone
	// regardless whether they are a member or not.
	case cleanMsg == "help":
		stageErr = executeStage(ctx, ddbClnt, helpStage, &state, func() error {
			return mem.SendMessage(ctx, smsClnt, messaging.MsgHelp)
		})

	// MEMBER DELETE STAGE
	// Removes member from prayertexter.
	case cleanMsg == "cancel" || cleanMsg == "stop":
		stageErr = executeStage(ctx, ddbClnt, memberDeleteStage, &state, func() error {
			return memberDelete(ctx, ddbClnt, smsClnt, mem)
		})

	// SIGN UP STAGE
	// Initial member sign up process.
	case cleanMsg == "pray" || mem.SetupStatus == object.MemberSetupInProgress:
		stageErr = executeStage(ctx, ddbClnt, signUpStage, &state, func() error {
			return signUp(ctx, ddbClnt, smsClnt, msg, mem)
		})

	// DROP MESSAGE STAGE
	// Drops all messages if they do not meet any of the previous criteria. This serves as a catch all to drop any
	// messages of non members (other than help and sign up messages).
	case mem.SetupStatus == "":
		slog.WarnContext(ctx, "non registered user dropping message", "phone", mem.Phone, "msg", msg.Body)
		stageErr = executeStage(ctx, ddbClnt, dropMessageStage, &state, func() error {
			return nil
		})

	// COMPLETE PRAYER STAGE
	// Intercessors confirm that they prayed for a prayer, a confirmations is sent out to the prayer requestor, and the
	// prayer is marked as completed.
	case cleanMsg == "prayed":
		stageErr = executeStage(ctx, ddbClnt, completePrayerStage, &state, func() error {
			return completePrayer(ctx, ddbClnt, smsClnt, mem)
		})

	// PRAYER REQUEST STAGE
	// Assigns a prayer request to intercessors.
	case mem.SetupStatus == object.MemberSetupComplete:
		stageErr = executeStage(ctx, ddbClnt, prayerRequestStage, &state, func() error {
			return prayerRequest(ctx, ddbClnt, smsClnt, msg, mem)
		})

	// This should never happen and if it does then it is a bug.
	default:
		err = errors.New("unexpected text message input/member status")
		return utility.LogAndWrapError(ctx, err, "could not satisfy any required conditions", "phone", mem.Phone, "msg",
			msg.Body)
	}

	if stageErr != nil {
		return stageErr
	}

	if err = state.Update(ctx, ddbClnt, true); err != nil {
		return utility.LogAndWrapError(ctx, err, stageErrPrefix+postStage, "phone", mem.Phone, "msg", msg.Body)
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

func executeStage(ctx context.Context, ddbClnt db.DDBConnecter, stageName string, state *object.State, stageFunc func() error) error {
	state.Stage = stageName
	if err := state.Update(ctx, ddbClnt, false); err != nil {
		return utility.LogAndWrapError(ctx, err, stageErrPrefix+stageName, "phone", state.Message.Phone, "msg",
			state.Message.Body)
	}

	if stageErr := stageFunc(); stageErr != nil {
		state.Error = stageErr.Error()
		state.Status = object.StateFailed
		if updateErr := state.Update(ctx, ddbClnt, false); updateErr != nil {
			return utility.LogAndWrapError(ctx, updateErr, stageErrPrefix+stageName, "stage error", stageErr.Error(),
				"phone", state.Message.Phone, "msg", state.Message.Body)
		}

		return utility.LogAndWrapError(ctx, stageErr, stageErrPrefix+stageName, "phone", state.Message.Phone,
			"msg", state.Message.Body)
	}

	return nil
}

func signUp(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, msg messaging.TextMessage, mem object.Member) error {
	cleanMsg := cleanStr(msg.Body)

	switch {
	case cleanMsg == "pray":
		if err := signUpStageOne(ctx, ddbClnt, smsClnt, mem); err != nil {
			return utility.WrapError(err, "failed sign up stage 1")
		}
	case mem.SetupStage == object.MemberSignUpStepOne:
		if err := signUpStageTwo(ctx, ddbClnt, smsClnt, mem, msg); err != nil {
			return utility.WrapError(err, "failed sign up stage 2")
		}
	case cleanMsg == "1" && mem.SetupStage == object.MemberSignUpStepTwo:
		if err := signUpFinalPrayerMessage(ctx, ddbClnt, smsClnt, mem); err != nil {
			return utility.WrapError(err, "failed sign up final prayer message")
		}
	case cleanMsg == "2" && mem.SetupStage == object.MemberSignUpStepTwo:
		if err := signUpStageThree(ctx, ddbClnt, smsClnt, mem); err != nil {
			return utility.WrapError(err, "failed sign up stage 3")
		}
	case mem.SetupStage == object.MemberSignUpStepThree:
		if err := signUpFinalIntercessorMessage(ctx, ddbClnt, smsClnt, mem, msg); err != nil {
			return utility.WrapError(err, "failed sign up final intercessor message")
		}
	default:
		if err := signUpWrongInput(ctx, smsClnt, mem, msg); err != nil {
			return utility.WrapError(err, "failed sign up wrong input")
		}
	}

	return nil
}

func signUpStageOne(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, mem object.Member) error {
	mem.SetupStatus = object.MemberSetupInProgress
	mem.SetupStage = object.MemberSignUpStepOne
	if err := mem.Put(ctx, ddbClnt); err != nil {
		return err
	}

	return mem.SendMessage(ctx, smsClnt, messaging.MsgNameRequest)
}

func signUpStageTwo(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, mem object.Member, msg messaging.TextMessage) error {
	hasProfanity, err := checkIfProfanity(ctx, smsClnt, mem, msg)
	if err != nil {
		return err
	} else if hasProfanity {
		return nil
	}

	if cleanStr(msg.Body) == "2" {
		mem.Name = "Anonymous"
	} else {
		mem.Name = msg.Body
	}

	isValid, err := checkIfNameValid(ctx, smsClnt, mem)
	if err != nil {
		return err
	} else if !isValid {
		return nil
	}

	mem.SetupStage = object.MemberSignUpStepTwo

	if err = mem.Put(ctx, ddbClnt); err != nil {
		return err
	}

	return mem.SendMessage(ctx, smsClnt, messaging.MsgMemberTypeRequest)
}

// checkIfProfanity reports whether there is profanity in a text message. If there is, it will inform the sender.
func checkIfProfanity(ctx context.Context, smsClnt messaging.TextSender, mem object.Member, msg messaging.TextMessage) (bool, error) {
	profanity := msg.CheckProfanity()
	if profanity != "" {
		msg := strings.Replace(messaging.MsgProfanityDetected, "PLACEHOLDER", profanity, 1)
		if err := mem.SendMessage(ctx, smsClnt, msg); err != nil {
			return true, err
		}

		return true, nil
	}

	return false, nil
}

// checkIfNameValid reports whether a name is valid. A valid name is at least 2 characters long and does not contain any
// numbers. If name is invalid, it will inform the sender.
func checkIfNameValid(ctx context.Context, smsClnt messaging.TextSender, mem object.Member) (bool, error) {
	letterCount := 0
	minLetters := 2
	isValid := true

	for _, ch := range mem.Name {
		switch {
		case unicode.IsLetter(ch):
			letterCount++
		case ch == ' ':
			// Do nothing; spaces are fine but don't count toward letters.
		default:
			isValid = false
		}
	}

	if letterCount < minLetters {
		isValid = false
	}

	if !isValid {
		if err := mem.SendMessage(ctx, smsClnt, messaging.MsgInvalidName); err != nil {
			return isValid, err
		}

		return isValid, nil
	}

	return isValid, nil
}

func signUpFinalPrayerMessage(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, mem object.Member) error {
	mem.SetupStatus = object.MemberSetupComplete
	mem.SetupStage = object.MemberSignUpStepFinal
	mem.Intercessor = false
	if err := mem.Put(ctx, ddbClnt); err != nil {
		return err
	}

	body := messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgSignUpConfirmation

	return mem.SendMessage(ctx, smsClnt, body)
}

func signUpStageThree(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, mem object.Member) error {
	mem.SetupStage = object.MemberSignUpStepThree
	mem.Intercessor = true
	if err := mem.Put(ctx, ddbClnt); err != nil {
		return err
	}

	return mem.SendMessage(ctx, smsClnt, messaging.MsgPrayerNumRequest)
}

func signUpFinalIntercessorMessage(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, mem object.Member, msg messaging.TextMessage) error {
	num, err := strconv.Atoi(cleanStr(msg.Body))
	if err != nil {
		return signUpWrongInput(ctx, smsClnt, mem, msg)
	}

	phones := object.IntercessorPhones{}
	if err = phones.Get(ctx, ddbClnt); err != nil {
		return err
	}

	phones.AddPhone(mem.Phone)
	if err = phones.Put(ctx, ddbClnt); err != nil {
		return err
	}

	mem.SetupStatus = object.MemberSetupComplete
	mem.SetupStage = object.MemberSignUpStepFinal
	mem.WeeklyPrayerLimit = num
	mem.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
	if err = mem.Put(ctx, ddbClnt); err != nil {
		return err
	}

	body := messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgIntercessorInstructions + "\n\n" +
		messaging.MsgSignUpConfirmation

	return mem.SendMessage(ctx, smsClnt, body)
}

func signUpWrongInput(ctx context.Context, smsClnt messaging.TextSender, mem object.Member, msg messaging.TextMessage) error {
	slog.WarnContext(ctx, "wrong input received during sign up", "member", mem.Phone, "msg", msg)

	return mem.SendMessage(ctx, smsClnt, messaging.MsgWrongInput)
}

func memberDelete(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, mem object.Member) error {
	if err := mem.Delete(ctx, ddbClnt); err != nil {
		return err
	}
	if mem.Intercessor {
		if err := removeIntercessor(ctx, ddbClnt, mem); err != nil {
			return err
		}
	}

	return mem.SendMessage(ctx, smsClnt, messaging.MsgRemoveUser)
}

func removeIntercessor(ctx context.Context, ddbClnt db.DDBConnecter, mem object.Member) error {
	phones := object.IntercessorPhones{}
	if err := phones.Get(ctx, ddbClnt); err != nil {
		return err
	}
	phones.RemovePhone(mem.Phone)
	if err := phones.Put(ctx, ddbClnt); err != nil {
		return err
	}

	// Moves an active prayer from the intercessor being removed to the Prayer queue. This is done to ensure that the
	// prayer eventually gets assigned to another intercessor.
	return moveActivePrayer(ctx, ddbClnt, mem)
}

func moveActivePrayer(ctx context.Context, ddbClnt db.DDBConnecter, mem object.Member) error {
	isActive, err := object.IsPrayerActive(ctx, ddbClnt, mem.Phone)
	if err != nil {
		return err
	}

	if isActive {
		pryr := object.Prayer{IntercessorPhone: mem.Phone}
		if err = pryr.Get(ctx, ddbClnt, false); err != nil {
			return err
		}

		if err = pryr.Delete(ctx, ddbClnt, false); err != nil {
			return err
		}

		// A random ID is generated since queued Prayers do not have an intercessor assigned to them. We use the ID in
		// place if the intercessors phone number until there is an available intercessor, at which time the ID will get
		// changed to the available intercessors phone number.
		var id string
		id, err = utility.GenerateID()
		if err != nil {
			return err
		}
		pryr.IntercessorPhone = id
		pryr.Intercessor = object.Member{}

		return pryr.Put(ctx, ddbClnt, true)
	}

	return nil
}

func prayerRequest(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, msg messaging.TextMessage, mem object.Member) error {
	hasProfanity, err := checkIfProfanity(ctx, smsClnt, mem, msg)
	if err != nil {
		return err
	} else if hasProfanity {
		return nil
	}

	// Check for #anon anywhere in the message, handle anonymous request.
	requestBody := msg.Body
	if strings.Contains(strings.ToLower(requestBody), "#anon") {
		mem.Name = "Anonymous"
		requestBody = strings.TrimSpace(strings.ReplaceAll(requestBody, "#anon", ""))
	}

	isValid, err := checkIfRequestValid(ctx, smsClnt, messaging.TextMessage{Phone: msg.Phone, Body: requestBody}, mem)
	if err != nil {
		return err
	} else if !isValid {
		return nil
	}

	intercessors, err := FindIntercessors(ctx, ddbClnt, mem.Phone)
	if err != nil && errors.Is(err, utility.ErrNoAvailableIntercessors) {
		slog.WarnContext(ctx, "no intercessors available", "request", requestBody, "requestor", msg.Phone)
		if err = queuePrayer(ctx, ddbClnt, smsClnt, messaging.TextMessage{Phone: msg.Phone, Body: requestBody}, mem); err != nil {
			return utility.WrapError(err, "failed to queue prayer")
		}
		return nil
	} else if err != nil {
		return utility.WrapError(err, "failed to find intercessors")
	}

	for _, intr := range intercessors {
		pryr := object.Prayer{
			Request:   requestBody,
			Requestor: mem,
		}

		if err = AssignPrayer(ctx, ddbClnt, smsClnt, pryr, intr); err != nil {
			return err
		}
	}

	return mem.SendMessage(ctx, smsClnt, messaging.MsgPrayerAssigned)
}

func checkIfRequestValid(ctx context.Context, smsClnt messaging.TextSender, msg messaging.TextMessage, mem object.Member) (bool, error) {
	minWords := 5
	if len(strings.Fields(msg.Body)) < minWords {
		if err := mem.SendMessage(ctx, smsClnt, messaging.MsgInvalidRequest); err != nil {
			return false, err
		}

		return false, nil
	}

	return true, nil
}

// AssignPrayer will save a prayer object to the dynamodb active prayers table with a newly assigned intercessor. It
// will also send the intercessor a text message with the newly assigned prayer request.
func AssignPrayer(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, pryr object.Prayer, intr object.Member) error {
	pryr.Intercessor = intr
	pryr.IntercessorPhone = intr.Phone
	if err := pryr.Put(ctx, ddbClnt, false); err != nil {
		return err
	}

	msg := strings.Replace(messaging.MsgPrayerIntro, "PLACEHOLDER", pryr.Requestor.Name, 1)
	msg = msg + pryr.Request + "\n\n" + messaging.MsgPrayed
	err := pryr.Intercessor.SendMessage(ctx, smsClnt, msg)
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "assigned prayer successfully")
	return nil
}

// FindIntercessors returns a slice of Member intercessors that are available to be assigned a prayer request. If there
// are no available intercessors, it will return an error.
func FindIntercessors(ctx context.Context, ddbClnt db.DDBConnecter, skipPhone string) ([]object.Member, error) {
	allPhones, err := getAndPreparePhones(ctx, ddbClnt, skipPhone)
	if err != nil {
		return nil, err
	}

	var intercessors []object.Member
	intercessorsPerPrayer := viper.GetInt(object.IntercessorsPerPrayerConfigPath)

	for len(intercessors) < intercessorsPerPrayer {
		randPhones := allPhones.GenRandPhones()
		if randPhones == nil {
			slog.InfoContext(ctx, "there are no more intercessors left to check")
			if len(intercessors) > 0 {
				slog.InfoContext(ctx, "there is at least one intercessor found, returning this even though it is less "+
					"than the desired number of intercessors per prayer")
				return intercessors, nil
			}

			// There are not any intercessors available at all.
			return nil, utility.ErrNoAvailableIntercessors
		}

		for _, phn := range randPhones {
			// Check if we've already reached the desired number of intercessors.
			if len(intercessors) >= intercessorsPerPrayer {
				return intercessors, nil
			}

			var intr *object.Member
			intr, err = processIntercessor(ctx, ddbClnt, phn)
			if err != nil && errors.Is(err, utility.ErrIntercessorUnavailable) {
				allPhones.RemovePhone(phn)
				continue
			} else if err != nil {
				return nil, err
			}

			intercessors = append(intercessors, *intr)
			allPhones.RemovePhone(phn)
			slog.InfoContext(ctx, "found one available intercessor")
		}
	}

	return intercessors, nil
}

func getAndPreparePhones(ctx context.Context, ddbClnt db.DDBConnecter, skipPhone string) (object.IntercessorPhones, error) {
	phones := object.IntercessorPhones{}
	if err := phones.Get(ctx, ddbClnt); err != nil {
		return phones, err
	}

	// Removes the prayer requestors phone number from IntercessorPhones so that they do not get assigned to pray for
	// their own prayer request. This could happen if the prayer requestor is also an intercessor.
	utility.RemoveItem(&phones.Phones, skipPhone)

	return phones, nil
}

func processIntercessor(ctx context.Context, ddbClnt db.DDBConnecter, phone string) (*object.Member, error) {
	intr := object.Member{Phone: phone}
	if err := intr.Get(ctx, ddbClnt); err != nil {
		return nil, err
	}

	isActive, err := object.IsPrayerActive(ctx, ddbClnt, intr.Phone)
	if err != nil {
		return nil, err
	}
	if isActive {
		// This intercessor already has 1 active prayer and is therefor unavailable. Each intercessor can only have a
		// maximum of 1 active prayer request at any given time.
		return nil, utility.ErrIntercessorUnavailable
	}

	if intr.PrayerCount < intr.WeeklyPrayerLimit {
		intr.PrayerCount++
	} else {
		var canReset bool
		canReset, err = canResetPrayerCount(intr)
		if err != nil {
			return nil, err
		}
		if canReset {
			// Reset intercessor's weekly prayer count
			intr.PrayerCount = 1
			intr.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
		} else {
			return nil, utility.ErrIntercessorUnavailable
		}
	}

	if err = intr.Put(ctx, ddbClnt); err != nil {
		return nil, err
	}
	return &intr, nil
}

func canResetPrayerCount(intr object.Member) (bool, error) {
	weekDays := 7
	dayHours := 24

	currentTime := time.Now()
	previousTime, err := time.Parse(time.RFC3339, intr.WeeklyPrayerDate)
	if err != nil {
		return false, err
	}
	diffDays := currentTime.Sub(previousTime).Hours() / float64(dayHours)
	return diffDays > float64(weekDays), nil
}

func queuePrayer(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, msg messaging.TextMessage, mem object.Member) error {
	pryr := object.Prayer{}
	// A random ID is generated since queued Prayers do not have an intercessor assigned to them. We use the ID in place
	// if the intercessors phone number until there is an available intercessor, at which time the ID will get changed
	// to the available intercessors phone number.
	id, err := utility.GenerateID()
	if err != nil {
		return err
	}

	pryr.IntercessorPhone = id
	pryr.Request = msg.Body
	pryr.Requestor = mem

	if err = pryr.Put(ctx, ddbClnt, true); err != nil {
		return err
	}

	return mem.SendMessage(ctx, smsClnt, messaging.MsgPrayerQueued)
}

func completePrayer(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, mem object.Member) error {
	pryr := object.Prayer{IntercessorPhone: mem.Phone}
	if err := pryr.Get(ctx, ddbClnt, false); err != nil {
		return err
	}

	if pryr.Request == "" {
		// Get Dynamodb calls return empty data if the key does not exist in the table. Therefor if prayer request is
		// empty here, it means that it did not exist in the database.
		if err := mem.SendMessage(ctx, smsClnt, messaging.MsgNoActivePrayer); err != nil {
			return err
		}
		return nil
	}

	if err := mem.SendMessage(ctx, smsClnt, messaging.MsgPrayerThankYou); err != nil {
		return err
	}

	msg := strings.Replace(messaging.MsgPrayerConfirmation, "PLACEHOLDER", mem.Name, 1)

	isActive, err := object.IsMemberActive(ctx, ddbClnt, pryr.Requestor.Phone)
	if err != nil {
		return err
	}

	if isActive {
		if err = pryr.Requestor.SendMessage(ctx, smsClnt, msg); err != nil {
			return err
		}
	} else {
		slog.WarnContext(ctx, "Skip sending message, member is not active", "recipient", pryr.Requestor.Phone,
			"body", msg)
	}

	return pryr.Delete(ctx, ddbClnt, false)
}
