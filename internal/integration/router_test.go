//go:build integration

package integration_test

import (
	"testing"

	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
)

// TestRouter_HelpCommand verifies an existing member sending "help" gets the
// help text. The router routes on cleanStr -> "help" regardless of case.
func TestRouter_HelpCommand(t *testing.T) {
	env := newTestEnv(t)
	const phone = "+15000000001"

	env.seedRequestor(phone, "Sam")

	env.handle(msg(phone, "HELP"))
	env.sender.AssertSentTo(t, phone, messaging.MsgHelp)
}

// TestRouter_CancelDeletesMember covers the cancel/stop branch: an existing
// member sending "cancel" (or "stop") is deleted and notified.
func TestRouter_CancelDeletesMember(t *testing.T) {
	env := newTestEnv(t)
	const phone = "+15000000002"

	env.seedRequestor(phone, "Sam")

	env.handle(msg(phone, "cancel"))
	env.sender.AssertSentTo(t, phone, messaging.MsgRemoveUser)
	env.requireNoMember(phone)
}

// TestRouter_StopDeletesMember mirrors cancel: "stop" is the SMS-standard
// opt-out keyword and the router treats it identically.
func TestRouter_StopDeletesMember(t *testing.T) {
	env := newTestEnv(t)
	const phone = "+15000000003"

	env.seedRequestor(phone, "Sam")

	env.handle(msg(phone, "STOP"))
	env.sender.AssertSentTo(t, phone, messaging.MsgRemoveUser)
	env.requireNoMember(phone)
}

// TestRouter_CancelRemovesIntercessorFromList verifies that cancelling an
// intercessor removes them from IntercessorPhones (not just the Member table).
func TestRouter_CancelRemovesIntercessorFromList(t *testing.T) {
	env := newTestEnv(t)
	const phone = "+15000000004"

	env.seedIntercessor(phone, "Ivy", 5)
	phones, _ := env.intercessors.Get(env.ctx)
	if len(phones.Phones) != 1 {
		t.Fatalf("seed sanity: expected 1 intercessor, got %d", len(phones.Phones))
	}

	env.handle(msg(phone, "cancel"))

	phones, _ = env.intercessors.Get(env.ctx)
	for _, p := range phones.Phones {
		if p == phone {
			t.Fatalf("intercessor %s still in list after cancel: %+v", phone, phones.Phones)
		}
	}
	env.requireNoMember(phone)
}

// TestRouter_BlockedUserMessageDropped verifies that any message from a
// blocked phone is silently dropped: no error, no SMS sent in response.
func TestRouter_BlockedUserMessageDropped(t *testing.T) {
	env := newTestEnv(t)
	const phone = "+15000000005"

	env.seedBlocked(phone)

	env.handle(msg(phone, "pray"))
	env.sender.AssertNotSentTo(t, phone)
	env.requireNoMember(phone) // dropped message must not create a stub member
}

// TestRouter_NonRegisteredUserNonPrayMessageDropped verifies that a phone with
// no member record sending something other than "pray"/help/cancel/etc gets
// silently dropped (per the DROP MESSAGE branch in Router.Handle).
func TestRouter_NonRegisteredUserNonPrayMessageDropped(t *testing.T) {
	env := newTestEnv(t)
	const phone = "+15000000006"

	env.handle(msg(phone, "random text"))
	env.sender.AssertNotSentTo(t, phone)
	env.requireNoMember(phone)
}

// TestRouter_HelpDuringSignUp verifies that "help" overrides the in-progress
// sign-up flow. The router checks `cleanMsg == "help"` before the SetupStatus
// branches, so a user mid-signup who texts help should get the help text
// (and remain at whatever stage they were on).
func TestRouter_HelpDuringSignUp(t *testing.T) {
	env := newTestEnv(t)
	const phone = "+15000000007"

	// Place the user at stage 1 (mid sign-up) by writing the partial state directly.
	if err := env.members.Save(env.ctx, &domain.Member{
		Phone:       phone,
		SetupStatus: domain.MemberSetupInProgress,
		SetupStage:  domain.MemberSignUpStepOne,
	}); err != nil {
		t.Fatalf("seed mid-signup member: %v", err)
	}

	env.handle(msg(phone, "help"))
	env.sender.AssertSentTo(t, phone, messaging.MsgHelp)

	// Stage must not have advanced -- help is read-only.
	mem := env.requireMember(phone)
	if mem.SetupStage != domain.MemberSignUpStepOne {
		t.Fatalf("expected stage 1 after help; got %d", mem.SetupStage)
	}
}

// TestRouter_StopMovesActivePrayerToQueue covers the headline intercessor
// opt-out flow: an intercessor with an in-flight active prayer sends STOP.
// removeIntercessor -> moveActivePrayer should re-queue the prayer with a
// fresh QueueID so it can be picked up by a different intercessor later.
func TestRouter_StopMovesActivePrayerToQueue(t *testing.T) {
	env := newTestEnv(t)
	const requestor = "+15000000008"
	const intr = "+15000000009"

	env.seedRequestor(requestor, "Sam")
	env.seedIntercessor(intr, "Ivy", 5)

	env.handle(msg(requestor, "please pray for my brother's recovery soon"))
	env.requireActivePrayer(intr)
	env.sender.Reset()

	env.handle(msg(intr, "STOP"))

	// Intercessor is fully removed.
	env.requireNoMember(intr)
	phones, err := env.intercessors.Get(env.ctx)
	if err != nil {
		t.Fatalf("get intercessor phones: %v", err)
	}
	for _, p := range phones.Phones {
		if p == intr {
			t.Fatalf("intercessor %s still in list after STOP: %+v", intr, phones.Phones)
		}
	}

	// Active prayer is gone; a fresh queued prayer carries the original requestor.
	env.requireNoActivePrayer(intr)
	queued := env.requireQueuedCount(1)
	if queued[0].Requestor.Phone != requestor {
		t.Fatalf("queued prayer Requestor.Phone=%s; expected %s", queued[0].Requestor.Phone, requestor)
	}
	if queued[0].Intercessor.Phone != "" {
		t.Fatalf("queued prayer Intercessor must be cleared; got %+v", queued[0].Intercessor)
	}

	// The departing intercessor still gets the standard removal notice.
	env.sender.AssertSentTo(t, intr, messaging.MsgRemoveUser)
}
