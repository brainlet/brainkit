// Ported from: packages/ai/src/error/verify-no-object-generated-error.ts
package aierror

import "testing"

// VerifyNoObjectGeneratedErrorExpected holds the expected values for verifying a NoObjectGeneratedError.
type VerifyNoObjectGeneratedErrorExpected struct {
	Message      string
	Response     LanguageModelResponseMetadata
	Usage        LanguageModelUsage
	FinishReason FinishReason
}

// VerifyNoObjectGeneratedError is a test helper that asserts the given error is
// a NoObjectGeneratedError with the expected field values.
func VerifyNoObjectGeneratedError(t *testing.T, err error, expected VerifyNoObjectGeneratedErrorExpected) {
	t.Helper()

	if !IsNoObjectGeneratedError(err) {
		t.Fatalf("expected error to be NoObjectGeneratedError, got %T", err)
	}

	noObjErr := err.(*NoObjectGeneratedError)

	if noObjErr.Message != expected.Message {
		t.Errorf("expected message %q, got %q", expected.Message, noObjErr.Message)
	}

	if noObjErr.Response == nil {
		t.Fatal("expected Response to be non-nil")
	}
	if noObjErr.Response.ID != expected.Response.ID {
		t.Errorf("expected response ID %q, got %q", expected.Response.ID, noObjErr.Response.ID)
	}
	if noObjErr.Response.ModelID != expected.Response.ModelID {
		t.Errorf("expected response ModelID %q, got %q", expected.Response.ModelID, noObjErr.Response.ModelID)
	}
	if !noObjErr.Response.Timestamp.Equal(expected.Response.Timestamp) {
		t.Errorf("expected response Timestamp %v, got %v", expected.Response.Timestamp, noObjErr.Response.Timestamp)
	}

	if noObjErr.Usage == nil {
		t.Fatal("expected Usage to be non-nil")
	}

	if noObjErr.FinishReason == nil {
		t.Fatal("expected FinishReason to be non-nil")
	}
	if *noObjErr.FinishReason != expected.FinishReason {
		t.Errorf("expected FinishReason %q, got %q", expected.FinishReason, *noObjErr.FinishReason)
	}
}
