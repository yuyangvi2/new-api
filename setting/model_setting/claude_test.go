package model_setting

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClaudeSettingsWriteHeadersMergesConfiguredValuesIntoSingleHeader(t *testing.T) {
	settings := &ClaudeSettings{
		HeadersSettings: map[string]map[string][]string{
			"claude-3-7-sonnet-20250219-thinking": {
				"anthropic-beta": {
					"token-efficient-tools-2025-02-19",
				},
			},
		},
	}

	headers := http.Header{}
	headers.Set("anthropic-beta", "output-128k-2025-02-19")

	settings.WriteHeaders("claude-3-7-sonnet-20250219-thinking", &headers)

	got := headers.Values("anthropic-beta")
	if len(got) != 1 {
		t.Fatalf("expected a single merged header value, got %v", got)
	}
	expected := "output-128k-2025-02-19,token-efficient-tools-2025-02-19"
	if got[0] != expected {
		t.Fatalf("expected merged header %q, got %q", expected, got[0])
	}
}

func TestClaudeSettingsWriteHeadersDeduplicatesAcrossCommaSeparatedAndRepeatedValues(t *testing.T) {
	settings := &ClaudeSettings{
		HeadersSettings: map[string]map[string][]string{
			"claude-3-7-sonnet-20250219-thinking": {
				"anthropic-beta": {
					"token-efficient-tools-2025-02-19",
					"computer-use-2025-01-24",
				},
			},
		},
	}

	headers := http.Header{}
	headers.Add("anthropic-beta", "output-128k-2025-02-19, token-efficient-tools-2025-02-19")
	headers.Add("anthropic-beta", "token-efficient-tools-2025-02-19")

	settings.WriteHeaders("claude-3-7-sonnet-20250219-thinking", &headers)

	got := headers.Values("anthropic-beta")
	if len(got) != 1 {
		t.Fatalf("expected duplicate values to collapse into one header, got %v", got)
	}
	expected := "output-128k-2025-02-19,token-efficient-tools-2025-02-19,computer-use-2025-01-24"
	if got[0] != expected {
		t.Fatalf("expected deduplicated merged header %q, got %q", expected, got[0])
	}
}

func TestShouldOmitClaudeSamplingParamsForNewClaudeModels(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  bool
	}{
		{name: "opus 4.6", model: "claude-opus-4-6", want: true},
		{name: "opus 4.6 suffix", model: "claude-opus-4-6-high", want: true},
		{name: "opus 4.7 models prefix", model: "models/claude-opus-4-7", want: true},
		{name: "opus 4.8 thinking", model: "claude-opus-4-8-thinking", want: true},
		{name: "sonnet 4.6", model: "claude-sonnet-4-6", want: true},
		{name: "opus 5", model: "claude-opus-5", want: true},
		{name: "sonnet 5", model: "claude-sonnet-5", want: true},
		{name: "fable 5", model: "claude-fable-5", want: true},
		{name: "mythos 5", model: "claude-mythos-5", want: true},
		{name: "sonnet 4.5", model: "claude-sonnet-4-5", want: false},
		{name: "claude 3.7", model: "claude-3-7-sonnet", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ShouldOmitClaudeSamplingParams(tt.model))
		})
	}
}
