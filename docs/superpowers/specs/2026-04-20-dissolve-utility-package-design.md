# Dissolve `internal/utility/` Package

## Problem

The `internal/utility/` package is a grab-bag that violates Go's convention of naming packages by what they provide. It contains four unrelated concerns: AWS configuration, error helpers, domain-specific errors, and ID generation.

## Approach

Partial dissolution. Move single-consumer items to their consuming packages. Keep only the genuinely cross-cutting error helpers in a small, focused package with a descriptive name. Create a dedicated package for AWS configuration.

## Changes

### New package: `internal/apperr/`

Cross-cutting error helper functions used by multiple packages:

- `WrapError(err error, message string) error`
- `LogAndWrapError(ctx context.Context, err error, message string, attrs ...any) error`
- `LogError(ctx context.Context, err error, message string, attrs ...any)`
- The unexported `constError` type stays here only if needed; otherwise each consumer defines its own

Existing tests from `utility/errors_test.go` move here.

### New package: `internal/awscfg/`

AWS configuration loading, used by `cmd/` entry points:

- `GetAwsConfig(ctx context.Context) (aws.Config, error)`
- `LoggingRetryer` struct (unexported — internal to package)
- `defaultAwsRegion`, `defaultAwsSvcRetryAttempts`, `defaultAwsSvcMaxBackoff` constants (unexported — only used within `GetAwsConfig`)

### Moved to `service/`

Items used exclusively within the service package:

- **`service/errors.go`**: `constError` type, `ErrNoAvailableIntercessors`, `ErrIntercessorUnavailable`, `ErrInvalidPhone`
- **`service/id.go`**: `GenerateID()` function
- **`service/id_test.go`**: Existing `GenerateID` test

### Inlined in `messaging/pinpoint.go`

- `IsAwsLocal()` — a one-liner (`os.Getenv("AWS_SAM_LOCAL") == "true"`) used only in `pinpoint.go`. Inline at the call site and remove as a standalone function.

### Deleted

- `internal/utility/` — entire directory removed

### Import updates

| File | Old import | New import |
|------|-----------|------------|
| `cmd/prayertexter/main.go` | `utility` | `awscfg` |
| `cmd/statecontroller/main.go` | `utility` | `awscfg` |
| `dev/prayertexter/main.go` | `utility` | `awscfg` |
| `repository/dynamodb.go` | `utility` | `apperr` |
| `messaging/pinpoint.go` | `utility` | `apperr` (+ inline `IsAwsLocal`) |
| `service/router.go` | `utility` | `apperr` (+ local errors) |
| `service/prayer.go` | `utility` | `apperr` (+ local errors, local `GenerateID`) |
| `service/member.go` | `utility` | local `GenerateID` (remove `utility` import) |
| `service/admin.go` | `utility` | local `ErrInvalidPhone` (remove `utility` import) |
| `service/prayer_test.go` | `utility` | local `ErrNoAvailableIntercessors` (remove `utility` import) |

## Verification

- `go build ./...` passes
- `go test ./...` passes with no regressions
- `grep -r "internal/utility" .` returns zero results
