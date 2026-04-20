package utility_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/utility"
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
			got := utility.WrapError(tt.err, tt.msg)
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

			got := utility.LogAndWrapError(context.Background(), tt.err, tt.msg, "testattr1", "1", "testattr2", "2")
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
