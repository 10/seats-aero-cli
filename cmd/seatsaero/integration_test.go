//go:build integration

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

// Six read-only calls at most. Refresh intentionally has no live test: queued
// IDs spend additional credits and the endpoint has an hourly request cap.
func TestLiveAPI(t *testing.T) {
	if os.Getenv("SEATSAERO_API_KEY") == "" {
		t.Skip("set SEATSAERO_API_KEY to run live API checks")
	}
	call := func(t *testing.T, args ...string) []byte {
		t.Helper()
		var stdout, stderr bytes.Buffer
		if code := run(args, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
			// Do not put live account data or credentials in test logs.
			var envelope errorEnvelope
			_ = json.Unmarshal(stderr.Bytes(), &envelope)
			t.Fatalf("%s failed: exit=%d code=%s HTTP=%d", args[0], code, envelope.Error.Code, envelope.Error.Status)
		}
		if !json.Valid(stdout.Bytes()) {
			t.Fatalf("%s returned invalid JSON", args[0])
		}
		t.Logf("%s: HTTP 2xx, valid JSON (%d bytes)", args[0], stdout.Len())
		return stdout.Bytes()
	}
	object := func(t *testing.T, body []byte, keys ...string) map[string]json.RawMessage {
		t.Helper()
		var result map[string]json.RawMessage
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatal("expected a JSON object")
		}
		for _, key := range keys {
			if _, ok := result[key]; !ok {
				t.Fatalf("missing response field %q", key)
			}
		}
		return result
	}
	var availabilityID string
	t.Run("search", func(t *testing.T) {
		result := object(t, call(t, "search", "SFO", "NRT", "--take", "10"), "data", "hasMore", "cursor")
		var rows []struct{ ID string }
		if err := json.Unmarshal(result["data"], &rows); err != nil {
			t.Fatal("invalid search rows")
		}
		if len(rows) > 0 {
			availabilityID = rows[0].ID
		}
	})
	t.Run("availability", func(t *testing.T) {
		object(t, call(t, "availability", "--source", "aeroplan", "--take", "10"), "data", "hasMore", "cursor")
	})
	t.Run("trips", func(t *testing.T) {
		if availabilityID == "" {
			t.Skip("search returned no availability ID to drill into")
		}
		object(t, call(t, "trips", availabilityID), "booking_links")
	})
	t.Run("routes", func(t *testing.T) {
		body := bytes.TrimSpace(call(t, "routes", "--source", "aeroplan"))
		if len(body) == 0 || body[0] != '[' {
			t.Fatal("routes must return a bare array")
		}
	})
	t.Run("destinations", func(t *testing.T) {
		result := object(t, call(t, "destinations", "--origin-airport", "SFO"), "success")
		if string(result["success"]) != "true" {
			t.Fatal("destinations success must be true")
		}
	})
	t.Run("alerts", func(t *testing.T) {
		object(t, call(t, "alerts"), "data")
	})
}
