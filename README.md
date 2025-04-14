# prayertexter

This application is a work in progress!

PrayerTexter is a Go application that lets users submit prayer requests via text message (SMS). These requests are then assigned to “intercessors”—other users who have volunteered to pray over incoming requests. Once prayed for, the original requestor receives a confirmation. Below is a thorough overview of how the application works, the business logic, and the overall code layout.


## High-Level Functionality

1. **User Sign-Up and Membership**
   • A user texts “pray” to a designated phone number to begin the sign-up process.
   • The application steps the user through a multi-stage signup, where the user can choose to remain anonymous, whether to become an intercessor, and how many prayers they can handle per week.
   • Upon successful sign-up, the user can send prayer requests any time.
   • A user can also opt out by texting “cancel” or “stop,” which removes them from the system.

2. **Submitting Prayer Requests**
   • Members simply text any request to the same phone number (e.g., “Please pray for XYZ”).
   • The system looks for available intercessors (factors: weekly prayer counts, whether they already have an active prayer, etc.).
   • If intercessors are available, the request is sent out to them over SMS. If none are available, the request remains in a “queued” state until the system can assign it.

3. **Intercessing and Confirmation**
   • When any intercessor has prayed for the request, they reply “prayed.”
   • PrayerTexter then notifies the requestor that someone has prayed. That intercessor’s active prayer is removed from their queue (so they can accept more).

4. **Additional Features**
   • Users can text “help” to receive the phone number’s contact and help information (required by SMS service regulations).
   • Multiple phone numbers can be assigned to handle announcements or asynchronous tasks (like stateresolver).


## Main Technical Flows

1. **Sign-Up Flow**
   1) A user texts “pray.”
   2) The system checks if they are new; if so, sets them as “IN PROGRESS,” step one.
   3) They are asked their name (or choose “2” for anonymous).
   4) They decide whether to be a regular member or an intercessor. If intercessor, how many prayers per week.
   5) The user is flagged “COMPLETE,” enabling them to submit requests. If intercessor, they’re added to the “IntercessorsPhones” list.

2. **Prayer Request**
   1) A member texts any arbitrary message with a prayer need.
   2) The system checks for profanity. If found, the request is refused. Otherwise, it tries to find available intercessors.
   3) Each suitable intercessor is updated in DynamoDB (incrementing their prayer counts, verifying no active request conflicts).
   4) The request is saved as an “active prayer” for each intercessor.
   5) If no intercessors can be assigned, the request goes into “QueuedPrayers.”

3. **Completing a Prayer**
   1) Intercessors reply “prayed.”
   2) If an active prayer is found for their phone number, it is removed from “ActivePrayers.”
   3) The requestor is notified that their prayer has been prayed over—unless the requestor has canceled membership.
   4) The intercessor’s “active prayer” slot is now cleared.

4. **Member Removal**
   1) A user can text “cancel” or “stop.”
   2) They’re removed from “Members,” and if they are an intercessor, from “IntercessorPhones.”
   3) If they had an active prayer assigned, that prayer is changed from active to queued so that future intercessors may cover it.

5. **StateTracker**
   - Tracks operations (e.g., sign-up or prayer assignment failures) for later recovery.
   - On success, the state is removed from StateTracker. On failure, it’s stored with the error message.


## Directory and Code Structure

PrayerTexter is structured to separate code for domain logic, AWS integrations, utility helpers, and the actual commands (Lambda entries). Notable directories and files:

1. **cmd Folder (Lambda Entrypoints)**
   - Each subfolder is a small Lambda function with its own “main.go.”
   - • `prayertexter`: The main function that receives incoming text messages (via API Gateway) and processes them through the “prayertexter” logic.
   - • `announcer`: Intended for sending announcements to all members, e.g., scheduled updates or maintenance.
   - • `stateresolver`: A scheduled (cron-like) Lambda for tasks such as assigning queued prayers, retrying failed operations, or sending reminders to intercessors.

2. **internal/config**
   - Central place to initialize configuration using Viper.
   - Sets default values for AWS retry attempts, backoff times, DynamoDB timeouts, table names, etc.
   - Maintains overrides via environment variables (e.g., “PRAY_CONF_AWS_SMS_PHONE” can override the default SMS phone number).

3. **internal/db**
   - DynamoDB logic (Get, Put, Delete) in a generic, reusable manner.
   - Contains a “DDBConnecter” interface simulating the AWS client; used in tests via mock implementations.
   - Logic for timeouts, table name configuration, and helper functions for retrieving or storing objects.

4. **internal/messaging**
   - Sends and receives text messages (SMS) using Amazon Pinpoint.
   - Defines the “TextMessage” structure (the core payload).
   - Provides logic for generating message strings (signup prompts, help messages, prayer instructions, etc.).
   - Offers “SendText” to actually transmit the SMS.

5. **internal/mock**
   - Mock implementations of external interfaces (DynamoDB, Pinpoint) for unit testing.
   - Enables thorough testing without calling real AWS services.

6. **internal/object**
   - Houses the main domain models (i.e., “Member,” “Prayer,” “IntercessorPhones,” “StateTracker,” etc.).
   - Each model has “Get,” “Put,” “Delete,” and specialized logic.
   - Example: “Member” includes fields for phone number, name, prayer count, etc. “Prayer” ties requestors to their assigned intercessors.

7. **internal/prayertexter**
   - Core business rules for receiving and handling text messages.
   - The “MainFlow” function decides how to handle each inbound SMS (sign up, help, cancel, prayer request, or prayer completion).
   - Calls out to supporting functions (e.g., “signUp,” “memberDelete,” “prayerRequest,” “completePrayer,” etc.).
   - “FindIntercessors” picks suitable intercessors based on prayer count, weekly limits, and existing active prayers.

8. **internal/utility**
   - Common helper functions: error wrappers, AWS configuration (i.e., “IsAwsLocal,” custom AWS retryer), random ID generation, slice utility, etc.

9. **localdev**
   - Docker Compose scripts for local DynamoDB, JSON files describing local DB tables, plus a helper shell script to start everything.

10. **Makefile and Templates**
   - Allows building each command for different architectures (x86/arm64).
   - “template.yaml” is the AWS SAM template describing the Lambda functions, the DynamoDB tables, and an API Gateway for receiving SMS events.

11. **Testing**
   - Tests are located alongside their packages in files named “*_test.go.”
   - Makes extensive use of “mock” packages for AWS calls to keep tests deterministic.
   - Covers sign-up flows, prayer queue logic, text-sending, and error-handling scenarios.


## Summary

- “cmd/prayertexter/main.go” is the primary Lambda handler for inbound SMS events via API Gateway.
- “internal/prayertexter/prayertexter.go” orchestrates each message’s flow: sign-up, prayer requests, completion, cancels, etc.
- “internal/object” models the data stored in DynamoDB (Members, Prayers, IntercessorPhones, etc.). Each model provides CRUD capabilities.
- “internal/db” generalizes DynamoDB interactions so that the logic can be shared and tested easily.
- “internal/messaging” handles SMS logic, from constructing messages to sending them through AWS Pinpoint.
- “internal/config” and “internal/utility” handle environment config, error handling, and AWS session setup.
- Tests leverage “mock” frameworks to avoid actual AWS calls.