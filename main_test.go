package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime/debug"
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

func TestBaseDirName(t *testing.T) {
	tests := []struct {
		cwd  string
		want string
	}{
		{"", ""},
		{"/", ""},
		{"///", ""},
		{`\`, ""},
		{"/Users/k/agy-statusline", "agy-statusline"},
		{"/Users/k/agy-statusline/", "agy-statusline"},
		{`C:\Users\k\agy-statusline`, "agy-statusline"},
		{`C:\Users\k\agy-statusline\`, "agy-statusline"},
		{"C:/Users/k/agy-statusline/", "agy-statusline"},
		{`C:\`, "C:"},
		{"C:/", "C:"},
		{`\\server\share\`, "share"},
		{"project", "project"},
	}
	for _, tc := range tests {
		if got := baseDirName(tc.cwd); got != tc.want {
			t.Errorf("baseDirName(%q) = %q, want %q", tc.cwd, got, tc.want)
		}
	}
}

func TestDetectTrueColor(t *testing.T) {
	t.Run("windows terminal WT_SESSION", func(t *testing.T) {
		t.Setenv("COLORTERM", "")
		t.Setenv("TERM_PROGRAM", "")
		t.Setenv("WT_SESSION", "00000000-0000-0000-0000-000000000000")
		if !detectTrueColor() {
			t.Fatal("detectTrueColor() = false with WT_SESSION set, want true")
		}
	})
	t.Run("fallback without env", func(t *testing.T) {
		t.Setenv("COLORTERM", "")
		t.Setenv("TERM_PROGRAM", "")
		t.Setenv("WT_SESSION", "")
		if detectTrueColor() {
			t.Fatal("detectTrueColor() = true with empty env, want false")
		}
	})
}

func TestDebugFilePath(t *testing.T) {
	want := filepath.Join(os.TempDir(), "agy-statusline-debug.json")
	if got := debugFilePath(); got != want {
		t.Errorf("debugFilePath() = %q, want %q", got, want)
	}
}

func TestReadBuild(t *testing.T) {
	cases := []struct {
		name    string
		release string
		info    *debug.BuildInfo
		want    string
	}{
		{
			name:    "release ldflags with revision",
			release: "v1.0.1",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "0123456789abcdef"},
				},
			},
			want: "v1.0.1 (revision=0123456789ab)",
		},
		{
			name:    "module tag version clean",
			release: "dev",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v1.0.1"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "0123456789abcdef"},
					{Key: "vcs.modified", Value: "false"},
				},
			},
			want: "v1.0.1 (revision=0123456789ab)",
		},
		{
			name:    "pseudo version with modified",
			release: "dev",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v1.0.1-0.20261121010000-0123456789ab"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "0123456789abcdef"},
					{Key: "vcs.modified", Value: "true"},
				},
			},
			want: "v1.0.1-0.20261121010000-0123456789ab (modified)",
		},
		{
			name:    "no metadata fallback",
			release: "dev",
			info:    nil,
			want:    "dev",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := readBuild(tc.release, tc.info).String(); got != tc.want {
				t.Errorf("readBuild().String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRunCLI(t *testing.T) {
	t.Run("help", func(t *testing.T) {
		for _, flag := range []string{"-h", "--help"} {
			var out, errBuf bytes.Buffer
			if code := runCLI([]string{flag}, strings.NewReader(""), &out, &errBuf); code != exitOK {
				t.Fatalf("runCLI(%q) = %d, want %d", flag, code, exitOK)
			}
			if !strings.Contains(out.String(), "Usage: agy-statusline") {
				t.Errorf("runCLI(%q) stdout = %q, want usage", flag, out.String())
			}
			if errBuf.Len() != 0 {
				t.Errorf("runCLI(%q) stderr = %q, want empty", flag, errBuf.String())
			}
		}
	})
	t.Run("version", func(t *testing.T) {
		for _, flag := range []string{"-v", "--version"} {
			var out, errBuf bytes.Buffer
			if code := runCLI([]string{flag}, strings.NewReader(""), &out, &errBuf); code != exitOK {
				t.Fatalf("runCLI(%q) = %d, want %d", flag, code, exitOK)
			}
			if !strings.HasPrefix(out.String(), "agy-statusline ") {
				t.Errorf("runCLI(%q) stdout = %q, want agy-statusline prefix", flag, out.String())
			}
			if errBuf.Len() != 0 {
				t.Errorf("runCLI(%q) stderr = %q, want empty", flag, errBuf.String())
			}
		}
	})
	t.Run("unknown arg", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		if code := runCLI([]string{"--bogus"}, strings.NewReader(""), &out, &errBuf); code != exitUsage {
			t.Fatalf("runCLI(--bogus) = %d, want %d", code, exitUsage)
		}
		if out.Len() != 0 {
			t.Errorf("runCLI(--bogus) stdout = %q, want empty", out.String())
		}
		if !strings.Contains(errBuf.String(), "unknown argument") {
			t.Errorf("runCLI(--bogus) stderr = %q, want unknown argument", errBuf.String())
		}
	})
	t.Run("invalid stdin fallback", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		if code := runCLI(nil, strings.NewReader("not-json"), &out, &errBuf); code != exitOK {
			t.Fatalf("runCLI(invalid stdin) = %d, want %d", code, exitOK)
		}
		if !strings.Contains(out.String(), "Antigravity") {
			t.Errorf("runCLI(invalid stdin) stdout = %q, want fallback", out.String())
		}
	})
}
