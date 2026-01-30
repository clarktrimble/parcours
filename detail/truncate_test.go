package detail

import (
	"encoding/json"
	"testing"
)

func TestSmartTruncate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		limit     int
		wantValid bool
		wantTrunc bool // should contain truncation indicator
	}{
		{
			name:      "under limit",
			input:     `{"a": "short"}`,
			limit:     100,
			wantValid: true,
			wantTrunc: false,
		},
		{
			name:      "truncate after comma",
			input:     `{"a": "first", "b": "second", "c": "third"}`,
			limit:     20,
			wantValid: true,
			wantTrunc: true,
		},
		{
			name:      "truncate nested object",
			input:     `{"outer": {"inner": "value", "more": "data"}}`,
			limit:     30,
			wantValid: true,
			wantTrunc: true,
		},
		{
			name:      "truncate array",
			input:     `{"items": [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]}`,
			limit:     25,
			wantValid: true,
			wantTrunc: true,
		},
		{
			name:      "deeply nested",
			input:     `{"a": {"b": {"c": {"d": "deep value here"}}}}`,
			limit:     30,
			wantValid: true,
			wantTrunc: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SmartTruncate(tt.input, tt.limit)

			// Check valid JSON
			var parsed any
			err := json.Unmarshal([]byte(result), &parsed)
			if tt.wantValid && err != nil {
				t.Errorf("expected valid JSON, got error: %v\nresult: %s", err, result)
			}

			// Check truncation indicator
			hasTrunc := containsTruncation(result)
			if tt.wantTrunc && !hasTrunc {
				t.Errorf("expected truncation indicator, got: %s", result)
			}
			if !tt.wantTrunc && hasTrunc {
				t.Errorf("unexpected truncation indicator in: %s", result)
			}

			t.Logf("input:  %s", tt.input)
			t.Logf("result: %s", result)
		})
	}
}

func containsTruncation(s string) bool {
	return len(s) > 0 && (contains(s, "_truncated") || contains(s, "--truncated--"))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
