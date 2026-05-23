# Dissolve `internal/utility/` Package — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the grab-bag `internal/utility/` package with focused, idiomatically-named packages and move single-consumer items to their consuming packages.

**Architecture:** Create `internal/apperr/` for cross-cutting error helpers, `internal/awscfg/` for AWS configuration, move domain errors and `GenerateID` into `service/`, and inline `IsAwsLocal` at its single call site. Then delete `internal/utility/`.

**Tech Stack:** Go 1.25, AWS SDK v2, testify

---

### Task 1: Create `internal/apperr/` package

**Files:**
- Create: `internal/apperr/apperr.go`
- Create: `internal/apperr/apperr_test.go`

- [ ] **Step 1: Create `internal/apperr/apperr.go`**

```go
package apperr

import (
	"context"
	"fmt"
	"log/slog"
)

func LogAndWrapError(ctx context.Context, err error, message string, attrs ...any) error {
	if err == nil {
		return nil
	}
	slog.ErrorContext(ctx, message, append(attrs, "error", err)...)
	return fmt.Errorf("%s: %w", message, err)
}

func LogError(ctx context.Context, err error, message string, attrs ...any) {
	slog.ErrorContext(ctx, message, append(attrs, "error", err)...)
}

func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
```

- [ ] **Step 2: Create `internal/apperr/apperr_test.go`**

```go
package apperr_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/apperr"
)

func TestWrapError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		msg     string
		wantNil bool
		wantStr string
	}{
		{
			name:    "wraps non-nil error",
			err:     errors.New("original error"),
			msg:     "wrapped new error message",
			wantStr: "wrapped new error message: original error",
		},
		{
			name:    "nil error returns nil",
			err:     nil,
			msg:     "wrapped new error message",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := apperr.WrapError(tt.err, tt.msg)
			if tt.wantNil {
				if got != nil {
					t.Errorf("WrapError(nil, %q) = %v, want nil", tt.msg, got)
				}
				return
			}
			if got.Error() != tt.wantStr {
				t.Errorf("WrapError() = %q, want %q", got.Error(), tt.wantStr)
			}
		})
	}
}

func TestLogAndWrapError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		msg        string
		wantNil    bool
		wantStr    string
		wantLogged string
	}{
		{
			name:       "wraps and logs non-nil error",
			err:        errors.New("original error"),
			msg:        "wrapped new error message",
			wantStr:    "wrapped new error message: original error",
			wantLogged: `level=ERROR msg="wrapped new error message" testattr1=1 testattr2=2 error="original error"`,
		},
		{
			name:    "nil error returns nil without logging",
			err:     nil,
			msg:     "wrapped new error message",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			orig := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
			t.Cleanup(func() { slog.SetDefault(orig) })

			got := apperr.LogAndWrapError(context.Background(), tt.err, tt.msg, "testattr1", "1", "testattr2", "2")
			if tt.wantNil {
				if got != nil {
					t.Errorf("LogAndWrapError(nil, %q) = %v, want nil", tt.msg, got)
				}
				if buf.Len() != 0 {
					t.Errorf("expected no log output, got %q", buf.String())
				}
				return
			}
			if got.Error() != tt.wantStr {
				t.Errorf("LogAndWrapError() = %q, want %q", got.Error(), tt.wantStr)
			}
			if !strings.Contains(buf.String(), tt.wantLogged) {
				t.Errorf("log output %q does not contain %q", buf.String(), tt.wantLogged)
			}
		})
	}
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/apperr/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/apperr/
git commit --no-gpg-sign -m "refactor: create internal/apperr package with error helpers"
```

---

### Task 2: Create `internal/awscfg/` package

**Files:**
- Create: `internal/awscfg/awscfg.go`
- Create: `internal/awscfg/retrylogger.go`

- [ ] **Step 1: Create `internal/awscfg/retrylogger.go`**

```go
package awscfg

import (
	"context"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

type loggingRetryer struct {
	delegate aws.Retryer
}

func (r *loggingRetryer) IsErrorRetryable(err error) bool {
	return r.delegate.IsErrorRetryable(err)
}

func (r *loggingRetryer) MaxAttempts() int {
	return r.delegate.MaxAttempts()
}

func (r *loggingRetryer) GetRetryToken(ctx context.Context, opErr error) (func(error) error, error) {
	return r.delegate.GetRetryToken(ctx, opErr)
}

func (r *loggingRetryer) GetInitialToken() func(error) error {
	return r.delegate.GetInitialToken()
}

func (r *loggingRetryer) RetryDelay(attempt int, opErr error) (time.Duration, error) {
	delay, calcErr := r.delegate.RetryDelay(attempt, opErr)
	slog.Warn("AWS retry", "attempt", attempt, "error", opErr, "delay", delay)
	return delay, calcErr
}
```

