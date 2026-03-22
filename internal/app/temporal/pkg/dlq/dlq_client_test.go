package dlq

import "testing"

func TestShouldLogUnconfiguredDLQ(t *testing.T) {
	if !shouldLogUnconfiguredDLQ(1) {
		t.Fatalf("expected first attempt to log")
	}
	if shouldLogUnconfiguredDLQ(2) {
		t.Fatalf("expected retries after first attempt to stay quiet")
	}
}
