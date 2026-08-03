package message

import "testing"

func TestStaleTokenErrCode_AliasBackwardCompat(t *testing.T) {
	// The deprecated alias must remain equal so external importers keep compiling.
	if StaleTokenErrCode != SessionExpiredErrCode {
		t.Fatalf("alias drift: StaleTokenErrCode=%d, SessionExpiredErrCode=%d",
			StaleTokenErrCode, SessionExpiredErrCode)
	}
	if StaleTokenErrCode != -14 {
		t.Fatalf("StaleTokenErrCode = %d, want -14", StaleTokenErrCode)
	}
	if SessionPauseDuration != 60*60*1e9 { // nanoseconds; 1 hour
		t.Fatalf("SessionPauseDuration = %v, want 1h", SessionPauseDuration)
	}
}

func TestIsStaleTokenError(t *testing.T) {
	// The monitor uses the literal "ret=-14" string to detect stale tokens.
	if !isStaleTokenError(errRetMinus14("ret=-14")) {
		t.Fatal("expected isStaleTokenError to match ret=-14")
	}
	if isStaleTokenError(errRetMinus14("ret=-99")) {
		t.Fatal("expected isStaleTokenError to NOT match ret=-99")
	}
	if isStaleTokenError(nil) {
		t.Fatal("nil error should not be a stale token error")
	}
}

// errRetMinus14 is a tiny stand-in error carrying the message format used by
// getUpdates failure strings.
type errRetMinus14 string

func (e errRetMinus14) Error() string { return string(e) }
