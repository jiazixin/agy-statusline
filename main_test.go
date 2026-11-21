package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVisibleWidth(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{"plain ascii", "hello", 5},
		{"ansi sgr", "\033[38;2;180;190;254m\033[1mGemini\033[0m", 6},
		{"cjk", "项目测试", 8},
		{"emoji lightning", "\033[38;5;245m⚡\033[0m120", 5},
		{"combining mark", "e\u0301", 1},
		{"incomplete escape", "\033[123", 5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := visibleWidth(tc.in); got != tc.want {
				t.Errorf("visibleWidth(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestFormatContextZeroTokens(t *testing.T) {
	cases := []struct {
		name string
		cw   map[string]any
		want string
	}{
		{
			name: "total_input_tokens zero",
			cw:   map[string]any{"total_input_tokens": 0.0, "context_window_size": 200000.0},
			want: "0.0%",
		},
		{
			name: "used_tokens zero",
			cw:   map[string]any{"used_tokens": 0.0, "context_window_size": 200000.0},
			want: "0/200k",
		},
		{
			name: "used_tokens zero fallback to positive total_input_tokens",
			cw:   map[string]any{"used_tokens": 0.0, "total_input_tokens": 20000.0, "context_window_size": 200000.0},
			want: "10%",
		},
		{
			name: "missing used and pct",
			cw:   map[string]any{"context_window_size": 200000.0},
			want: "Ctx --",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatContext(tc.cw, false)
			if !strings.Contains(got, tc.want) {
				t.Errorf("formatContext() = %q, want substring %q", got, tc.want)
			}
		})
	}
}

func TestFormatQuotaLargeAmount(t *testing.T) {
	data := map[string]any{
		"quota": map[string]any{
			"bucket-1": map[string]any{
				"remaining_amount": 1000000.0,
				"reset_in_seconds": 45.0,
			},
		},
	}
	got := formatQuota(data, "", false)
	if !strings.Contains(got, "1000000") || strings.Contains(got, "e+") {
		t.Errorf("formatQuota() = %q, want plain integer 1000000 without scientific notation", got)
	}
}

func TestAsFloatRejectsNaNInf(t *testing.T) {
	for _, raw := range []any{"NaN", "Inf", "-Inf"} {
		if f, ok := asFloat(raw); ok {
			t.Errorf("asFloat(%q) = (%v, true), want (_, false)", raw, f)
		}
	}
}

func TestRunNullPayloadErrors(t *testing.T) {
	var buf bytes.Buffer
	if err := run(strings.NewReader("null"), &buf); err == nil {
		t.Fatalf("run(null) succeeded with output %q, want error", buf.String())
	}
}

func TestRunEmptyInputNoOutput(t *testing.T) {
	var buf bytes.Buffer
	if err := run(strings.NewReader("   \n"), &buf); err != nil {
		t.Fatalf("run(empty) unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("run(empty) wrote %q, want empty", buf.String())
	}
}