- [ ] **Step 2: Create `internal/awscfg/awscfg.go`**

```go
package awscfg

import (
	"context"
	"os"
	"time"

	"github.com/4JesusApps/prayertexter/internal/apperr"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
)

const (
	defaultRegion      = "us-west-1"
	defaultMaxRetry    = 5
	defaultMaxBackoff  = 10
)

func GetAwsConfig(ctx context.Context) (aws.Config, error) {
	region := defaultRegion
	if r := os.Getenv("PRAY_CONF_AWS_REGION"); r != "" {
		region = r
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region),
		config.WithRetryer(func() aws.Retryer {
			retryer := retry.NewStandard(func(o *retry.StandardOptions) {
				o.MaxAttempts = defaultMaxRetry
				o.MaxBackoff = time.Duration(defaultMaxBackoff) * time.Second
			})
			return &loggingRetryer{delegate: retryer}
		}))

	return cfg, apperr.WrapError(err, "failed to get aws config")
}
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./internal/awscfg/...`
Expected: Success (no output)

- [ ] **Step 4: Commit**

```bash
git add internal/awscfg/
git commit --no-gpg-sign -m "refactor: create internal/awscfg package for AWS config"
```

---

### Task 3: Create `service/errors.go` and `service/id.go`

**Files:**
- Create: `internal/service/errors.go`
- Create: `internal/service/id.go`
- Create: `internal/service/id_test.go`

- [ ] **Step 1: Create `internal/service/errors.go`**

```go
package service

type constError string

func (err constError) Error() string {
	return string(err)
}

const (
	ErrNoAvailableIntercessors = constError("no available intercessors")
	ErrIntercessorUnavailable  = constError("intercessor unavailable")
	ErrInvalidPhone            = constError("no valid phone numbers found")
)
```

- [ ] **Step 2: Create `internal/service/id.go`**

```go
package service

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/4JesusApps/prayertexter/internal/apperr"
)

func generateID() (string, error) {
	size := 16
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", apperr.WrapError(err, "failed generate ID")
	}

	return hex.EncodeToString(bytes), nil
}
```

Note: `generateID` is unexported because it's only used within the `service` package. This is a tighter API surface than the old `utility.GenerateID`.

- [ ] **Step 3: Create `internal/service/id_test.go`**

```go
package service

import "testing"

func TestGenerateID(t *testing.T) {
	t.Run("generate id and confirm basic details", func(t *testing.T) {
		id, err := generateID()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(id) != 32 {
			t.Errorf("expected string of 32 length, got %v", id)
		}
	})
}
```

Note: This is an internal (white-box) test (`package service` not `package service_test`) because `generateID` is now unexported.

- [ ] **Step 4: Verify it compiles and tests pass**

Run: `go test ./internal/service/ -run TestGenerateID -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/errors.go internal/service/id.go internal/service/id_test.go
git commit --no-gpg-sign -m "refactor: add service-local errors and generateID"
```

---

### Task 4: Update all consumers to use new packages

**Files:**
- Modify: `internal/service/prayer.go` — replace `utility` with `apperr` and local symbols
- Modify: `internal/service/member.go` — replace `utility.GenerateID` with local `generateID`, remove `utility` import
- Modify: `internal/service/admin.go` — replace `utility.ErrInvalidPhone` with local `ErrInvalidPhone`, remove `utility` import
- Modify: `internal/service/router.go` — replace `utility` with `apperr`
- Modify: `internal/service/prayer_test.go` — replace `utility.ErrNoAvailableIntercessors` with `service.ErrNoAvailableIntercessors`
- Modify: `internal/messaging/pinpoint.go` — replace `utility` with `apperr`, inline `IsAwsLocal`
- Modify: `internal/repository/dynamodb.go` — replace `utility` with `apperr`
- Modify: `cmd/prayertexter/main.go` — replace `utility` with `awscfg`
- Modify: `cmd/statecontroller/main.go` — replace `utility` with `awscfg`
- Modify: `dev/prayertexter/main.go` — replace `utility` with `awscfg`

- [ ] **Step 1: Update `internal/service/prayer.go`**

Replace the import:
```go
// Old:
	"github.com/4JesusApps/prayertexter/internal/utility"

// New:
	"github.com/4JesusApps/prayertexter/internal/apperr"
```

Replace all usages (8 occurrences):
- `utility.ErrNoAvailableIntercessors` → `ErrNoAvailableIntercessors` (3 occurrences)
- `utility.ErrIntercessorUnavailable` → `ErrIntercessorUnavailable` (3 occurrences)
- `utility.WrapError` → `apperr.WrapError` (4 occurrences)
- `utility.LogError` → `apperr.LogError` (2 occurrences)
- `utility.GenerateID()` → `generateID()` (1 occurrence)

