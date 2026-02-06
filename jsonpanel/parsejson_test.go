package jsonpanel

import (
	"encoding/json"
	"testing"
)

func TestRepairTruncatedJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool   // should result be valid JSON?
		wantPath  string // expected truncation path
	}{
		{
			name:      "no truncation",
			input:     `{"foo": "bar"}`,
			wantValid: true,
			wantPath:  "",
		},
		{
			name:      "truncated in string",
			input:     `{"foo": "bar--truncated--`,
			wantValid: true,
			wantPath:  "foo",
		},
		{
			name:      "truncated in array",
			input:     `{"items": [1, 2, 3--truncated--`,
			wantValid: true,
			wantPath:  "items",
		},
		{
			name:      "truncated nested objects",
			input:     `{"a": {"b": {"c": "d--truncated--`,
			wantValid: true,
			wantPath:  "a.b.c",
		},
		{
			name:      "truncated mixed nesting",
			input:     `{"arr": [{"key": "val--truncated--`,
			wantValid: true,
			wantPath:  "arr.key",
		},
		{
			name:      "real world body",
			input:     `{"id":"abc","payload":{"area":{"site_id":"main"},"fields":{"tag_number":["0,1,3,--truncated--`,
			wantValid: true,
			wantPath:  "payload.fields.tag_number",
		},
		{
			name:      "truncated mid-key",
			input:     `{"payload":{"fields":{"wlan_m--truncated--`,
			wantValid: true,
			wantPath:  "payload.fields.wlan_m",
		},
		{
			name:      "truncated mid-number",
			input:     `{"count": 123--truncated--`,
			wantValid: true,
			wantPath:  "count",
		},
		{
			name:      "truncated with escapes",
			input:     `{"msg": "hello \"world--truncated--`,
			wantValid: true,
			wantPath:  "msg",
		},
		{
			name:      "truncated after comma in object",
			input:     `{"a": 1,--truncated--`,
			wantValid: true,
			wantPath:  "",
		},
		{
			name:      "empty object truncated",
			input:     `{--truncated--`,
			wantValid: true,
			wantPath:  "",
		},
		{
			name:      "root level array",
			input:     `["a", "b--truncated--`,
			wantValid: true,
			wantPath:  "",
		},
		{
			name:      "escaped backslash",
			input:     `{"path": "c:\\users\\--truncated--`,
			wantValid: true,
			wantPath:  "path",
		},
		{
			name:      "whitespace in JSON",
			input:     `{ "key" : "val--truncated--`,
			wantValid: true,
			wantPath:  "key",
		},
		{
			name:      "boolean truncation",
			input:     `{"flag": tru--truncated--`,
			wantValid: true,
			wantPath:  "flag",
		},
		{
			name:      "null truncation",
			input:     `{"val": nul--truncated--`,
			wantValid: true,
			wantPath:  "val",
		},
		{
			name:      "deeply nested arrays",
			input:     `{"a":[[["val--truncated--`,
			wantValid: true,
			wantPath:  "a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, truncPath := repairTruncatedJSON(tt.input)

			var parsed any
			err := json.Unmarshal([]byte(result), &parsed)

			if tt.wantValid && err != nil {
				t.Errorf("expected valid JSON, got error: %v\nresult: %s", err, result)
			}
			if !tt.wantValid && err == nil {
				t.Errorf("expected invalid JSON, but it parsed successfully\nresult: %s", result)
			}
			if truncPath != tt.wantPath {
				t.Errorf("expected path %q, got %q", tt.wantPath, truncPath)
			}
		})
	}
}
