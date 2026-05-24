//go:build integration

package integration_test

import (
	"slices"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
)

// seedAdmin creates an Administrator member.
func (e *testEnv) seedAdmin(phone, name string) {
	e.t.Helper()
	mem := &domain.Member{
		Phone:         phone,
		Name:          name,
		Administrator: true,
		SetupStatus:   domain.MemberSetupComplete,
		SetupStage:    domain.MemberSignUpStepFinal,
	}
	if err := e.members.Save(e.ctx, mem); err != nil {
		e.t.Fatalf("seed admin %s: %v", phone, err)
	}
}

// TestAdmin_BlockExistingMember covers the headline admin flow: an
// administrator sends "#block <phone>"; the target is added to the blocked
// list, removed from members, and notified.
func TestAdmin_BlockExistingMember(t *testing.T) {
	env := newTestEnv(t)
	const admin = "+14000000001"
	const target = "+11234567890" // BlockUser prepends "+1"; body must use 10-digit form
	const targetBody = "1234567890"

	env.seedAdmin(admin, "Boss")
	env.seedRequestor(target, "Victim")

	env.handle(msg(admin, "#block "+targetBody))

	// Blocked list contains the target.
	bp, err := env.blocked.Get(env.ctx)
	if err != nil {
		t.Fatalf("get blocked: %v", err)
	}
	if !slices.Contains(bp.Phones, target) {
		t.Fatalf("expected %s in BlockedPhones, got %+v", target, bp.Phones)
	}

	// Target member has been deleted.
	env.requireNoMember(target)

	// Target was notified, admin got success confirmation.
	env.sender.AssertSentTo(t, target, messaging.MsgBlockedNotification)
	env.sender.AssertSentTo(t, admin, messaging.MsgSuccessfullyBlocked)
}

// TestAdmin_BlockUnregisteredPhone covers the case where the blocked phone is
// not a member: the phone still gets added to the block list and notified.
func TestAdmin_BlockUnregisteredPhone(t *testing.T) {
	env := newTestEnv(t)
	const admin = "+14000000002"
	const target = "+19998887777"
	const targetBody = "9998887777"

	env.seedAdmin(admin, "Boss")

	env.handle(msg(admin, "#block "+targetBody))

	bp, err := env.blocked.Get(env.ctx)
	if err != nil {
		t.Fatalf("get blocked: %v", err)
	}
	if !slices.Contains(bp.Phones, target) {
		t.Fatalf("expected %s in BlockedPhones, got %+v", target, bp.Phones)
	}
	env.sender.AssertSentTo(t, admin, messaging.MsgSuccessfullyBlocked)
}

// TestAdmin_BlockRejectedForNonAdmin verifies a non-administrator member who
// sends "#block ..." gets MsgUnauthorized and no state changes.
func TestAdmin_BlockRejectedForNonAdmin(t *testing.T) {
	env := newTestEnv(t)
	const notAdmin = "+14000000003"
	const target = "+11234567890"

	env.seedRequestor(notAdmin, "Regular") // Administrator=false

	env.handle(msg(notAdmin, "#block 1234567890"))

	env.sender.AssertSentTo(t, notAdmin, messaging.MsgUnauthorized)

	// Block list still empty.
	bp, err := env.blocked.Get(env.ctx)
	if err != nil {
		t.Fatalf("get blocked: %v", err)
	}
	if slices.Contains(bp.Phones, target) {
		t.Fatalf("unauthorized block must not have added %s to list", target)
	}
}

// TestAdmin_BlockAlreadyBlocked verifies duplicate block attempts trigger
// MsgUserAlreadyBlocked.
func TestAdmin_BlockAlreadyBlocked(t *testing.T) {
	env := newTestEnv(t)
	const admin = "+14000000004"
	const target = "+19998887777"

	env.seedAdmin(admin, "Boss")
	env.seedBlocked(target)

	env.handle(msg(admin, "#block 9998887777"))
	env.sender.AssertSentTo(t, admin, messaging.MsgUserAlreadyBlocked)
}

// TestAdmin_BlockInvalidPhone verifies that a malformed phone in the body
// triggers MsgInvalidPhone.
func TestAdmin_BlockInvalidPhone(t *testing.T) {
	env := newTestEnv(t)
	const admin = "+14000000005"

	env.seedAdmin(admin, "Boss")

	env.handle(msg(admin, "#block 12345")) // not 10 digits
	env.sender.AssertSentTo(t, admin, messaging.MsgInvalidPhone)
}

// TestAdmin_BlockActiveIntercessorRequeuesPrayer is the rollback-style admin
// scenario: blocking an intercessor who currently holds an active prayer must
// move that prayer back to the queue so a different intercessor can pick it
// up. Exercises moveActivePrayer through DeleteWithoutNotification.
func TestAdmin_BlockActiveIntercessorRequeuesPrayer(t *testing.T) {
	env := newTestEnv(t)
	const admin = "+14000000006"
	const requestor = "+14000000007"
	const intr = "+19998880000"
	const intrBody = "9998880000"

	env.seedAdmin(admin, "Boss")
	env.seedRequestor(requestor, "Sam")
	env.seedIntercessor(intr, "Ivy", 5)

	// Generate an active prayer assigned to the intercessor via the normal flow.
	env.handle(msg(requestor, "please pray for my upcoming surgery this Friday"))
	env.requireActivePrayer(intr)
	env.sender.Reset() // ignore prior SMS

	// Admin blocks the intercessor.
	env.handle(msg(admin, "#block "+intrBody))

	// Intercessor is gone from Member table and intercessor list.
	env.requireNoMember(intr)
	phones, err := env.intercessors.Get(env.ctx)
	if err != nil {
		t.Fatalf("get intercessor phones: %v", err)
	}
	for _, p := range phones.Phones {
		if p == intr {
			t.Fatalf("intercessor %s still in list after block: %+v", intr, phones.Phones)
		}
	}

	// The active prayer is gone and a new queued prayer exists with the
	// original requestor info (fresh ID, intercessor cleared).
	env.requireNoActivePrayer(intr)
	queued := env.requireQueuedCount(1)
	if queued[0].Requestor.Phone != requestor {
		t.Fatalf("expected queued prayer Requestor.Phone=%s, got %s", requestor, queued[0].Requestor.Phone)
	}
	if queued[0].Intercessor.Phone != "" {
		t.Fatalf("expected queued prayer Intercessor cleared, got %+v", queued[0].Intercessor)
	}

	// Admin + blocked intercessor each got their expected SMS.
	env.sender.AssertSentTo(t, admin, messaging.MsgSuccessfullyBlocked)
	env.sender.AssertSentTo(t, intr, messaging.MsgBlockedNotification)
}
