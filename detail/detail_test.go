package detail

import (
	"encoding/json"
	"testing"
)

func TestRepairTruncatedJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantValid bool // should result be valid JSON?
	}{
		{
			name:      "no truncation",
			input:     `{"foo": "bar"}`,
			wantValid: true,
		},
		{
			name:      "truncated in string",
			input:     `{"foo": "bar--truncated--`,
			wantValid: true,
		},
		{
			name:      "truncated in array",
			input:     `{"items": [1, 2, 3--truncated--`,
			wantValid: true,
		},
		{
			name:      "truncated nested objects",
			input:     `{"a": {"b": {"c": "d--truncated--`,
			wantValid: true,
		},
		{
			name:      "truncated mixed nesting",
			input:     `{"arr": [{"key": "val--truncated--`,
			wantValid: true,
		},
		{
			name:      "real world body",
			input:     `{"id":"abc","payload":{"area":{"site_id":"main"},"fields":{"tag_number":["0,1,3,--truncated--`,
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repairTruncatedJSON(tt.input)

			var parsed any
			err := json.Unmarshal([]byte(result), &parsed)

			if tt.wantValid && err != nil {
				t.Errorf("expected valid JSON, got error: %v\nresult: %s", err, result)
			}
			if !tt.wantValid && err == nil {
				t.Errorf("expected invalid JSON, but it parsed successfully\nresult: %s", result)
			}
		})
	}
}
