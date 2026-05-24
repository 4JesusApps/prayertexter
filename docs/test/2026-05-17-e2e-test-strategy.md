# PrayerTexter E2E & Integration Test Strategy

**Last updated**: 2026-05-24 (revision 2 — incorporates AWS Prescriptive Guidance, AWS
Integrated Application Test Kit, and the SMS event-destination harness pattern; reframes
Layer 4 around a Gameplan A / Gameplan B split; deprioritizes Layer 5)

## Status

Tracks layers implemented on branch `test-enhancements` (not yet merged at time of writing).

| Layer | Status | Where it lives |
|-------|--------|----------------|
| 2a. PinpointSender retry tests + SNS payload fixture | Done | `internal/messaging/pinpoint_test.go`, `cmd/prayertexter/testdata/sns-event.json`, `cmd/prayertexter/main_test.go` |
| 1. Testcontainers integration tests | Done (~40 tests) | `internal/integration/` (build tag `integration`) |
| 2b. Handler refactor + handler tests | Done | `cmd/prayertexter/main.go` (`processRecords`, `newRouter`), `dev/prayertexter/main.go` (`parseRequest`), matching `_test.go` files |
| 3. SAM local automation | Skipped | See [Layer 3: Why this was skipped](#layer-3-automated-sam-local-smoke-tests) |
| 4. Cloud E2E tests | Planned | This revision adds Gameplan A (staging) vs Gameplan B (ephemeral) |
| 5. Production smoke test | Deferred indefinitely | CloudWatch alarms on Pinpoint metrics are the cheaper substitute |

Tangential improvements landed alongside the test work:

- `internal/buildinfo`: `Version()` reads `debug.ReadBuildInfo()` instead of the half-wired
  `-ldflags "-X main.version=..."` pattern. Every build path (CI, SAM, dev) now logs a
  real commit SHA without opt-in.

## Table of Contents

- [Current State](#current-state)
- [Architecture Observations](#architecture-observations)
- [Option Analysis](#option-analysis)
  - [Option A: End-User Messaging Simulator Number](#option-a-end-user-messaging-simulator-number)
  - [Option B: Expanding SAM Local Testing](#option-b-expanding-sam-local-testing)
  - [Option C: Multi-Layered Integration Tests (testcontainers-go)](#option-c-multi-layered-integration-tests-testcontainers-go)
- [Industry Patterns & Research](#industry-patterns--research)
- [Recommended Multi-Layer Strategy](#recommended-multi-layer-strategy)
  - [Layer 1: Integration Tests with Testcontainers](#layer-1-integration-tests-with-testcontainers)
  - [Layer 2: Handler & Pinpoint Unit Tests](#layer-2-handler--pinpoint-unit-tests)
  - [Layer 3: Automated SAM Local Smoke Tests](#layer-3-automated-sam-local-smoke-tests)
  - [Layer 4: Cloud E2E Tests](#layer-4-cloud-e2e-tests)
    - [Shared Test Harness Design](#shared-test-harness-design)
    - [Gameplan A: Permanent Staging Stack](#gameplan-a-permanent-staging-stack)
    - [Gameplan B: Ephemeral Stacks per Run](#gameplan-b-ephemeral-stacks-per-run)
    - [Prerequisite: Template Parameterization](#prerequisite-template-parameterization)
    - [CI Auth: GitHub OIDC](#ci-auth-github-oidc)
  - [Layer 5: Production Smoke Test](#layer-5-production-smoke-test)
- [Priority Matrix](#priority-matrix)
- [Recommendation](#recommendation)
- [Research References](#research-references)

---

## Current State

### What Exists

- **Unit tests with mocks**: Service layer is well-tested using mockery-generated mocks
  (router, member, prayer, admin). Uses `testify/suite` and `testify/mock`.
- **SAM local dev setup**: Docker-based DynamoDB Local + `sam local start-api` with an API
  Gateway trigger. Testing is manual via `curl`.
- **CI**: GitHub Actions runs `go test -v ./...` and `golangci-lint` on push/PR to main.

### What's Missing

| Gap | Impact |
|-----|--------|
| Zero integration tests against real DynamoDB (all mocked) | Can't catch serialization bugs, schema mismatches, or query issues |
| Zero handler-level tests (cmd/ entry points at 0% coverage) | SNS event parsing, error handling untested |
| Zero SMS testing (PinpointSender untested; SAM local just logs) | Retry logic, throttle handling, message formatting untested |
| Zero automated end-to-end flow testing | Multi-step flows (sign-up, prayer lifecycle) only tested in isolation |
| StateController (scheduled jobs) never tested outside unit mocks | Queue assignment and reminder logic untested in realistic conditions |
| No contract tests for inbound SMS webhook payload | Payload format changes could silently break the app |

---

## Architecture Observations

The app has excellent interface boundaries that make testing tractable:

| Interface | Location | What It Abstracts |
|-----------|----------|-------------------|
| `MessageSender` | `internal/messaging/sender.go` | SMS sending |
| `DDBClient` | `internal/repository/dynamodb.go` | DynamoDB SDK |
| `PinpointClient` | `internal/messaging/pinpoint.go` | Pinpoint SDK |
| Per-entity repository interfaces | `internal/repository/*.go` | Data access |

The `AWS_SAM_LOCAL` check in `internal/messaging/pinpoint.go` is a seam that already
short-circuits real SMS sends. This same pattern can be leveraged for integration tests.

The inbound message flow has two entry shapes: **SNS event** (production) and **API Gateway
event** (dev), both converging at `Router.Handle(ctx, TextMessage)`. This means integration
tests can target `Router.Handle` directly and cover all business logic without needing
Lambda/API Gateway/SNS infrastructure.

### Service Dependency Graph

```
Lambda Handler (cmd/)
    |
    v
Router (service/router.go) -- message dispatcher
    |--- MemberService -- sign-up, deletion, help
    |--- PrayerService -- prayer requests, completion, scheduling
    |--- AdminService  -- user blocking
    |
    v
Repository Layer (internal/repository/)
    |--- MemberRepository
    |--- PrayerRepository (active + queued)
    |--- BlockedPhonesRepository
    |--- IntercessorPhonesRepository
    |
    v
DynamoDB (generic DynamoDBRepository[T])
```

---

## Option Analysis

### Option A: End-User Messaging Simulator Number

**Concept**: A dedicated phone number that automated tests send real SMS to, exercising the
full production path (Phone -> Pinpoint -> SNS -> Lambda -> DynamoDB -> Pinpoint -> Phone).

#### Available Approaches

| Approach | Cost | Fidelity | CI-Friendly |
|----------|------|----------|-------------|
| AWS SMS Simulator Numbers | Free | Medium (tests Pinpoint API, no carrier delivery) | Yes |
| Mailosaur (test phone number with API) | ~$100/mo | Very High (real delivery + programmatic read) | Partially (latency) |
| Dedicated test Pinpoint number | Per-SMS cost | Highest | No (slow, flaky, expensive) |
| Twilio test credentials + magic numbers | Free | N/A (app uses Pinpoint, not Twilio) | N/A |

#### Assessment

AWS SMS Simulator Numbers are the best fit for this app. They validate Pinpoint API
integration (auth, formatting, throttle handling) without carrier costs or delivery latency.
Mailosaur or a real number is only valuable for periodic production smoke tests, not CI.

**Key limitation**: This only tests the *outbound* half. The *inbound* half (SMS -> Pinpoint
-> SNS -> Lambda) is AWS-managed infrastructure, not application code. The only way to test
that path is a deployed stack with a real phone number sending real SMS. That's a
nightly/weekly smoke test, not a CI gate.

#### AWS SMS Simulator Numbers Reference

AWS provides simulator phone numbers that can be used as the destination for Pinpoint
`SendTextMessage` calls. They simulate success and various failure modes without actual
carrier delivery. See
[AWS SMS Simulator Phone Numbers](https://docs.aws.amazon.com/sms-voice/latest/userguide/test-phone-numbers.html).

---

### Option B: Expanding SAM Local Testing

**Concept**: Automate the existing `dev/` SAM local setup with scripted test scenarios.

#### Current State

The dev directory contains:
- `dev/dynamodb/compose.yaml` -- DynamoDB Local in Docker (port 8000)
- `dev/prayertexter/template.yaml` -- SAM template with API Gateway trigger + embedded tables
- `dev/prayertexter/main.go` -- API Gateway handler (vs SNS handler in production)
- `dev/Makefile` -- Build commands for all three Lambda functions

Testing is manual: start Docker, start SAM local, `curl` POST JSON payloads.

#### Expansion Paths

1. **Scripted scenario runner**: A shell script or Go program that:
   - Boots DynamoDB Local + creates tables
   - Seeds test data (members, intercessors, blocked phones)
   - Starts SAM local
   - Fires HTTP requests simulating text messages
   - Queries DynamoDB to assert state changes
   - Parses Lambda logs for expected SMS log lines

2. **Docker Compose orchestration**: Extend `compose.yaml` to include SAM local + seed data.

3. **Makefile targets**: `make seed`, `make test-scenarios` for turnkey dev testing.

#### Assessment

This is worth improving as a **developer workflow tool** but should not be the CI backbone.

**Reasons:**
- SAM local is slow to start (cold Docker containers)
- The dev handler uses API Gateway events, not SNS -- it tests a different entry path than
  production
- StateController and EventBridge scheduling are not exercisable through SAM local
- Log-based SMS assertion is fragile
- Docker-in-Docker or Docker-in-CI adds complexity to GitHub Actions

**Verdict**: Keep SAM local as a manual dev tool and build the automated test layer
differently.

---

### Option C: Multi-Layered Integration Tests (testcontainers-go)

**Concept**: Use `testcontainers-go` with the DynamoDB module to spin up a real DynamoDB
instance per test suite, wire up real repositories, and test full business flows with a
recording SMS mock.

This is the **highest-ROI option**.

#### Architecture

```
Go test process
|-- testcontainers-go starts DynamoDB Local container
|-- Creates all 4 tables (Member, ActivePrayer, QueuedPrayer, General)
|-- Wires real repositories against the container
|-- Uses RecordingMessageSender (captures all SMS for assertion)
|-- Builds real Router + Services
|-- Runs full user journey scenarios
```

#### What This Catches That Unit Tests Cannot

The repository layer uses a generic `DynamoDBRepository[T]` with only single-hash-key
GetItem/PutItem/DeleteItem/Scan -- no GSIs, range keys, or conditional expressions. The
"schema drift" surface area is small (just the `keyField` value and `attributevalue`
struct-tag-less marshaling). The real value of Layer 1 is **multi-step flow correctness**
under realistic DB behavior, not schema-mismatch detection:

- **Rollback correctness**: `reserveIntercessor` + `rollbackAssignedPrayer` in
  `internal/service/prayer.go` performs a multi-step Save/Delete sequence with rollback on
  partial failure. Mocks can't validate the DB is left consistent when SendMessage fails
  after the reservation Save succeeded.
- **Shared `General` table contention**: Both `BlockedPhones` and `IntercessorPhones`
  write to the same table with different keys (`internal/repository/phones.go`). Concurrent
  intercessor sign-ups + admin block operations can collide -- mocks paper over this.
- **Queue-to-active state transitions**: `processQueuedPrayer` reads via `GetAll(queued=true)`,
  writes to the active table, and deletes from the queued table. Ordering and visibility
  semantics differ from a perfect in-memory mock.
- **Full prayer lifecycle**: request -> queue -> assign -> remind -> complete
- **Sign-up state machine** (stages 1 -> 2 -> 3 -> 99) against real persisted state
- **Edge cases in intercessor selection** (`FindIntercessors` with skip lists, weekly-limit
  reset boundary) with real data
- **Marshaling round-trips** for domain structs (catches the day someone adds a `time.Time`
  field and forgets it doesn't survive default attributevalue marshaling)

#### Example Test Scenarios

| Scenario | Steps | Asserts |
|----------|-------|---------|
| Full sign-up flow | Send "pray" -> send name -> select type -> set limit | Member in DB with correct fields, SMS sent at each step |
| Prayer with available intercessors | Seed 3 intercessors, send prayer request | ActivePrayer records created, SMS to all intercessors + requestor |
| Prayer with no intercessors | Empty intercessor pool, send prayer request | QueuedPrayer created, queuing SMS sent to requestor |
| Scheduled job: assign queued | Seed queued prayer + intercessors, call RunScheduledJobs | Prayer moved from queued to active, SMS sent |
| Scheduled job: remind | Seed active prayer with old reminder date, call RunScheduledJobs | Reminder SMS sent, count incremented |
| Block user | Admin sends "#block +1234567890" | Member gets blocked SMS, blocked phones list updated |
| Profanity filter | Send prayer with profanity | Rejection SMS sent, no prayer created |
| Help command | Existing member sends "help" | Help text SMS sent |
| Cancel/delete | Existing member sends "cancel" | Member removed from DB, confirmation SMS |
| Anonymous prayer | Send prayer with #anon trigger | Prayer created with masked requestor name |

---

## Industry Patterns & Research

This section captures the 2026 state-of-practice for serverless test automation, gathered
from AWS official guidance, the AWS Compute Blog, and serverless practitioners. The
patterns documented here inform the revised Layer 4 below.

### AWS Prescriptive Guidance (official)

The [AWS serverless testing prescriptive guidance](https://docs.aws.amazon.com/prescriptive-guidance/latest/serverless-application-testing/best-practices.html)
opens with: *"based on current tooling, we recommend that you focus on testing in the cloud
as much as possible."* The headline implications for this app:

- **Emulators (LocalStack, SAM local) are second-class** for E2E and integration. They're
  best used for fast unit-test feedback loops and for offline development; cloud testing is
  the source of truth.
- **Mocks of cloud services should not be used to validate the integration with those
  services.** This is exactly what Layer 1 (testcontainers DynamoDB Local) and Layer 4
  (real AWS) are for.
- **End-to-end tests should not use mocks.** They test states and complex behavior that
  cannot be easily simulated.
- **Separate Lambda code from business logic.** The Lambda handler should be a thin
  adapter; business logic should be portable and testable without Lambda-specific
  scaffolding. This is exactly what the Layer 2b refactor did (`processRecords`,
  `parseRequest`, `newRouter`).
- **Use test harnesses for asynchronous workflows.** A test harness is *testing
  infrastructure you deploy alongside your application* — event listeners that subscribe to
  the same events your app produces, storage where test results can be captured, and
  polling logic in tests that waits for expected outcomes. This is the key pattern
  underpinning the revised Layer 4 design below.
- **Use unique identifiers (correlation IDs) for test isolation** so concurrent or repeated
  test runs don't interfere with each other.

### AWS Integrated Application Test Kit (IATK)

[IATK](https://aws.amazon.com/blogs/compute/aws-integrated-application-test-kit/) is AWS's
own implementation of the test-harness pattern, released after the initial version of this
document. It is currently **Python-only**, so it can't be used directly from this Go app,
but it codifies the pattern Layer 4 should follow:

1. The harness creates EventBridge rules / SQS queues that mirror the production event
   topology, scoped to the test run.
2. The system-under-test publishes events as it normally would; the harness captures
   them.
3. Tests poll the harness for expected outcomes with a configurable SLA timeout.
4. The harness tears itself down at the end of the run.

The Go-language equivalent for this app is a ~100-line collector Lambda plus a DynamoDB
capture table — sketched in [Shared Test Harness Design](#shared-test-harness-design).

### Yan Cui's Pattern (theburningmonk.com)

Yan Cui — one of the most-cited voices in serverless practice — has been doing
[ephemeral-stack E2E testing](https://theburningmonk.com/2022/05/my-testing-strategy-for-serverless-applications/)
for years. His specific
[SNS/Kinesis E2E technique](https://theburningmonk.com/2019/09/how-to-include-sns-and-kinesis-in-your-e2e-tests/)
is the canonical reference for capturing async event flows:

- Deploy a test-only "listener" Lambda **conditionally** via CloudFormation conditions or
  stage-specific templates — it does not run in production.
- The listener subscribes to the same event source the app publishes to (SNS topic,
  Kinesis stream, etc.) and writes received messages to a temporary DynamoDB table.
- Tests poll that table once per second for ~20 attempts, looking for messages matching a
  per-test correlation ID.
- For SNS specifically, an SQS queue subscribed to the topic is a simpler alternative.

Cui frames "remocal testing" (local Go code, real cloud services) as the daily
inner-loop, with full E2E reserved for CI / nightly.

### LocalStack Maturity for This App

The LocalStack ecosystem has a testcontainers-go module and supports many AWS services,
but the relevant services for this app sit at the edge of its coverage:

- **DynamoDB**: well-supported. Already used at Layer 1.
- **SNS**: well-supported. Could replace some of the real-AWS test harness if cost is an
  issue.
- **Pinpoint (v1)**: present but [Amazon Pinpoint is retiring on 2026-10-30 and LocalStack
  has signaled removal soon after](https://docs.localstack.cloud/aws/services/pinpoint/).
- **AWS End User Messaging SMS V2** (the service this app actually uses via the
  `pinpointsmsvoicev2` SDK package): support is uncertain. **Not safe to depend on.**

**Implication**: Real AWS is the only credible path for testing the Pinpoint API
integration. LocalStack is fine for DynamoDB+SNS-only test scenarios but doesn't move the
needle for the SMS-out half of the loop.

### AWS End User Messaging SMS V2: Testing Affordances

This is the part the original version of this doc under-explored. The V2 service has two
features that together make outbound-SMS assertions clean and free:

1. **[Simulator phone numbers](https://docs.aws.amazon.com/sms-voice/latest/userguide/test-phone-numbers.html)** — both origination and destination. Messages stay
   inside AWS (no carrier delivery, no cost). For the US-based app, both ends must be US
   simulator numbers. Different destination simulator numbers produce different event
   outcomes (success, various failure modes), so tests can exercise retry paths.

2. **[Configuration sets with event destinations](https://docs.aws.amazon.com/sms-voice/latest/userguide/configuration-sets-event-destinations.html)** — every
   `SendTextMessage` call against a configured phone pool emits send/delivery events to
   one of CloudWatch Logs, Kinesis Firehose, or an **SNS topic**. The SNS topic option is
   what makes the harness Lambda pattern straightforward: subscribe a collector Lambda to
   the event-destination SNS topic, write events to DynamoDB keyed by correlation ID,
   tests poll for them.

Both features are free for testing volumes (simulator numbers don't incur SMS costs;
configuration set events are billed at standard SNS/DDB rates).

### GitHub Actions → AWS Authentication

Modern practice is
[GitHub OIDC](https://aws.amazon.com/blogs/security/use-iam-roles-to-connect-github-actions-to-actions-in-aws/)
with short-lived role credentials, not long-lived `AWS_ACCESS_KEY_ID` secrets in repo
settings. The `aws-actions/configure-aws-credentials` action handles the OIDC handshake; a
single IAM role in the AWS account trusts GitHub's OIDC provider, scoped by repo and
optionally branch / environment. Apply to any future E2E workflow on day one.

### CloudFormation Ephemeral Stack Mechanics

Ephemeral stacks have known footguns documented in
[CloudFormation orphan-resource guidance](https://www.serverlesssam.com/p/cleanup-cloudformation/):

- **Lambda + VPC**: ENI cleanup delays stack deletion by minutes. This app has no VPC
  Lambdas, so this footgun does not apply.
- **S3 buckets with objects**: cannot auto-delete; stack delete fails. Avoid creating S3
  buckets in test stacks, or use `BucketName` + lifecycle policies that auto-empty.
- **SAM deploy bucket**: `sam delete` handles cleanup of the SAM artifacts bucket and ECR
  companion stack if present. Use `sam delete` rather than raw `aws cloudformation
  delete-stack`.
- **Cross-stack `!ImportValue` with hardcoded export names blocks concurrent ephemeral
  deploys.** This app's `deploy/` layout has this problem today. The fix is documented in
  [Prerequisite: Template Parameterization](#prerequisite-template-parameterization).

### What the aws-samples/serverless-test-samples Repo Shows (and Doesn't)

The [official AWS samples repo](https://github.com/aws-samples/serverless-test-samples)
covers Python, C#, Java, and TypeScript with patterns for API Gateway, EventBridge, Step
Functions, Kinesis, and schema/contract testing. **There are no Go samples.** The
patterns translate cleanly — they're language-agnostic — but expect to do some
boilerplate translation rather than copy-pasting.

---

## Recommended Multi-Layer Strategy

### Layer 1: Integration Tests with Testcontainers

**Priority**: HIGH -- Do second (after Layer 2a quick wins). See Priority Matrix.

**Tool**: `testcontainers-go/modules/dynamodb`

**Runs in**: CI on every PR + locally

**Tests**: Full service flows against real DynamoDB with recorded SMS assertions.

#### Proposed File Layout

```
internal/integration/
|-- integration_test.go    -- TestMain: start DynamoDB container, create tables
|-- signup_test.go         -- Member sign-up flows
|-- prayer_test.go         -- Prayer request/complete/queue/assign flows
|-- scheduler_test.go      -- StateController scheduled job flows
|-- admin_test.go          -- Block user flows
|-- router_test.go         -- Message routing with real DB state
|-- helpers_test.go        -- RecordingMessageSender, test data seeders
```

#### RecordingMessageSender

A test double that implements `MessageSender` and stores all calls for assertion:

```go
type RecordingMessageSender struct {
    mu       sync.Mutex
    Messages []SentMessage
}

type SentMessage struct {
    To   string
    Body string
}

func (r *RecordingMessageSender) SendMessage(ctx context.Context, to, body string) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.Messages = append(r.Messages, SentMessage{To: to, Body: body})
    return nil
}

func (r *RecordingMessageSender) Reset() {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.Messages = nil
}

func (r *RecordingMessageSender) AssertSentTo(t *testing.T, phone string, bodyContains string) {
    r.mu.Lock()
    defer r.mu.Unlock()
    for _, m := range r.Messages {
        if m.To == phone && strings.Contains(m.Body, bodyContains) {
            return
        }
    }
    t.Errorf("expected SMS to %s containing %q, got: %v", phone, bodyContains, r.Messages)
}
```

#### Build Tag and CI Integration

Use `//go:build integration` so `go test ./...` does not run them by default. CI runs them
as a separate step:

```yaml
# In .github/workflows/buildandtest.yml
- name: Integration tests
  run: go test -v -tags integration ./internal/integration/...
```

Docker is available in GitHub Actions runners natively, so testcontainers works without
additional setup.

#### Dependencies to Add

```
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/dynamodb
```

#### DynamoDB Local Setup Footgun

DynamoDB Local partitions data by access-key + region pair unless launched with `-sharedDb`.
Without it, the same test process using two different credential providers will see two
separate databases and silently fail to find records. Two equivalent fixes -- pick one and
apply it consistently in `TestMain`:

1. Launch the container with `-sharedDb` (mirrors `dev/dynamodb/compose.yaml`):

   ```go
   container, err := dynamodb.Run(ctx,
       "amazon/dynamodb-local:latest",
       dynamodb.WithSharedDB(),
   )
   ```

2. Or pin deterministic credentials before constructing the AWS config:

   ```go
   t.Setenv("AWS_ACCESS_KEY_ID", "test")
   t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
   t.Setenv("AWS_REGION", "us-west-1")
   ```

Either works; `-sharedDb` matches the dev environment and is the lower-friction default.

---

### Layer 2: Handler & Pinpoint Unit Tests

**Priority**: Layer 2a (PinpointSender + SNS payload fixture) is HIGHEST -- do first.
Layer 2b (handler refactor + handler tests) is MEDIUM and can be deferred. See Priority
Matrix.

**Runs in**: CI on every PR (part of `go test ./...`)

#### Handler Tests

Test the Lambda entry points with crafted SNS/API Gateway events and mocked services.

**Coverage targets:**
- Valid SNS event parsing and TextMessage extraction
- Multi-record SNS event handling (the warning log path in `cmd/prayertexter/main.go`)
- Malformed JSON in SNS message body (one record bad, others succeed -> `errors.Join`)
- AWS config initialization failure path
- API Gateway request parsing (dev handler)

**Approach**: The current `cmd/prayertexter/main.go` handler does all wiring inline (Config
-> AWS config -> repositories -> services -> Router). To make it testable, extract:

```go
// cmd/prayertexter/main.go
func newRouter(ctx context.Context, cfg config.Config) (*service.Router, error) { /* wiring */ }

func handler(ctx context.Context, snsEvent events.SNSEvent) error {
    cfg := config.Load()
    router, err := newRouter(ctx, cfg)
    if err != nil { return err }
    return processRecords(ctx, router, snsEvent.Records)
}

func processRecords(ctx context.Context, r *service.Router, records []events.SNSEventRecord) error { ... }
```

Tests target `processRecords` with a `*service.Router` built from mocks. This is a real
refactor (the inline wiring is currently ~30 lines per handler) -- weigh the coverage gain
against the cost. Many serverless shops accept ~0% coverage on the wiring half of `main.go`
because the alternative is a leaky abstraction over Lambda; the *event-parsing* half is
what matters and is worth extracting.

#### Inbound Payload Contract Fixture

Closes the "no contract tests for inbound SMS webhook payload" gap in the table above.
The Pinpoint inbound SMS event shape is owned by AWS and can drift. Cheapest defense:

```
internal/integration/testdata/sns-event.json   -- known-good captured payload
```

Plus a unit test in `cmd/prayertexter/main_test.go` that loads the fixture, feeds it to
the extracted `processRecords` (or to `json.Unmarshal` directly into `domain.TextMessage`),
and asserts the expected `domain.TextMessage` fields. Update the fixture whenever a real
production event reveals new fields. This catches the unmarshal-side regression without
needing a deployed stack.

#### PinpointSender Tests

Test with a mocked `PinpointClient` interface (already defined in `messaging/pinpoint.go`).

**Prerequisite**: `PinpointClient` is not currently in `.mockery.yaml`. Add it before
running mockery:

```yaml
# .mockery.yaml
github.com/4JesusApps/prayertexter/internal/messaging:
  config:
    dir: internal/mocks/messaging
    filename: mocks.go
  interfaces:
    MessageSender: {}
    PinpointClient: {}   # ADD THIS
```

Then `mockery` regenerates `internal/mocks/messaging/mocks.go` with `MockPinpointClient`.

| Test Case | Setup | Assert |
|-----------|-------|--------|
| Successful send | Mock returns nil | No error, SendTextMessage called once |
| Throttle -> retry -> success | Mock returns ThrottlingException then nil | No error, called twice |
| Throttle -> exhaust retries | Mock returns ThrottlingException 3 times | Error returned after 3 attempts |
| Non-throttle error | Mock returns other API error | Error returned immediately (1 call) |
| AWS_SAM_LOCAL=true | Set env var | SendTextMessage never called, no error |
| Message wrapping | Any send | Verify body has MsgPre prefix and MsgPost suffix |

---

### Layer 3: Automated SAM Local Smoke Tests

**Status**: **Skipped.** Layer 1 (testcontainers integration tests, ~40 cases covering
sign-up, prayer, scheduler, admin, and router flows) lands the same coverage with better
assertions and faster iteration. The doc's own caveat — *"worthwhile only when Layer 1
lands first; otherwise it duplicates Layer 1's coverage with worse tooling. Its unique
value is enabling local StateController invocation"* — turns into "skip it entirely" once
Layer 1 includes `scheduler_test.go`.

The one piece that's still worth doing (and is independent of the rest of Layer 3) is
[adding StateController to the dev/prayertexter SAM template](#adding-statecontroller-to-the-dev-workflow)
so devs can `sam local invoke StateController` against the same DDB Local container they
use for curl-driven manual testing. That's a 5-line YAML change, not a layer.

The original Layer 3 spec is preserved below for historical context, but the recommended
action is to skip the seed scripts and scenario runners. `dev/addmembers.sh` already
covers the only practical use case (manual seed via curl).

**Original priority (preserved for context)**: MEDIUM -- Improves developer workflow.

**Runs in**: Developer machine only (not CI)

#### Proposed Additions to dev/

```
dev/
|-- Makefile              -- (existing, add new targets)
|-- dynamodb/
|   |-- compose.yaml      -- (existing)
|   |-- seed.sh           -- NEW: create tables + seed test data
|   |-- seed-data/        -- NEW: JSON files with test members, intercessors, etc.
|-- prayertexter/
|   |-- template.yaml     -- (existing)
|   |-- main.go           -- (existing)
|-- scenarios/            -- NEW: automated test scenarios
|   |-- run.sh            -- Orchestrator: start infra, run scenarios, report
|   |-- signup.sh         -- Sign-up flow via curl
|   |-- prayer.sh         -- Prayer request flow via curl
|   |-- admin.sh          -- Admin block flow via curl
```

#### New Makefile Targets

```makefile
seed:           # Create tables and populate with test data
test-scenarios: # Run all automated scenarios against running SAM local
start-all:      # docker compose up + sam local start-api (background)
stop-all:       # Tear down all local infra
```

#### Limitations (Documented Here for Awareness)

- Cannot test SNS event path (the dev handler uses an API Gateway adapter)
- EventBridge scheduling is not exercisable locally (no local EventBridge in SAM)
- SMS sends are logged only, not delivered
- Requires Docker + SAM CLI installed locally

#### Adding StateController to the Dev Workflow

`cmd/statecontroller/main.go` takes only `ctx` -- no event payload -- so it can be added
to `dev/prayertexter/template.yaml` as a second `AWS::Serverless::Function` and invoked via
`sam local invoke StateController` against the same DynamoDB Local instance. This unlocks
manual testing of `RunScheduledJobs` (queue assignment + reminder logic) without deploying
to AWS. EventBridge *triggering* still can't be tested locally, but the handler logic can.

```yaml
# Add to dev/prayertexter/template.yaml under Resources:
StateController:
  Type: AWS::Serverless::Function
  Metadata:
    BuildMethod: go1.x
  Properties:
    Architectures:
      - arm64
    CodeUri: ../../cmd/statecontroller/
    Handler: bootstrap
    Runtime: provided.al2023
```

Then: `sam local invoke StateController --docker-network sam-backend`.

---

### Layer 4: Cloud E2E Tests

**Priority**: HIGH -- The structural gap Layer 1 cannot fill (IAM, real SNS->Lambda event
source mapping, real Pinpoint integration, EventBridge scheduling, CloudFormation template
correctness, cold starts).

**Runs in**: Nightly via cron + on-demand via workflow_dispatch. **Not per-PR** —
ephemeral stack create + tear-down takes 5-10 minutes, slower than is useful for PR
feedback, and Layer 1 already covers what's worth covering per-PR.

This section was revised in 2026-05-24 to incorporate the AWS End User Messaging SMS V2
event-destination harness pattern, AWS IATK's harness conventions, and Yan Cui's
SNS-listener pattern. The original revision under-specified the outbound-SMS assertion
mechanism, which is the bit that turns "tests against real AWS" from log-scraping into
meaningful pass/fail.

#### What This Validates (That Lower Layers Cannot)

- Real IAM permissions (Lambda execution role, DynamoDB policies, Pinpoint access, SNS
  publish, EventBridge invoke)
- Real SNS -> Lambda event source mapping (the production inbound path)
- Real Pinpoint API integration (auth, phone pool, configuration set, retry behavior under
  real throttling)
- Real EventBridge -> StateController triggering
- CloudFormation template correctness (exports/imports resolve, IAM policies render)
- Lambda cold-start behavior and resource sizing
- Cross-stack wiring (DB -> Prayertexter -> StateController exports/imports)

#### What This Cannot Validate (and shouldn't try)

- Real carrier delivery — that's [Layer 5](#layer-5-production-smoke-test) territory and
  not worth the cost/flakiness for a ministry app this size
- The inbound carrier -> Pinpoint -> SNS half — AWS-managed, not application code; the
  closest you can get is the
  [SNS payload fixture in Layer 2a](#inbound-payload-contract-fixture) that pins the JSON
  shape that crosses from AWS-owned infra into your code
- Production-scale load — separate concern, not a CI gate
- Production data state — tests are designed to never touch prod tables

---

#### Shared Test Harness Design

This is the same across both Gameplan A and Gameplan B. It's the pattern AWS IATK
implements in Python and that Yan Cui has been doing manually in Node/JS for years.

```
Test stack (staging or ephemeral):
  All production resources (db, prayertexter, statecontroller)
  PLUS:
  - SMS simulator origination phone number, registered to the prayertexter phone pool
  - Configuration set "pt-test-events" attached to the phone pool
  - SNS topic "pt-test-sms-events" as the configuration set's event destination
  - DynamoDB table "pt-test-sms-capture" (capture log, keyed by correlation ID)
  - Lambda "pt-test-sms-collector": subscribes to pt-test-sms-events SNS topic,
    writes every event to pt-test-sms-capture
  - CloudFormation Outputs: ARNs + names of the inbound SNS topic, capture table,
    simulator origination number, and StateController Lambda ARN
```

**Per-test flow:**

```
1. corrId := uuid()                          // e.g. "pt-e2e-abc123def"
2. Synthesize an inbound SMS payload with corrId embedded in the body or in a custom
   attribute, and publish it as JSON to the inbound SNS topic via AWS SDK.
3. Poll the capture DynamoDB table for rows tagged with corrId. Use a bounded retry
   loop: ~30s timeout, 1s interval, give up with a clear "expected SMS X to phone Y
   not observed within 30s" assertion failure.
4. Assert on captured outbound events (destination phone, message body fragments).
5. Assert on production DynamoDB state directly (member created, prayer queued, etc.)
   using the SDK against the test stack's table names from stack outputs.
6. Delete rows tagged with corrId from the capture table (per-test cleanup).
```

**Why this works:**

- The collector Lambda is the [test harness AWS Prescriptive Guidance recommends](https://docs.aws.amazon.com/prescriptive-guidance/latest/serverless-application-testing/best-practices.html#test-harnesses) —
  observability infrastructure deployed alongside the application that turns async event
  flows into assertable state.
- Correlation IDs satisfy the "use unique identifiers for test isolation" rule from the
  same guidance, letting concurrent test runs or retries coexist without interfering.
- Pinpoint configuration set + SNS event destination is free for testing volumes and
  doesn't require a separate AWS service or third-party tool.

**Why NOT log scraping:** The original Layer 4 spec implied tests would parse Lambda logs
for outbound SMS lines (this is also how `dev/` testing works today). That's fragile —
log format changes break tests silently; assertions on log substrings don't compose; and
CloudWatch ingestion delays make polling slow. The event-destination harness gives
strongly-typed, queryable outbound assertions with no log parsing.

---

#### Gameplan A: Permanent Staging Stack

**Effort estimate**: ~3 days. **Recommended for this app.**

**Concept**: Deploy a long-lived `staging` triplet (db / prayertexter / statecontroller)
plus the test harness once. CI deploys updates to it on merge to a `staging` branch (or
on push to main, pre-prod-deploy). E2E tests run against it.

**Pros:**

- No deploy/teardown wait per CI run; tests start in seconds, not minutes
- One-time IAM/OIDC setup; no per-run trust-policy churn
- Cheaper to operate (DDB on-demand + Lambda free-tier = near-zero idle cost)
- Simpler failure modes (no stack-delete-failed orphans)
- Test isolation comes from per-test correlation IDs, not from per-run stack
  isolation

**Cons (and mitigations):**

- Drift risk: staging template can diverge from prod if someone edits one and forgets
  the other. Mitigation: the templates ARE the same files (`deploy/db`,
  `deploy/prayertexter`, `deploy/statecontroller`) parameterized by `StackPrefix` —
  staging is "the same templates, deployed with `StackPrefix=staging`."
- Doesn't validate "did this PR break CloudFormation template syntax" — staging is
  already deployed, so a broken template fails the next staging deploy, not the next
  E2E run. Mitigation: run `sam validate --lint` in CI per-PR.
- Shared test data accumulation: capture table grows. Mitigation: 7-day TTL on capture
  table rows + per-test cleanup of explicit corrId rows.

**CI workflow shape:**

```yaml
# .github/workflows/e2e-staging.yml
name: E2E Tests (Staging)
on:
  push:
    branches: [main]
  workflow_dispatch: {}
permissions:
  id-token: write
  contents: read
jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ secrets.E2E_ROLE_ARN }}
          aws-region: us-west-1
      - uses: aws-actions/setup-sam@v2
      - name: Deploy staging if changed
        run: |
          sam build -t deploy/db/template.yaml && \
            sam deploy --stack-name pt-staging-db --parameter-overrides StackPrefix=pt-staging \
              --no-confirm-changeset --no-fail-on-empty-changeset
          sam build -t deploy/prayertexter/template.yaml && \
            sam deploy --stack-name pt-staging-prayertexter --parameter-overrides StackPrefix=pt-staging \
              --no-confirm-changeset --no-fail-on-empty-changeset
          sam build -t deploy/statecontroller/template.yaml && \
            sam deploy --stack-name pt-staging-statecontroller --parameter-overrides StackPrefix=pt-staging \
              --no-confirm-changeset --no-fail-on-empty-changeset
      - name: Run E2E tests
        run: go test -v -tags e2e -timeout 15m ./test/e2e/...
        env:
          PT_E2E_STACK_PREFIX: pt-staging
```

---

#### Gameplan B: Ephemeral Stacks per Run

**Effort estimate**: ~5 days incl. template parameterization. **Defer until Gameplan A
reveals a need.**

**Concept**: Each CI run creates `pt-e2e-${sha}-{db,prayertexter,statecontroller}`,
deploys, tests, and deletes. The original Layer 4 spec from 2026-05-17.

**When this is worth the extra effort over Gameplan A:**

- You ship CloudFormation template changes often enough that "did this PR's template
  changes deploy cleanly" needs to be a per-run gate, not a `sam validate --lint`
  check
- Multiple developers want to E2E-test conflicting changes simultaneously without
  trampling staging
- You have explicit "production must match what CI deployed bit-for-bit" compliance
  needs (unlikely for this app)

**Workflow:**

1. CI assumes the OIDC role
2. `sam build && sam deploy` the db stack with `StackPrefix=pt-e2e-${sha}`
3. `sam build && sam deploy` the prayertexter stack with the same prefix
4. `sam build && sam deploy` the statecontroller stack with the same prefix
5. Discover resource ARNs from stack outputs (`aws cloudformation describe-stacks`)
6. Run `go test -tags e2e ./test/e2e/...` with discovered ARNs in env vars
7. `sam delete` each stack in reverse dependency order — must run on `if: always()`
   so failed tests don't leak resources

**Known footguns** (from [research](#cloudformation-ephemeral-stack-mechanics)):

- Stack delete on failure must be unconditional. Wrap with `if: always()` in the
  workflow.
- This app has no VPC Lambdas, so ENI cleanup delays don't apply — good.
- Be careful with any S3 buckets created in the test stack; they block delete if
  non-empty. Currently no test-stack S3 buckets are proposed.
- `sam delete` is preferred over raw `aws cloudformation delete-stack` because it
  also cleans up the SAM artifacts bucket and ECR companion stack.

**CI workflow shape:**

```yaml
# .github/workflows/e2e-ephemeral.yml
name: E2E Tests (Ephemeral)
on:
  schedule:
    - cron: '0 6 * * *'  # 6am UTC daily
  workflow_dispatch: {}
permissions:
  id-token: write
  contents: read
jobs:
  e2e:
    runs-on: ubuntu-latest
    env:
      PT_E2E_STACK_PREFIX: pt-e2e-${{ github.sha }}
    steps:
      - uses: actions/checkout@v4
      - uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ secrets.E2E_ROLE_ARN }}
          aws-region: us-west-1
      - uses: aws-actions/setup-sam@v2
      - name: Deploy ephemeral stacks
        run: |
          for stack in db prayertexter statecontroller; do
            sam build -t deploy/$stack/template.yaml
            sam deploy --stack-name $PT_E2E_STACK_PREFIX-$stack \
              --parameter-overrides StackPrefix=$PT_E2E_STACK_PREFIX \
              --no-confirm-changeset --no-fail-on-empty-changeset
          done
      - name: Run E2E tests
        run: go test -v -tags e2e -timeout 15m ./test/e2e/...
      - name: Teardown ephemeral stacks
        if: always()
        run: |
          for stack in statecontroller prayertexter db; do
            sam delete --stack-name $PT_E2E_STACK_PREFIX-$stack --no-prompts || true
          done
```

---

#### Prerequisite: Template Parameterization

Required for both Gameplans. The current `deploy/` layout uses hardcoded export names
(`db-MemberTableName`, `prayertexter-SMSPhonePoolARN`, etc.) which prevent two
deployments from coexisting in the same account.

**The fix** (~2-3 hours):

1. Add a `StackPrefix` parameter (default: `pt-prod` for production parity) to all three
   templates.
2. Change every `Export.Name: db-MemberTableName` to `Export.Name: !Sub
   "${StackPrefix}-db-MemberTableName"`.
3. Change every `!ImportValue db-MemberTableName` to `!ImportValue !Sub
   "${StackPrefix}-db-MemberTableName"`.
4. Production deploys keep `StackPrefix=pt-prod` (or whatever the current effective
   prefix maps to); staging deploys use `StackPrefix=pt-staging`; ephemeral runs use
   `StackPrefix=pt-e2e-${sha}`.

This is the smallest diff that unblocks all downstream E2E work without collapsing the
three-stack layout. See the [CloudFormation export-name parameterization
guidance](https://medium.com/@geoff.ford_33546/cloudformation-export-management-2f25b2d09443)
for the standard pattern.

---

#### CI Auth: GitHub OIDC

Use OIDC, not long-lived `AWS_ACCESS_KEY_ID` secrets in GitHub. One-time setup:

1. Create an OIDC provider in IAM trusting `token.actions.githubusercontent.com`.
2. Create an IAM role (`prayertexter-e2e-deploy`) with a trust policy scoped to
   `repo:4JesusApps/prayertexter:ref:refs/heads/main` (or per-branch as needed).
3. Attach a deploy + test policy: `cloudformation:*`, `iam:PassRole`, scoped DynamoDB
   actions on `pt-staging-*` / `pt-e2e-*` table ARNs, scoped Lambda + SNS + Pinpoint
   actions, S3 actions on the SAM artifacts bucket.
4. Store the role ARN as a repo variable (`E2E_ROLE_ARN`) — it's not a secret since the
   role can only be assumed by the configured GitHub OIDC trust.

Standard pattern, documented in the [AWS Security Blog OIDC walkthrough](https://aws.amazon.com/blogs/security/use-iam-roles-to-connect-github-actions-to-actions-in-aws/).

---

#### Proposed Go File Layout

```
test/e2e/
|-- e2e_test.go         -- TestMain: discover stack outputs into package-level vars
|-- inbound_test.go     -- Publish to SNS, assert DB state + captured outbound SMS
|-- scheduler_test.go   -- Invoke StateController, assert DB state + captured outbound SMS
|-- harness.go          -- Collector table polling, corrId generation, cleanup
|-- aws.go              -- Thin AWS SDK wrappers (publish to SNS, query DDB, invoke Lambda)
```

Build-tagged with `//go:build e2e` so `go test ./...` and the Layer 1 integration suite
remain unaffected.

#### Test Harness CloudFormation Snippet (sketch)

To live in a new `deploy/test-harness/template.yaml` and deploy alongside the staging or
ephemeral triplet:

```yaml
Parameters:
  StackPrefix:
    Type: String
  PhonePoolId:
    Type: String   # Imported from prayertexter stack output

Resources:
  SmsEventsTopic:
    Type: AWS::SNS::Topic
    Properties:
      TopicName: !Sub "${StackPrefix}-sms-events"

  CaptureTable:
    Type: AWS::DynamoDB::Table
    Properties:
      TableName: !Sub "${StackPrefix}-sms-capture"
      BillingMode: PAY_PER_REQUEST
      AttributeDefinitions:
        - AttributeName: corrId
          AttributeType: S
        - AttributeName: eventId
          AttributeType: S
      KeySchema:
        - AttributeName: corrId
          KeyType: HASH
        - AttributeName: eventId
          KeyType: RANGE
      TimeToLiveSpecification:
        AttributeName: expireAt
        Enabled: true

  CollectorFunction:
    Type: AWS::Serverless::Function
    # ... subscribes to SmsEventsTopic, extracts corrId from message
    # attributes or payload, PutItem to CaptureTable

  ConfigurationSet:
    Type: AWS::PinpointSMSVoiceV2::ConfigurationSet
    Properties:
      ConfigurationSetName: !Sub "${StackPrefix}-events"

  EventDestination:
    Type: AWS::PinpointSMSVoiceV2::EventDestination
    Properties:
      ConfigurationSetName: !Ref ConfigurationSet
      EventDestinationName: capture
      MatchingEventTypes: [ALL]
      SnsDestination:
        TopicArn: !GetAtt SmsEventsTopic.Arn

Outputs:
  CaptureTableName:
    Value: !Ref CaptureTable
    Export:
      Name: !Sub "${StackPrefix}-CaptureTableName"
  ConfigurationSetName:
    Value: !Ref ConfigurationSet
    Export:
      Name: !Sub "${StackPrefix}-ConfigurationSetName"
```

The prayertexter Lambda's `PinpointSender` needs to pass the configuration set name on
`SendTextMessage` calls when running in staging/ephemeral; either via env var or by
detecting the simulator origination number. The current `PinpointSender` constructor
already accepts a phone pool string, so this is a minor extension.

---

### Layer 5: Production Smoke Test

**Priority**: DEFERRED INDEFINITELY for this app. **Substitute: CloudWatch alarms.**

For a ministry-scale app, a real-carrier production smoke test is overkill relative to
what CloudWatch already gives you for free:

- **Pinpoint delivery rate alarm**: Set a CloudWatch alarm on the
  `AWS/SMSVoice.TextMessageDeliveryRate` metric in the phone pool's dimension. Alerts on
  carrier-side failures (the thing a smoke test would catch) without any test
  infrastructure.
- **SNS topic message count alarm**: An anomaly-detection alarm on inbound SNS topic
  message count catches "Pinpoint -> SNS plumbing broken" silently.
- **Lambda error rate + duration alarms**: Standard issue, catches both code-level and
  infra-level regressions.

Roll Layer 5 only if an incident specifically motivated by carrier-side failure ever
happens. Until then, the alarms above are 90% of the signal at 0% of the maintenance
burden.

The original spec is preserved below for reference.

**Original priority (preserved for context)**: FUTURE -- Only after Layers 1-3 are solid.

**Runs in**: Weekly or on deploy (triggered by deployment pipeline)

#### Concept

A canary test using a real dedicated test phone number to verify the full production path.

#### Approach Options

| Option | How It Works | Cost |
|--------|-------------|------|
| Mailosaur | Provides a phone number + API to read received SMS. Send SMS to app, check Mailosaur inbox for response. | ~$100/mo |
| Dedicated test device + API | Physical phone or SMS gateway that receives messages and exposes them via API. | Hardware + carrier plan |
| AWS Pinpoint + CloudWatch | Send test SMS, verify delivery via CloudWatch SMS delivery logs (no content verification). | Minimal |

#### Recommended Approach: Mailosaur

1. Provision a Mailosaur phone number
2. Register it as a member in the production app
3. Scheduled test sends an SMS from the Mailosaur number (or publishes directly to SNS)
4. Poll Mailosaur API for the expected response SMS
5. Assert response content matches expected template
6. Alert on failure (PagerDuty, Slack, email)

#### What This Validates

- The complete production path: real carrier -> Pinpoint -> SNS -> Lambda -> DynamoDB ->
  Pinpoint -> real carrier
- Infrastructure health (no silent failures)
- Pinpoint phone pool validity and carrier routing
- End-user experience

#### Limitations

- Slow (carrier delivery can take seconds to minutes)
- Flaky (carrier issues, rate limiting, delivery delays)
- Expensive relative to other layers
- Cannot run in parallel (shared production state)

---

## Priority Matrix

Updated 2026-05-24 to reflect implementation progress and the Layer 4 Gameplan A/B split.

| Layer | Status | ROI | Effort | Runs In | Catches |
|-------|--------|-----|--------|---------|---------|
| 2a. PinpointSender + SNS payload fixture | Done | Very High | Very Low (~half day) | CI, every PR | Retry logic bugs, inbound payload drift |
| 1. Testcontainers integration | Done | Very High | Medium (2-3 days) | CI, every PR | Rollback/concurrency bugs, multi-step flow bugs, state machine regressions |
| 2b. Handler refactor + handler tests | Done | Medium | Medium (1-2 days) | CI, every PR | Entry point parsing, multi-record SNS handling |
| 3. SAM local automation | **Skipped** | — | — | — | Layer 1 covers it better; skip the seed scripts |
| 4. Cloud E2E — template parameterization (prerequisite) | Planned | Required | Low (2-3 hours) | One-time | Unblocks Gameplans A and B both |
| 4. Cloud E2E — Gameplan A (staging) | **Recommended next** | High | Medium (~3 days) | Push to main + manual | IAM, real SNS->Lambda, real Pinpoint, EventBridge scheduling |
| 4. Cloud E2E — Gameplan B (ephemeral) | Deferred | High | Higher (~5 days) | Nightly + manual | Everything Gameplan A catches, plus CloudFormation template-change validation per run |
| 5. Production smoke | **Deferred indefinitely** | Low (substitute: CloudWatch alarms) | High (2-3 days + ongoing carrier cost) | Weekly | Full carrier path validation |

---

## Recommendation

### What's done (2026-05-24)

Layers 2a, 1, and 2b are complete on branch `test-enhancements` — ~40 integration tests
against real DynamoDB Local, full `PinpointSender` retry coverage, the inbound SNS payload
fixture, and a handler refactor (`processRecords` / `parseRequest`) with dedicated unit
tests. Together these cover the majority of regressions before code reaches AWS.

Layer 3 was skipped. Layer 1 covers the same surface area with better tooling, and
`dev/addmembers.sh` already handles the only remaining use case (manual seed via curl).

### What's next

**1. Template parameterization (~2-3 hours).** Add a `StackPrefix` parameter to all three
`deploy/*` templates so a non-prod stack can coexist with prod in the same account. This
is the prerequisite for everything in Layer 4. Smallest diff with the biggest unlock — do
this even if no further E2E work is committed to.

**2. Layer 4, Gameplan A (~3 days).** Stand up a permanent staging stack triplet + the
test harness (SMS configuration set, event-destination SNS topic, collector Lambda,
capture DynamoDB table). Add `test/e2e/` with corrId-based assertions. Gate on push to
main + manual dispatch via GitHub OIDC. This catches the structural gaps Layer 1 cannot:
IAM, real SNS->Lambda, real Pinpoint API behavior, EventBridge scheduling, template
correctness for staging.

**3. CloudWatch alarms (~1 hour, in lieu of Layer 5).** Pinpoint delivery rate, SNS topic
anomaly-detection on inbound count, Lambda error rate + duration. These cover the
production-health monitoring intent of Layer 5 at zero ongoing maintenance cost.

### What to skip or defer

- **Layer 4 Gameplan B (ephemeral stacks)**: defer until staging reveals a need. The
  scenarios that motivate ephemeral over staging — frequent CloudFormation template
  changes, multi-developer template-change conflicts, deploy-correctness compliance —
  don't apply to a single-dev ministry app yet.
- **Layer 5 (Mailosaur / real carrier)**: defer indefinitely. CloudWatch alarms above
  substitute the regression-prevention value at fraction of the cost.

### The one-line summary

Layer 4 Gameplan A is the highest-value next investment. Everything else either is
already done, has a cheaper substitute, or doesn't apply at this app's scale.

---

## Research References

References added in the 2026-05-24 revision are marked with **(new)**.

### Testcontainers for Go
- [testcontainers-go DynamoDB module](https://golang.testcontainers.org/modules/dynamodb/)
- [testcontainers-go LocalStack module](https://golang.testcontainers.org/modules/localstack/)
- [dynamotest (ory/dockertest)](https://github.com/upsidr/dynamotest)

### AWS Official Guidance
- [AWS Prescriptive Guidance: Best practices for testing serverless applications](https://docs.aws.amazon.com/prescriptive-guidance/latest/serverless-application-testing/best-practices.html) — the canonical AWS recommendation: *"focus on testing in the cloud as much as possible"*; introduces the test harness pattern. **(new emphasis)**
- [AWS Prescriptive Guidance: Testing serverless applications on AWS (introduction)](https://docs.aws.amazon.com/prescriptive-guidance/latest/serverless-application-testing/introduction.html) **(new)**
- [AWS Lambda Testing Guide](https://docs.aws.amazon.com/lambda/latest/dg/testing-guide.html)
- [Introducing the AWS Integrated Application Test Kit (IATK)](https://aws.amazon.com/blogs/compute/aws-integrated-application-test-kit/) — AWS's own Python implementation of the test-harness pattern Layer 4 uses. **(new)**
- [aws-samples/serverless-test-samples (GitHub)](https://github.com/aws-samples/serverless-test-samples) — the official multi-language test patterns repo. No Go samples. **(new)**

### AWS End User Messaging SMS V2
- [Simulator phone numbers](https://docs.aws.amazon.com/sms-voice/latest/userguide/test-phone-numbers.html) — origination + destination, free, stay inside AWS.
- [Event destinations (CloudWatch / Firehose / SNS)](https://docs.aws.amazon.com/sms-voice/latest/userguide/configuration-sets-event-destinations.html) — the mechanism that makes outbound-SMS assertions tractable. **(new — central to Layer 4)**
- [CreateEventDestination API reference](https://docs.aws.amazon.com/pinpoint/latest/apireference_smsvoicev2/API_CreateEventDestination.html) **(new)**
- [SendTextMessage API reference](https://docs.aws.amazon.com/pinpoint/latest/apireference_smsvoicev2/API_SendTextMessage.html) **(new)**
- [How to send SMS using configuration sets (AWS Messaging Blog)](https://aws.amazon.com/blogs/messaging-and-targeting/how-to-send-sms-using-configurations-sets-with-amazon-pinpoint/) **(new)**

### Serverless Testing — Practitioners
- [Yan Cui: My testing strategy for serverless applications](https://theburningmonk.com/2022/05/my-testing-strategy-for-serverless-applications/) — ephemeral stacks + "remocal" testing. **(new)**
- [Yan Cui: How to include SNS and Kinesis in your e2e tests](https://theburningmonk.com/2019/09/how-to-include-sns-and-kinesis-in-your-e2e-tests/) — the listener-Lambda + DynamoDB capture pattern that Layer 4 adopts. **(new)**
- [Serverless Testing Pyramid (Ran the Builder)](https://www.ranthebuilder.cloud/post/serverless-testing-lambda-pyramid)
- [Guide to Serverless & Lambda Testing — Part 1 (Ran the Builder)](https://ranthebuilder.cloud/blog/guide-to-serverless-lambda-testing-best-practices-part-1/) **(new)**
- [Testable Lambdas in Go](https://dev.to/prozz/serverless-in-go-how-to-write-testable-lambdas-4925)
- [Contract Testing with Pact](https://docs.pact.io/)

### AWS SAM and Deployment
- [SAM Accelerate / sam sync](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/using-sam-cli-sync.html) — fast inner-loop cloud testing during dev. **(new)**
- [Accelerating serverless development with AWS SAM Accelerate (AWS Compute Blog)](https://aws.amazon.com/blogs/compute/accelerating-serverless-development-with-aws-sam-accelerate/) **(new)**
- [SAM local invoke](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/using-sam-cli-local-invoke.html)
- [sam delete (SAM CLI reference)](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/sam-cli-command-reference-sam-delete.html) **(new)**

### CloudFormation Ephemeral Stack Mechanics
- [CloudFormation export-name parameterization (Medium)](https://medium.com/@geoff.ford_33546/cloudformation-export-management-2f25b2d09443) **(new)**
- [Dynamic bindings for CloudFormation stacks (dev.to)](https://dev.to/lambdasharp/dynamic-bindings-for-cloudformation-stacks-15l6) **(new)**
- [Preventing CloudFormation orphaned resources](https://www.serverlesssam.com/p/cleanup-cloudformation/) **(new)**
- [Zero orphaned resources: force deleting any CloudFormation stack](https://dev.to/aws-heroes/zero-orphaned-resources-force-deleting-any-cloudformation-stack-3b1h) **(new)**

### CI / GitHub Actions ↔ AWS
- [aws-actions/configure-aws-credentials (GitHub)](https://github.com/aws-actions/configure-aws-credentials) **(new)**
- [AWS Security Blog: Use IAM roles to connect GitHub Actions to AWS (OIDC)](https://aws.amazon.com/blogs/security/use-iam-roles-to-connect-github-actions-to-actions-in-aws/) **(new)**
- [GitHub Docs: OpenID Connect](https://docs.github.com/en/actions/concepts/security/openid-connect) **(new)**

### Emulation (Background; Not Used Here)
- [LocalStack](https://github.com/localstack/localstack)
- [LocalStack Pinpoint coverage](https://docs.localstack.cloud/aws/services/pinpoint/) — note: Pinpoint v1 retires 2026-10-30; LocalStack coverage of SMS V2 is uncertain. **(new)**
- [LocalStack testcontainers-go integration](https://golang.testcontainers.org/modules/localstack/) **(new)**
- [SAM local vs LocalStack](https://blog.localstack.cloud/testing-serverless-apps-locally-aws-sam-local-vs-localstack/)
- [MiniStack (lightweight alternative)](https://repost.aws/articles/AR-WRCanAJScabqnY2OQ5H7A/local-lambda-testing-with-ministack-sam-cli-and-finch)

### Real-Carrier SMS Testing (Layer 5; Deferred)
- [Mailosaur SMS Testing](https://mailosaur.com/sms-testing)
- [Sinch SMS Testing Sandbox](https://sinch.com/messaging/sms-api/testing-sandbox/)
