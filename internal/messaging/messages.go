package messaging

// Default values for configuration that has been exposed to be used with the config package.
const (
	DefaultPhonePool    = "dummy"
	PhonePoolConfigPath = "conf.aws.sms.phonepool"

	DefaultTimeout    = 60
	TimeoutConfigPath = "conf.aws.sms.timeout"
)

// Sign up stage messages.
const (
	MsgNameRequest = "Reply with your name, or 2 to stay anonymous."
	MsgInvalidName = "Sorry, that name is not valid. Please reply with a name that is at least 2 letters long and " +
		"only contains letters or spaces."
	MsgMemberTypeRequest = "Reply 1 to send prayer request, or 2 to be added to the intercessors list (to pray for " +
		"others). 2 will also allow you to send in prayer requests."
	MsgPrayerInstructions = "You are now signed up to send prayer requests! You can text them directly to this number" +
		" at any time."
	MsgPrayerNumRequest = "Reply with the number of maximum prayer texts that you are willing to receive and pray for " +
		"each week."
	MsgIntercessorInstructions = "You are now signed up to receive prayer requests. Please try to pray for the " +
		"requests as soon as you receive them. " + MsgPrayed
	MsgWrongInput         = "Incorrect input received during sign up, please try again."
	MsgSignUpConfirmation = "You have opted into PrayerTexter. Msg & data rates may apply."
	MsgRemoveUser         = "You have been removed from PrayerTexter. To sign back up, text the word pray to this " +
		"number."
)

// Prayer request stage messages.
const (
	MsgInvalidRequest = "Sorry, that request is not valid. Prayer requests must contain at least 5 words."
	MsgPrayerIntro    = "Hello! Please pray for PLACEHOLDER:\n\n"
	MsgPrayed         = "Once you have prayed, reply with the word prayed so that the prayer can be confirmed."
	MsgPrayerQueued   = "We could not find any available intercessors. Your prayer has been added to the queue and " +
		"will get sent out as soon as someone is available."
	MsgPrayerAssigned = "Your prayer request has been sent out and assigned!"
	MsgPrayerReminder = "This is a friendly reminder to pray for PLACEHOLDER:\n\n"
)

// Prayer completion stage messages.
const (
	MsgNoActivePrayer     = "You have no active prayers to mark as prayed."
	MsgPrayerThankYou     = "Thank you for praying! We let the prayer requestor know that you have prayed for them."
	MsgPrayerConfirmation = "You're prayer request has been prayed for by PLACEHOLDER."
)

// Block user stage messages.
const (
	MsgUnauthorized        = "You are unauthorized to perform this action."
	MsgInvalidPhone        = "The phone number provided is invalid. Please use this format: 123-456-7890."
	MsgUserAlreadyBlocked  = "The phone number provided already exists on the block list."
	MsgSuccessfullyBlocked = "The phone number provided has been successfully added to the block list."
	MsgBlockedNotification = "You have been blocked from using PrayerTexter. If you feel this is an error, feel free " +
		"to reach out to us. "
)

// Other (general) message content sent by prayertexter.
const (
	MsgProfanityDetected = "There was profanity found in your message:\n\nPLACEHOLDER\n\nPlease try again"
	MsgHelp              = "To receive support, please email info@4jesusministries.com or call/text (949) 313-4375. " +
		"Thank you!"
	MsgPre  = "PrayerTexter: "
	MsgPost = "Reply HELP for help or STOP to cancel."
)
