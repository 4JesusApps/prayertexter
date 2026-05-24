//go:build integration

package integration_test

import (
	"testing"

	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
)

// TestSignup_PrayerOnly walks a new user through the full prayer-only sign-up
// path (stage 1 -> 2 -> 99) and verifies persisted state and SMS at every step.
func TestSignup_PrayerOnly(t *testing.T) {
	env := newTestEnv(t)
	const phone = "+11000000001"

	env.handle(msg(phone, "pray"))
	env.sender.AssertSentTo(t, phone, messaging.MsgNameRequest)
	mem := env.requireMember(phone)
	if mem.SetupStage != domain.MemberSignUpStepOne {
		t.Fatalf("after 'pray': expected stage 1, got %d", mem.SetupStage)
	}
	if mem.SetupStatus != domain.MemberSetupInProgress {
		t.Fatalf("after 'pray': expected status IN PROGRESS, got %q", mem.SetupStatus)
	}

	env.handle(msg(phone, "John Smith"))
	env.sender.AssertSentTo(t, phone, messaging.MsgMemberTypeRequest)
	mem = env.requireMember(phone)
	if mem.Name != "John Smith" {
		t.Fatalf("expected name John Smith, got %q", mem.Name)
	}
	if mem.SetupStage != domain.MemberSignUpStepTwo {
		t.Fatalf("after name: expected stage 2, got %d", mem.SetupStage)
	}

	env.handle(msg(phone, "1"))
	env.sender.AssertSentTo(t, phone, messaging.MsgPrayerInstructions)
	env.sender.AssertSentTo(t, phone, messaging.MsgSignUpConfirmation)
	mem = env.requireMember(phone)
	if mem.SetupStage != domain.MemberSignUpStepFinal {
		t.Fatalf("expected final stage 99, got %d", mem.SetupStage)
	}
	if mem.SetupStatus != domain.MemberSetupComplete {
		t.Fatalf("expected status COMPLETE, got %q", mem.SetupStatus)
	}
	if mem.Intercessor {
		t.Fatal("prayer-only user must not be flagged as intercessor")
	}
}

// TestSignup_Intercessor walks a user through the intercessor sign-up branch
// (stage 1 -> 2 -> 3 -> 99) and asserts the phone is added to IntercessorPhones.
func TestSignup_Intercessor(t *testing.T) {
	env := newTestEnv(t)
	const phone = "+11000000002"

	env.handle(msg(phone, "pray"))
	env.handle(msg(phone, "Jane Doe"))
	env.handle(msg(phone, "2"))
	env.sender.AssertSentTo(t, phone, messaging.MsgPrayerNumRequest)
	mem := env.requireMember(phone)
	if mem.SetupStage != domain.MemberSignUpStepThree {
		t.Fatalf("expected stage 3, got %d", mem.SetupStage)
	}
	if !mem.Intercessor {
		t.Fatal("expected Intercessor=true after selecting '2'")
	}

	env.handle(msg(phone, "5"))
	env.sender.AssertSentTo(t, phone, messaging.MsgIntercessorInstructions)
	env.sender.AssertSentTo(t, phone, messaging.MsgSignUpConfirmation)
	mem = env.requireMember(phone)
	if mem.SetupStage != domain.MemberSignUpStepFinal {
		t.Fatalf("expected final stage 99, got %d", mem.SetupStage)
	}
	if mem.WeeklyPrayerLimit != 5 {
		t.Fatalf("expected WeeklyPrayerLimit=5, got %d", mem.WeeklyPrayerLimit)
	}

	phones, err := env.intercessors.Get(env.ctx)
	if err != nil {
		t.Fatalf("get intercessor list: %v", err)
	}
	found := false
	for _, p := range phones.Phones {
		if p == phone {
			found = true
		}
	}
	if !found {
		t.Fatalf("intercessor phone %s not added to IntercessorPhones list: %+v", phone, phones.Phones)
	}
}

// TestSignup_AnonymousName covers the "reply 2 to stay anonymous" branch.
func TestSignup_AnonymousName(t *testing.T) {
	env := newTestEnv(t)
	const phone = "+11000000003"

	env.handle(msg(phone, "pray"))
	env.handle(msg(phone, "2")) // chose anonymous at name step
	env.sender.AssertSentTo(t, phone, messaging.MsgMemberTypeRequest)
	mem := env.requireMember(phone)
	if mem.Name != "Anonymous" {
		t.Fatalf("expected name Anonymous, got %q", mem.Name)
	}
}

// TestSignup_InvalidNameKeepsUserAtStageOne verifies an invalid name (digits
// only) bounces with MsgInvalidName and does not advance the user to stage 2.
func TestSignup_InvalidNameKeepsUserAtStageOne(t *testing.T) {
	env := newTestEnv(t)
	const phone = "+11000000004"

	env.handle(msg(phone, "pray"))
	env.handle(msg(phone, "12345"))
	env.sender.AssertSentTo(t, phone, messaging.MsgInvalidName)
	mem := env.requireMember(phone)
	if mem.SetupStage != domain.MemberSignUpStepOne {
		t.Fatalf("expected to remain at stage 1 after invalid name, got %d", mem.SetupStage)
	}

	// Sending a valid name on retry must still work.
	env.handle(msg(phone, "Valid Name"))
	env.sender.AssertSentTo(t, phone, messaging.MsgMemberTypeRequest)
}

