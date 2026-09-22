package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCustomTime_MarshalJSON(t *testing.T) {
	fixedTime := time.Date(2025, 1, 1, 15, 1, 4, 0, time.UTC)
	ct := NewCustomTime(fixedTime)

	bytes, err := json.Marshal(ct)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	expected := `"2025-01-01 15:01:04"`
	if string(bytes) != expected {
		t.Errorf("expected %s, got %s", expected, string(bytes))
	}
}

func TestCustomTime_MarshalJSON_NonUTC(t *testing.T) {
	loc := time.FixedZone("WIB", 7*3600)
	fixedTime := time.Date(2025, 1, 1, 22, 1, 4, 0, loc)
	ct := NewCustomTime(fixedTime)

	bytes, err := json.Marshal(ct)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	expected := `"2025-01-01 15:01:04"`
	if string(bytes) != expected {
		t.Errorf("expected %s in UTC, got %s", expected, string(bytes))
	}
}

func TestErrorResponse_Format(t *testing.T) {
	resp := NewErrorResponse("Validation failed", "title is required", "images must contain at least 1 item")

	bytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	if resp.Success != false {
		t.Errorf("expected Success to be false")
	}
	if resp.Message != "Validation failed" {
		t.Errorf("expected Message 'Validation failed', got '%s'", resp.Message)
	}
	if len(resp.Errors) != 2 {
		t.Errorf("expected 2 error items, got %d", len(resp.Errors))
	}

	_ = bytes
}
