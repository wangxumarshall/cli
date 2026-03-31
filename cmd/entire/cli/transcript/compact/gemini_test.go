package compact

import (
	"testing"
)

func TestCompact_GeminiFixture(t *testing.T) {
	t.Parallel()
	assertFixtureTransform(t, agentOpts("gemini-cli"), "testdata/gemini_full.jsonl", "testdata/gemini_expected.jsonl")
}

func TestIsGeminiFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want bool
	}{
		{
			name: "valid gemini",
			in:   `{"sessionId":"s1","messages":[]}`,
			want: true,
		},
		{
			name: "opencode has info key",
			in:   `{"info":{"id":"s1"},"messages":[]}`,
			want: false,
		},
		{
			name: "JSONL not JSON object",
			in:   `{"type":"user","message":{}}` + "\n" + `{"type":"assistant","message":{}}`,
			want: false,
		},
		{
			name: "empty",
			in:   "",
			want: false,
		},
		{
			name: "missing messages key",
			in:   `{"sessionId":"s1"}`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isGeminiFormat([]byte(tt.in))
			if got != tt.want {
				t.Errorf("isGeminiFormat(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