// TestSignup_InvalidPrayerLimit verifies that an intercessor entering 0 (or
// negative) at the prayer-limit step gets MsgInvalidPrayerLimit and stays at
// stage 3.
func TestSignup_InvalidPrayerLimit(t *testing.T) {
	env := newTestEnv(t)
	const phone = "+11000000005"

	env.handle(msg(phone, "pray"))
	env.handle(msg(phone, "Alice"))
	env.handle(msg(phone, "2"))
	env.handle(msg(phone, "0"))
	env.sender.AssertSentTo(t, phone, messaging.MsgInvalidPrayerLimit)
	mem := env.requireMember(phone)
	if mem.SetupStage != domain.MemberSignUpStepThree {
		t.Fatalf("expected to remain at stage 3 after invalid limit, got %d", mem.SetupStage)
	}
}

// TestSignup_WrongInputAtTypeStep verifies that sending anything other than 1
// or 2 at the member-type step routes to signUpWrongInput.
func TestSignup_WrongInputAtTypeStep(t *testing.T) {
	env := newTestEnv(t)
	const phone = "+11000000006"

	env.handle(msg(phone, "pray"))
	env.handle(msg(phone, "Bob"))
	env.handle(msg(phone, "9"))
	env.sender.AssertSentTo(t, phone, messaging.MsgWrongInput)
	mem := env.requireMember(phone)
	if mem.SetupStage != domain.MemberSignUpStepTwo {
		t.Fatalf("expected to remain at stage 2 after wrong input, got %d", mem.SetupStage)
	}
}

// TestSignup_ProfaneNameRejected verifies the profanity filter intercepts
// names containing flagged words during sign-up.
func TestSignup_ProfaneNameRejected(t *testing.T) {
	env := newTestEnv(t)
	const phone = "+11000000007"

	env.handle(msg(phone, "pray"))
	env.handle(msg(phone, "fuck this"))
	// The profanity template renders MsgPre + "..." + word + "..." + MsgPost;
	// asserting on the literal flagged word is the most stable check.
	env.sender.AssertSentTo(t, phone, "fuck")
	mem := env.requireMember(phone)
	if mem.SetupStage != domain.MemberSignUpStepOne {
		t.Fatalf("expected to remain at stage 1 after profane name, got %d", mem.SetupStage)
	}
}

// TestSignup_CapitalPrayStarts verifies "Pray" with a capital P (and any other
// case variation) starts the sign-up flow. cleanStr lowercases letters before
// matching, so the trigger is case-insensitive.
func TestSignup_CapitalPrayStarts(t *testing.T) {
	env := newTestEnv(t)
	const phone = "+11000000008"

	env.handle(msg(phone, "Pray"))
	env.sender.AssertSentTo(t, phone, messaging.MsgNameRequest)
	mem := env.requireMember(phone)
	if mem.SetupStage != domain.MemberSignUpStepOne {
		t.Fatalf("expected stage 1 after 'Pray', got %d", mem.SetupStage)
	}
	if mem.SetupStatus != domain.MemberSetupInProgress {
		t.Fatalf("expected IN PROGRESS, got %q", mem.SetupStatus)
	}
}

// TestSignup_OneLetterNameRejected covers the boundary case of isNameValid:
// a name with letters but fewer than the minimum (2) is invalid.
func TestSignup_OneLetterNameRejected(t *testing.T) {
	env := newTestEnv(t)
	const phone = "+11000000009"

	env.handle(msg(phone, "pray"))
	env.handle(msg(phone, "A"))
	env.sender.AssertSentTo(t, phone, messaging.MsgInvalidName)
	mem := env.requireMember(phone)
	if mem.SetupStage != domain.MemberSignUpStepOne {
		t.Fatalf("expected to stay at stage 1 after 1-letter name, got %d", mem.SetupStage)
	}
}

// TestSignup_PrayerLimitWithComma verifies that cleanStr strips non-alphanumeric
// characters from the intercessor prayer-limit input. "5," should still parse
// as 5.
func TestSignup_PrayerLimitWithComma(t *testing.T) {
	env := newTestEnv(t)
	const phone = "+11000000010"

	env.handle(msg(phone, "pray"))
	env.handle(msg(phone, "Carol"))
	env.handle(msg(phone, "2"))
	env.handle(msg(phone, "5,")) // comma should be stripped by cleanStr

	env.sender.AssertSentTo(t, phone, messaging.MsgSignUpConfirmation)
	mem := env.requireMember(phone)
	if mem.WeeklyPrayerLimit != 5 {
		t.Fatalf("expected WeeklyPrayerLimit=5 after '5,', got %d", mem.WeeklyPrayerLimit)
	}
}

// TestSignup_NonNumericPrayerLimit verifies that a non-numeric prayer-limit
// reply at stage 3 falls through to signUpWrongInput.
func TestSignup_NonNumericPrayerLimit(t *testing.T) {
	env := newTestEnv(t)
	const phone = "+11000000011"

	env.handle(msg(phone, "pray"))
	env.handle(msg(phone, "Dave"))
	env.handle(msg(phone, "2"))
	env.handle(msg(phone, "abc"))
	env.sender.AssertSentTo(t, phone, messaging.MsgWrongInput)
	mem := env.requireMember(phone)
	if mem.SetupStage != domain.MemberSignUpStepThree {
		t.Fatalf("expected to stay at stage 3 after non-numeric limit, got %d", mem.SetupStage)
	}
}
