package utility_test

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/utility"
)

func TestErrorOperations(t *testing.T) {
	origErr := errors.New("original error")
	expectedErrString := "wrapped new error message: original error"
	newErrorMsg := "wrapped new error message"

	t.Run("WrapError", func(t *testing.T) {
		t.Run("basic error wrapping string test", func(t *testing.T) {
			testErr(t, origErr, expectedErrString, newErrorMsg, false)
		})

		t.Run("nil err as parameter should also return nil", func(t *testing.T) {
			testErr(t, origErr, expectedErrString, newErrorMsg, true)
		})
	})
	t.Run("LogAndWrapError", func(t *testing.T) {
		t.Run("basic error wrapping/logging string test", func(t *testing.T) {
			testErrAndLog(t, origErr, expectedErrString, newErrorMsg, false)
		})

		t.Run("nil err as parameter should also return nil and not log anything", func(t *testing.T) {
			testErrAndLog(t, origErr, expectedErrString, newErrorMsg, true)
		})
	})
}

func testErr(t *testing.T, origErr error, expectedErrString, newErrorMsg string, nilErr bool) {
	if nilErr {
		newErr := utility.WrapError(nil, newErrorMsg)

		if newErr != nil {
			t.Errorf("expected nil error, got %v", newErr.Error())
		}
	} else {
		newErr := utility.WrapError(origErr, newErrorMsg)

		if newErr.Error() != expectedErrString {
			t.Errorf("expected error string %v, got %v", expectedErrString, newErr.Error())
		}
	}
}

func testErrAndLog(t *testing.T, origErr error, expectedErrString, newErrorMsg string, nilErr bool) {
	var buff bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buff, nil))
	slog.SetDefault(log)
	expectedLog := `level=ERROR msg="wrapped new error message" testattr1=1 testattr2=2 error="original error"`

	if nilErr {
		testNilErrorAndLog(t, newErrorMsg, &buff)
	} else {
		testActualErrorAndLog(t, origErr, newErrorMsg, expectedErrString, expectedLog, &buff)
	}
}

func testNilErrorAndLog(t *testing.T, newErrorMsg string, buff *bytes.Buffer) {
	newErr := utility.LogAndWrapError(nil, newErrorMsg, "testattr1", "1", "testattr2", "2")

	if newErr != nil {
		t.Errorf("expected nil error, got %v", newErr.Error())
	}

	if buff.Len() != 0 {
		t.Errorf("expected no logging, got %v", buff.String())
	}
}

func testActualErrorAndLog(t *testing.T, origErr error, newErrorMsg, expectedErrString, expectedLog string,
	buff *bytes.Buffer) {
	newErr := utility.LogAndWrapError(origErr, newErrorMsg, "testattr1", "1", "testattr2", "2")

	if newErr.Error() != expectedErrString {
		t.Errorf("expected error string %v, got %v", expectedErrString, newErr.Error())
	}

	if !strings.Contains(buff.String(), expectedLog) {
		t.Errorf("expected string %v to contain substring %v", buff.String(), expectedLog)
	}
}
