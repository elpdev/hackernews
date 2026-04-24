package doctor

import "testing"

func TestStatusString(t *testing.T) {
	if OK.String() != "OK" || Warn.String() != "WARN" || Fail.String() != "FAIL" {
		t.Fatalf("unexpected status strings: %s %s %s", OK, Warn, Fail)
	}
}

func TestSyncChecksWhenDisabled(t *testing.T) {
	checks := syncChecks(Options{})
	if len(checks) != 1 {
		t.Fatalf("expected one sync check, got %+v", checks)
	}
	if checks[0].Status != Warn || checks[0].Name != "sync" {
		t.Fatalf("unexpected disabled sync check: %+v", checks[0])
	}
}

func TestLookPathCheckWarnsWhenMissing(t *testing.T) {
	check := lookPathCheck("missing", []string{"definitely-not-a-hackernews-command"})
	if check.Status != Warn {
		t.Fatalf("expected warning for missing command, got %+v", check)
	}
}