- [ ] **Step 2: Update `internal/service/member.go`**

Remove the `utility` import entirely. Replace:
- `utility.GenerateID()` → `generateID()` (1 occurrence at line 88)

- [ ] **Step 3: Update `internal/service/admin.go`**

Remove the `utility` import entirely. Replace:
- `utility.ErrInvalidPhone` → `ErrInvalidPhone` (2 occurrences at lines 42 and 79)

- [ ] **Step 4: Update `internal/service/router.go`**

Replace the import:
```go
// Old:
	"github.com/4JesusApps/prayertexter/internal/utility"

// New:
	"github.com/4JesusApps/prayertexter/internal/apperr"
```

Replace all usages:
- `utility.LogAndWrapError` → `apperr.LogAndWrapError` (3 occurrences)

- [ ] **Step 5: Update `internal/service/prayer_test.go`**

Replace the import:
```go
// Old:
	"github.com/4JesusApps/prayertexter/internal/utility"

// New:
	"github.com/4JesusApps/prayertexter/internal/service"
```

Replace all usages:
- `utility.ErrNoAvailableIntercessors` → `service.ErrNoAvailableIntercessors` (2 occurrences at lines 161 and 180)

- [ ] **Step 6: Update `internal/messaging/pinpoint.go`**

Replace the import:
```go
// Old:
	"github.com/4JesusApps/prayertexter/internal/utility"

// New:
	"os"

	"github.com/4JesusApps/prayertexter/internal/apperr"
```

Replace usages:
- `utility.IsAwsLocal()` → `os.Getenv("AWS_SAM_LOCAL") == "true"` (1 occurrence at line 38)
- `utility.LogAndWrapError` → `apperr.LogAndWrapError` (1 occurrence at line 74)

Note: `"os"` is a new import needed for the inlined `IsAwsLocal` check.

- [ ] **Step 7: Update `internal/repository/dynamodb.go`**

Replace the import:
```go
// Old:
	"github.com/4JesusApps/prayertexter/internal/utility"

// New:
	"github.com/4JesusApps/prayertexter/internal/apperr"
```

Replace all usages:
- `utility.WrapError` → `apperr.WrapError` (5 occurrences)

- [ ] **Step 8: Update `cmd/prayertexter/main.go`**

Replace the import:
```go
// Old:
	"github.com/4JesusApps/prayertexter/internal/utility"

// New:
	"github.com/4JesusApps/prayertexter/internal/awscfg"
```

Replace usage:
- `utility.GetAwsConfig(ctx)` → `awscfg.GetAwsConfig(ctx)` (1 occurrence at line 40)

- [ ] **Step 9: Update `cmd/statecontroller/main.go`**

Replace the import:
```go
// Old:
	"github.com/4JesusApps/prayertexter/internal/utility"

// New:
	"github.com/4JesusApps/prayertexter/internal/awscfg"
```

Replace usage:
- `utility.GetAwsConfig(ctx)` → `awscfg.GetAwsConfig(ctx)` (1 occurrence at line 24)

- [ ] **Step 10: Update `dev/prayertexter/main.go`**

Replace the import:
```go
// Old:
	"github.com/4JesusApps/prayertexter/internal/utility"

// New:
	"github.com/4JesusApps/prayertexter/internal/awscfg"
```

Replace usage:
- `utility.GetAwsConfig(ctx)` → `awscfg.GetAwsConfig(ctx)` (1 occurrence at line 39)

- [ ] **Step 11: Verify everything compiles and tests pass**

Run: `go build ./...`
Expected: Success (no output)

Run: `go test ./...`
Expected: All tests PASS

- [ ] **Step 12: Commit**

```bash
git add internal/service/prayer.go internal/service/member.go internal/service/admin.go internal/service/router.go internal/service/prayer_test.go internal/messaging/pinpoint.go internal/repository/dynamodb.go cmd/prayertexter/main.go cmd/statecontroller/main.go dev/prayertexter/main.go
git commit --no-gpg-sign -m "refactor: update all imports from utility to apperr/awscfg/local"
```

---

### Task 5: Delete `internal/utility/`

**Files:**
- Delete: `internal/utility/` (entire directory)

- [ ] **Step 1: Delete the utility directory**

```bash
rm -rf internal/utility/
```

- [ ] **Step 2: Verify no references remain**

Run: `grep -r "internal/utility" --include="*.go" .`
Expected: No output (zero matches)

- [ ] **Step 3: Verify everything still compiles and tests pass**

Run: `go build ./...`
Expected: Success (no output)

Run: `go test ./...`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add -A internal/utility/
git commit --no-gpg-sign -m "refactor: delete internal/utility package"
```
