# PrayerTexter E2E & Integration Test Strategy

## Table of Contents

- [Current State](#current-state)
- [Architecture Observations](#architecture-observations)
- [Option Analysis](#option-analysis)
  - [Option A: End-User Messaging Simulator Number](#option-a-end-user-messaging-simulator-number)
  - [Option B: Expanding SAM Local Testing](#option-b-expanding-sam-local-testing)
  - [Option C: Multi-Layered Integration Tests (testcontainers-go)](#option-c-multi-layered-integration-tests-testcontainers-go)
- [Recommended Multi-Layer Strategy](#recommended-multi-layer-strategy)
  - [Layer 1: Integration Tests with Testcontainers](#layer-1-integration-tests-with-testcontainers)
  - [Layer 2: Handler & Pinpoint Unit Tests](#layer-2-handler--pinpoint-unit-tests)
  - [Layer 3: Automated SAM Local Smoke Tests](#layer-3-automated-sam-local-smoke-tests)
  - [Layer 4: Cloud E2E Tests](#layer-4-cloud-e2e-tests)
  - [Layer 5: Production Smoke Test](#layer-5-production-smoke-test)
- [Priority Matrix](#priority-matrix)
- [Recommendation](#recommendation)

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

**Priority**: MEDIUM -- Improves developer workflow.

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

**Priority**: LOWER -- Run periodically, not per-PR.

**Runs in**: Nightly or on release branches (scheduled GitHub Actions workflow)

#### Concept

Deploy an ephemeral SAM stack and run E2E tests against real AWS services.

#### Prerequisite: Cross-Stack Imports Block Ephemeral Deploys

The current `deploy/` layout has three separate stacks (`db`, `prayertexter`,
`statecontroller`) wired together with hardcoded CloudFormation exports/imports:

- `deploy/prayertexter/template.yaml` uses `!ImportValue db-MemberTableName`,
  `db-ActivePrayerTableName`, `db-GeneralTableName`, `db-QueuedPrayerTableName`
- `deploy/statecontroller/template.yaml` does the same and also imports
  `prayertexter-SMSPhonePoolARN`
- Export names are hardcoded with the prefixes `db-` and `prayertexter-`, so two
  ephemeral environments cannot coexist in the same account

A naive `sam deploy --stack-name pt-e2e-${sha}` against just the prayertexter template
will fail: the imports resolve against fixed names that point at the *real* db stack.

**Options to make ephemeral deploys viable (pick one before Layer 4 work starts):**

1. **Merge templates** into a single `deploy/e2e/template.yaml` with all resources in one
   stack. Simplest; loses the deploy-independence of the current layout.
2. **Parameterize the export prefix** so each stack can be deployed with a
   `--parameter-overrides StackPrefix=e2e-${sha}` and exports/imports use
   `!Sub "${StackPrefix}-db-MemberTableName"`. Preserves the three-stack layout; requires
   editing both the producer (`deploy/db/template.yaml`) and consumer templates.
3. **Deploy the full triplet per E2E run** with renamed exports. Most faithful to prod
   topology; slowest (three stack creations + waits).

Option 2 is the recommended path: it's the smallest diff that unblocks E2E without
collapsing the existing deploy structure.

#### Workflow

1. Deploy the db stack (or all three) with a per-run prefix (e.g.,
   `pt-e2e-${short-sha}-db`)
2. Deploy the prayertexter stack referencing those exports
3. Wait for stack creation to complete; discover resource ARNs from stack outputs
4. Run a Go test binary that:
   - Publishes test messages to the SNS topic via AWS SDK
   - Uses AWS SMS simulator phone numbers as destinations for Pinpoint sends
   - Queries DynamoDB tables to assert state changes
   - Invokes the StateController Lambda directly to test scheduled jobs
5. `sam delete` each stack in reverse dependency order to tear down

#### What This Validates (That Lower Layers Cannot)

- Real IAM permissions (Lambda execution role, DynamoDB policies, Pinpoint access)
- Real SNS -> Lambda event source mapping
- Real Pinpoint API integration (with simulator numbers)
- Real EventBridge -> StateController triggering
- CloudFormation template correctness
- Cold start behavior and timeouts

#### CI Configuration

```yaml
# .github/workflows/e2e-nightly.yml
name: Nightly E2E Tests
on:
  schedule:
    - cron: '0 6 * * *'  # 6am UTC daily
  workflow_dispatch: {}    # Allow manual trigger

jobs:
  e2e:
    runs-on: ubuntu-latest
    permissions:
      id-token: write
      contents: read
    steps:
      - uses: actions/checkout@v4
      - uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ secrets.E2E_ROLE_ARN }}
          aws-region: us-west-1
      - uses: aws-actions/setup-sam@v2
      - run: sam build -t deploy/prayertexter/template.yaml
      - run: sam deploy --stack-name pt-e2e-${{ github.sha }} --no-confirm-changeset --no-fail-on-empty-changeset
      - run: go test -v -tags e2e ./test/e2e/...
      - run: sam delete --stack-name pt-e2e-${{ github.sha }} --no-prompts
        if: always()
```

#### AWS Prerequisites

- Dedicated IAM role for E2E tests with scoped permissions
- Separate AWS account or at minimum a resource naming convention to avoid collisions
- Budget alarm to catch runaway costs

#### Proposed File Layout

```
test/e2e/
|-- e2e_test.go         -- TestMain: deploy stack, discover resource ARNs
|-- sns_publish_test.go -- Publish messages to SNS topic, assert DB state
|-- scheduler_test.go   -- Invoke StateController, assert DB state
|-- cleanup_test.go     -- Verify teardown
|-- helpers.go          -- AWS SDK wrappers for test assertions
```

---

### Layer 5: Production Smoke Test

**Priority**: FUTURE -- Only after Layers 1-3 are solid.

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

Layer 2 is split into 2a (quick wins) and 2b (handler refactor) because the cost and ROI
of each half are very different.

| Layer | ROI | Effort | Runs In | Catches |
|-------|-----|--------|---------|---------|
| 2a. PinpointSender + SNS payload fixture | Very High | Very Low (~half day) | CI, every PR | Retry logic bugs, inbound payload drift |
| 1. Testcontainers integration | Very High | Medium (2-3 days) | CI, every PR | Rollback/concurrency bugs, multi-step flow bugs, state machine regressions |
| 2b. Handler refactor + handler tests | Medium | Medium (1-2 days) | CI, every PR | Entry point parsing, multi-record SNS handling |
| 3. SAM local automation | Medium | Low (1 day) | Dev machine | Developer confidence, StateController manual runs |
| 4. Cloud E2E | Medium | High (3-5 days incl. template parameterization) | Nightly CI | IAM, infra config, real SNS->Lambda wiring |
| 5. Production smoke | Low (now) | High (2-3 days) | Weekly | Full carrier path validation |

---

## Recommendation

**Do Layer 2a first.** PinpointSender retry logic is the only untested piece of
timing-sensitive control flow in the app; a bug there silently drops user-visible SMS in
production. With the `PinpointClient` mockery prereq added, the test suite is ~30 minutes
of work. The SNS payload fixture is another hour and closes the inbound contract gap. This
is the cheapest, highest-confidence win on the board.

**Then Layer 1.** Testcontainers + DynamoDB Local fills the biggest *structural* gap: there
is currently no way to validate multi-step rollback paths, queue-to-active state
transitions, or concurrent writes to the shared `General` table without deploying to AWS.
Frame this work around flow/rollback correctness, not schema-drift detection -- the
repository layer's surface area for schema bugs is genuinely small.

**Then Layer 2b** (handler refactor) if and only if the entry-point coverage is judged
worth the wiring extraction. Many serverless shops accept ~0% on Lambda wiring code; the
*event-parsing* paths are the part worth covering, and those are already addressed by the
fixture test in Layer 2a.

After those three layers, the majority of regressions will be caught before code ever
reaches AWS. Layers 3-5 add incremental value:

- **Layer 3** is worthwhile only when Layer 1 lands first -- otherwise it duplicates Layer 1's
  coverage with worse tooling. Its unique value is enabling local StateController invocation.
- **Layer 4** has a hard prerequisite (template export parameterization) that should be
  scoped before the layer is committed to.
- **Layer 5** is a production-health concern, not a regression-prevention one, and should
  wait until Layers 1-3 are solid.

---

## Research References

### Testcontainers for Go
- [testcontainers-go DynamoDB module](https://golang.testcontainers.org/modules/dynamodb/)
- [testcontainers-go LocalStack module](https://golang.testcontainers.org/modules/localstack/)
- [dynamotest (ory/dockertest)](https://github.com/upsidr/dynamotest)

### AWS Testing
- [AWS SMS Simulator Phone Numbers](https://docs.aws.amazon.com/sms-voice/latest/userguide/test-phone-numbers.html)
- [AWS Serverless Testing Best Practices](https://docs.aws.amazon.com/prescriptive-guidance/latest/serverless-application-testing/best-practices.html)
- [AWS Lambda Testing Guide](https://docs.aws.amazon.com/lambda/latest/dg/testing-guide.html)
- [SAM local invoke](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/using-sam-cli-local-invoke.html)

### SAM Local and LocalStack
- [LocalStack](https://github.com/localstack/localstack)
- [SAM local vs LocalStack](https://blog.localstack.cloud/testing-serverless-apps-locally-aws-sam-local-vs-localstack/)
- [MiniStack (lightweight alternative)](https://repost.aws/articles/AR-WRCanAJScabqnY2OQ5H7A/local-lambda-testing-with-ministack-sam-cli-and-finch)

### SMS Testing
- [Mailosaur SMS Testing](https://mailosaur.com/sms-testing)
- [Sinch SMS Testing Sandbox](https://sinch.com/messaging/sms-api/testing-sandbox/)

### Serverless Testing Patterns
- [Serverless Testing Pyramid](https://www.ranthebuilder.cloud/post/serverless-testing-lambda-pyramid)
- [Testable Lambdas in Go](https://dev.to/prozz/serverless-in-go-how-to-write-testable-lambdas-4925)
- [Contract Testing with Pact](https://docs.pact.io/)
