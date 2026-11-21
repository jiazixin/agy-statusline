// Antigravity CLI (agy) custom status line — Go edition.
//
// Catppuccin / Tokyo-Night inspired palette with smooth gradient context bar,
// truecolor (24-bit) support with automatic 256-color fallback, and responsive layout.
package main

import (
	"cmp"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	exitOK    = 0
	exitUsage = 2
)

// version may be set via -ldflags "-X main.version=v1.0.0" for builds without VCS metadata.
var version = "dev"

const usageText = `Usage: agy-statusline [options]

Custom two-line status bar for Antigravity CLI (agy).
Reads a JSON status payload from stdin and writes the rendered status line to stdout.

Options:
  -h, --help       Show this help message and exit
  -v, --version    Show version information and exit

Environment:
  COLORTERM        Contains "truecolor" or "24bit" to enable 24-bit color
  TERM_PROGRAM     Auto-enables 24-bit color for supported terminals
  WT_SESSION       Auto-enables 24-bit color in Windows Terminal
  AGY_STATUS_DEBUG Set to 1 to dump raw stdin JSON to the system temp directory
`

type buildInfo struct {
	Version  string
	Revision string
	Modified bool
}

func currentBuild() buildInfo {
	info, _ := debug.ReadBuildInfo()
	return readBuild(version, info)
}

func readBuild(release string, info *debug.BuildInfo) buildInfo {
	var b buildInfo
	var moduleVersion string
	if info != nil {
		moduleVersion = info.Main.Version
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				b.Revision = s.Value[:min(12, len(s.Value))]
			case "vcs.modified":
				b.Modified = s.Value == "true"
			}
		}
	}
	b.Version = cmp.Or(usableVersion(release), usableVersion(moduleVersion), b.Revision, "dev")
	return b
}

func usableVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "dev" || v == "(devel)" {
		return ""
	}
	return v
}

func (b buildInfo) String() string {
	var details []string
	if b.Revision != "" && !strings.Contains(b.Version, b.Revision) {
		details = append(details, "revision="+b.Revision)
	}
	if b.Modified {
		details = append(details, "modified")
	}
	if len(details) == 0 {
		return b.Version
	}
	return b.Version + " (" + strings.Join(details, ", ") + ")"
}

type rgb [3]int

const (
	reset = "\033[0m"
	bold  = "\033[1m"
)

var (
	trueColor = detectTrueColor()

	cMuted    = fg(rgb{127, 132, 156}, 245)
	cSep      = fg(rgb{69, 71, 90}, 239)
	cTrack    = fg(rgb{69, 71, 90}, 239)
	cGreen    = fg(rgb{166, 227, 161}, 114)
	cTeal     = fg(rgb{148, 226, 213}, 116)
	cSky      = fg(rgb{137, 220, 235}, 117)
	cBlue     = fg(rgb{137, 180, 250}, 111)
	cLavender = fg(rgb{180, 190, 254}, 147)
	cMauve    = fg(rgb{203, 166, 247}, 183)
	cYellow   = fg(rgb{249, 226, 175}, 222)
	cPeach    = fg(rgb{250, 179, 135}, 216)
	cRed      = fg(rgb{243, 139, 168}, 211)
	cSubtext  = fg(rgb{166, 173, 200}, 250)

	rgbTeal   = rgb{148, 226, 213}
	rgbSky    = rgb{137, 220, 235}
	rgbYellow = rgb{249, 226, 175}
	rgbPeach  = rgb{250, 179, 135}
	rgbRed    = rgb{243, 139, 168}

	bucketLabelReplacer = strings.NewReplacer(
		"weekly", "wk",
		"seven_day", "7d",
		"five_hour", "5h",
	)
)

func detectTrueColor() bool {
	ct := strings.ToLower(os.Getenv("COLORTERM"))
	if containsAny(ct, "truecolor", "24bit") {
		return true
	}
	if os.Getenv("WT_SESSION") != "" {
		return true
	}
	tp := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	return containsAny(
		tp,
		"iterm", "ghostty", "wezterm", "warp", "vscode",
		"kitty", "alacritty", "hyper", "rio", "tabby",
	)
}

func containsAny(s string, subs ...string) bool {
	return slices.ContainsFunc(subs, func(sub string) bool {
		return strings.Contains(s, sub)
	})
}

func fg(c rgb, fallback256 int) string {
	if trueColor {
		return fmt.Sprintf("\033[38;2;%d;%d;%dm", c[0], c[1], c[2])
	}
	return fmt.Sprintf("\033[38;5;%dm", fallback256)
}

func lerpRGB(c1, c2 rgb, t float64) rgb {
	t = max(0.0, min(1.0, t))
	return rgb{
		int(math.Round(float64(c1[0]) + float64(c2[0]-c1[0])*t)),
		int(math.Round(float64(c1[1]) + float64(c2[1]-c1[1])*t)),
		int(math.Round(float64(c1[2]) + float64(c2[2]-c1[2])*t)),
	}
}

// isWideRune reports whether r occupies 2 terminal cells (East Asian Wide/Fullwidth & common wide emoji).
func isWideRune(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115F,
		r == 0x231A || r == 0x231B,
		r == 0x2329 || r == 0x232A,
		r >= 0x23E9 && r <= 0x23EC,
		r == 0x23F0 || r == 0x23F3,
		r == 0x25FD || r == 0x25FE,
		r == 0x2614 || r == 0x2615,
		r >= 0x2648 && r <= 0x2653,
		r == 0x267F || r == 0x2693 || r == 0x26A1,
		r == 0x26AA || r == 0x26AB,
		r == 0x26BD || r == 0x26BE,
		r == 0x26C4 || r == 0x26C5,
		r == 0x26CE || r == 0x26D4 || r == 0x26EA,
		r == 0x26F2 || r == 0x26F3 || r == 0x26F5 || r == 0x26FA || r == 0x26FD,
		r == 0x2705 || r == 0x270A || r == 0x270B || r == 0x2728,
		r == 0x274C || r == 0x274E,
		r >= 0x2753 && r <= 0x2755,
		r == 0x2757,
		r >= 0x2795 && r <= 0x2797,
		r == 0x27B0 || r == 0x27BF,
		r == 0x2B1B || r == 0x2B1C || r == 0x2B50 || r == 0x2B55,
		r >= 0x2E80 && r <= 0x303E,
		r >= 0x3040 && r <= 0x4DBF,
		r >= 0x4E00 && r <= 0xA4CF,
		r >= 0xAC00 && r <= 0xD7A3,
		r >= 0xF900 && r <= 0xFAFF,
		r >= 0xFE10 && r <= 0xFE19,
		r >= 0xFE30 && r <= 0xFE6F,
		r >= 0xFF00 && r <= 0xFF60,
		r >= 0xFFE0 && r <= 0xFFE6,
		r >= 0x1F300 && r <= 0x1F64F,
		r >= 0x1F680 && r <= 0x1F6FF,
		r >= 0x1F900 && r <= 0x1FAFF,
		r >= 0x20000 && r <= 0x3FFFD:
		return true
	default:
		return false
	}
}

// visibleWidth computes terminal cell width while skipping ANSI SGR escape sequences without allocating.
func visibleWidth(s string) int {
	w := 0
	for i := 0; i < len(s); {
		if s[i] == '\033' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && ((s[j] >= '0' && s[j] <= '9') || s[j] == ';') {
				j++
			}
			if j < len(s) && s[j] == 'm' {
				i = j + 1
				continue
			}
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Cf, r) {
			continue
		}
		if isWideRune(r) {
			w += 2
		} else {
			w++
		}
	}
	return w
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func asFloat(v any) (float64, bool) {
	var f float64
	switch n := v.(type) {
	case float64:
		f = n
	case int:
		f = float64(n)
	case int64:
		f = float64(n)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		if err != nil {
			return 0, false
		}
		f = parsed
	default:
		return 0, false
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

func asBool(v any) bool {
	b, _ := v.(bool)
	return b
}

// firstPositiveFloat returns the first key in m whose numeric value is > 0.
func firstPositiveFloat(m map[string]any, keys ...string) (float64, bool) {
	for _, k := range keys {
		if f, ok := asFloat(m[k]); ok && f > 0 {
			return f, true
		}
	}
	return 0, false
}

// firstFloatPreferPositive returns the first positive numeric value among keys,
// falling back to the first non-negative numeric value (such as 0) if none is positive.
func firstFloatPreferPositive(m map[string]any, keys ...string) (float64, bool) {
	var fallback float64
	hasFallback := false
	for _, k := range keys {
		f, ok := asFloat(m[k])
		if !ok || f < 0 {
			continue
		}
		if f > 0 {
			return f, true
		}
		if !hasFallback {
			fallback = f
			hasFallback = true
		}
	}
	return fallback, hasFallback
}

func formatPlainNumber(v any) string {
	if f, ok := asFloat(v); ok {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return asString(v)
}

func fmtTokens(v any) string {
	n, ok := asFloat(v)
	if !ok {
		return ""
	}
	n = max(0.0, n)
	if n >= 1_000_000 {
		val := n / 1_000_000
		if val < 10 {
			return fmt.Sprintf("%.1fM", val)
		}
		return fmt.Sprintf("%.0fM", val)
	}
	if n >= 1_000 {
		val := n / 1_000
		if val < 10 {
			return fmt.Sprintf("%.1fk", val)
		}
		return fmt.Sprintf("%.0fk", val)
	}
	return strconv.Itoa(int(n))
}

func fmtDuration(v any) string {
	f, ok := asFloat(v)
	if !ok {
		return ""
	}
	sec := int(f)
	if sec <= 0 {
		return "now"
	}
	days := sec / 86400
	rem := sec % 86400
	hours := rem / 3600
	rem %= 3600
	mins := rem / 60
	if days > 0 {
		if hours > 0 {
			return fmt.Sprintf("%dd%dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	}
	if hours > 0 {
		if mins > 0 {
			return fmt.Sprintf("%dh%02dm", hours, mins)
		}
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", max(1, mins))
}

func gradientProgressBar(pct float64, width int) (string, string) {
	pct = max(0.0, min(100.0, pct))
	filled := int(math.Round(pct / 100.0 * float64(width)))
	empty := width - filled

	var startRGB, endRGB rgb
	var accent string
	var fb int
	switch {
	case pct >= 80:
		startRGB, endRGB = rgbPeach, rgbRed
		accent, fb = cRed, 211
	case pct >= 60:
		startRGB, endRGB = rgbYellow, rgbPeach
		accent, fb = cYellow, 222
	default:
		startRGB, endRGB = rgbTeal, rgbSky
		accent, fb = cTeal, 116
	}

	var b strings.Builder
	if trueColor && filled > 0 {
		denom := float64(max(1, width-1))
		for i := range filled {
			c := lerpRGB(startRGB, endRGB, float64(i)/denom)
			fmt.Fprintf(&b, "\033[38;2;%d;%d;%dm━", c[0], c[1], c[2])
		}
	} else if filled > 0 {
		b.WriteString(fg(endRGB, fb))
		b.WriteString(strings.Repeat("━", filled))
	}
	if empty > 0 {
		b.WriteString(cTrack)
		b.WriteString(strings.Repeat("─", empty))
	}
	b.WriteString(reset)
	return b.String(), accent
}

func quotaColor(remPct float64) string {
	switch {
	case remPct <= 20:
		return cRed
	case remPct <= 40:
		return cPeach
	case remPct <= 65:
		return cYellow
	default:
		return cGreen
	}
}

func formatContext(cw map[string]any, compact bool) string {
	if cw == nil {
		return ""
	}

	usedPct, hasPct := asFloat(cw["used_percentage"])
	maxTokens, hasMax := firstPositiveFloat(cw, "context_window_size", "total_tokens")
	usedTokens, hasUsed := firstFloatPreferPositive(cw, "used_tokens", "total_input_tokens")

	if !hasPct && hasUsed && hasMax {
		usedPct = (usedTokens / maxTokens) * 100.0
		hasPct = true
	}
	if !hasPct {
		return cMuted + "Ctx --" + reset
	}
	usedPct = max(0.0, usedPct)

	barWidth := 10
	if compact {
		barWidth = 8
	}
	bar, accent := gradientProgressBar(usedPct, barWidth)

	if !hasUsed && hasMax {
		usedTokens = maxTokens * (usedPct / 100.0)
		hasUsed = true
	}

	var pctStr string
	if usedPct >= 10 {
		pctStr = fmt.Sprintf("%s%s%.0f%%%s", accent, bold, usedPct, reset)
	} else {
		pctStr = fmt.Sprintf("%s%s%.1f%%%s", accent, bold, usedPct, reset)
	}

	if !compact && hasUsed && hasMax {
		tokStr := fmt.Sprintf("%s%s/%s%s", cMuted, fmtTokens(usedTokens), fmtTokens(maxTokens), reset)
		return bar + " " + pctStr + " " + tokStr
	}
	return bar + " " + pctStr
}

func formatCache(cw map[string]any, compact bool) string {
	if cw == nil {
		return ""
	}
	usage := asMap(cw["current_usage"])
	if usage == nil {
		return ""
	}
	inp, _ := asFloat(usage["input_tokens"])
	cacheRead, _ := asFloat(usage["cache_read_input_tokens"])
	cacheCreate, _ := asFloat(usage["cache_creation_input_tokens"])

	inp = max(0.0, inp)
	cacheRead = max(0.0, cacheRead)
	cacheCreate = max(0.0, cacheCreate)

	total := inp + cacheRead + cacheCreate
	if total <= 0 {
		return ""
	}

	hitPct := max(0.0, min(100.0, (cacheRead/total)*100.0))
	var color string
	switch {
	case hitPct >= 70:
		color = cTeal
	case hitPct >= 30:
		color = cSky
	case hitPct > 0:
		color = cYellow
	default:
		color = cMuted
	}

	if compact {
		return fmt.Sprintf("%sc:%s%s%s%.0f%%%s", cMuted, reset, color, bold, hitPct, reset)
	}
	return fmt.Sprintf("%scache%s %s%s%.0f%%%s", cMuted, reset, color, bold, hitPct, reset)
}

func shortenBucketLabel(name string, stripPrefix bool) string {
	label := name
	if stripPrefix {
		if _, after, ok := strings.Cut(label, "-"); ok {
			label = after
		}
	}
	return bucketLabelReplacer.Replace(label)
}

type quotaBucket struct {
	key string
	val map[string]any
}

func bucketRank(key string) int {
	k := strings.ToLower(key)
	switch {
	case strings.Contains(k, "5h") || strings.Contains(k, "hour"):
		return 0
	case strings.Contains(k, "day") || strings.Contains(k, "24h"):
		return 1
	case strings.Contains(k, "week") || strings.Contains(k, "7d"):
		return 2
	default:
		return 3
	}
}

func formatQuota(data map[string]any, modelHint string, compact bool) string {
	var items []string
	if quota := asMap(data["quota"]); quota != nil {
		if windows, ok := quota["windows"].([]any); ok {
			for _, rawW := range windows {
				w := asMap(rawW)
				if w == nil || asBool(w["disabled"]) {
					continue
				}
				rem, ok := asFloat(w["remaining_fraction"])
				if !ok {
					continue
				}
				remPct := rem * 100.0
				color := quotaColor(remPct)
				label := shortenBucketLabel(asString(w["type"]), false)
				cd := ""
				if !compact {
					cd = fmtDuration(w["reset_in_seconds"])
				}
				cdStr := ""
				if cd != "" {
					cdStr = " " + cMuted + cd + reset
				}
				prefix := ""
				if label != "" {
					prefix = cMuted + label + reset + " "
				}
				items = append(items, fmt.Sprintf("%s%s%s%.0f%%%s%s", prefix, color, bold, remPct, reset, cdStr))
			}
		} else {
			var buckets []quotaBucket
			for k, rawV := range quota {
				v := asMap(rawV)
				if v == nil || asBool(v["disabled"]) {
					continue
				}
				if v["remaining_fraction"] != nil || v["remaining_amount"] != nil {
					buckets = append(buckets, quotaBucket{key: k, val: v})
				}
			}

			if len(buckets) > 0 && modelHint != "" {
				mh := strings.ToLower(modelHint)
				var matched []quotaBucket
				switch {
				case strings.Contains(mh, "gemini"):
					for _, b := range buckets {
						if strings.Contains(strings.ToLower(b.key), "gemini") {
							matched = append(matched, b)
						}
					}
				case containsAny(mh, "claude", "sonnet", "opus", "haiku", "gpt", "o1", "o3", "o4"):
					for _, b := range buckets {
						if containsAny(strings.ToLower(b.key), "3p", "claude", "third", "openai", "gpt") {
							matched = append(matched, b)
						}
					}
				}
				if len(matched) > 0 {
					buckets = matched
				}
			}

			slices.SortFunc(buckets, func(a, b quotaBucket) int {
				if c := cmp.Compare(bucketRank(a.key), bucketRank(b.key)); c != 0 {
					return c
				}
				return cmp.Compare(strings.ToLower(a.key), strings.ToLower(b.key))
			})

			prefixes := make(map[string]struct{})
			for _, b := range buckets {
				if p, _, ok := strings.Cut(b.key, "-"); ok {
					prefixes[p] = struct{}{}
				}
			}
			stripPrefix := len(prefixes) == 1

			for _, b := range buckets {
				label := shortenBucketLabel(b.key, stripPrefix)
				var valStr string
				if rem, ok := asFloat(b.val["remaining_fraction"]); ok {
					remPct := rem * 100.0
					color := quotaColor(remPct)
					valStr = fmt.Sprintf("%s%s%.0f%%%s", color, bold, remPct, reset)
				} else {
					valStr = fmt.Sprintf("%s%s%s%s", cGreen, bold, formatPlainNumber(b.val["remaining_amount"]), reset)
				}
				cd := ""
				if !compact {
					cd = fmtDuration(b.val["reset_in_seconds"])
				}
				cdStr := ""
				if cd != "" {
					cdStr = cMuted + "·" + cd + reset
				}
				items = append(items, fmt.Sprintf("%s%s%s %s%s", cMuted, label, reset, valStr, cdStr))
			}
		}
	}

	if len(items) == 0 {
		if rl := asMap(data["rate_limits"]); rl != nil {
			for _, pair := range [][2]string{{"five_hour", "5h"}, {"seven_day", "7d"}} {
				if block := asMap(rl[pair[0]]); block != nil {
					if used, ok := asFloat(block["used_percentage"]); ok {
						remPct := max(0.0, 100.0-used)
						color := quotaColor(remPct)
						items = append(items, fmt.Sprintf("%s%s%s %s%s%.0f%%%s", cMuted, pair[1], reset, color, bold, remPct, reset))
					}
				}
			}
		}
	}

	var rawCredits any
	if v, ok := data["ai_credits"]; ok {
		rawCredits = v
	} else {
		rawCredits = data["credits"]
	}

	var creditsVal float64
	hasCredits := false
	if f, ok := asFloat(rawCredits); ok {
		creditsVal, hasCredits = f, true
	} else if cm := asMap(rawCredits); cm != nil {
		for _, ck := range []string{"remaining", "remaining_amount", "balance", "count"} {
			if f, ok := asFloat(cm[ck]); ok {
				creditsVal, hasCredits = f, true
				break
			}
		}
	}

	if hasCredits {
		var cStr string
		if creditsVal == math.Trunc(creditsVal) {
			cStr = strconv.FormatFloat(creditsVal, 'f', 0, 64)
		} else {
			cStr = fmt.Sprintf("%.1f", creditsVal)
		}
		cColor := cGreen
		if creditsVal <= 10 {
			cColor = cRed
		} else if creditsVal <= 50 {
			cColor = cPeach
		}
		items = append(items, fmt.Sprintf("%s⚡%s%s%s%s%s", cMuted, reset, cColor, bold, cStr, reset))
	}

	return strings.Join(items, "  ")
}

func formatState(state any) string {
	s := strings.ToLower(asString(state))
	if s == "" {
		return ""
	}
	switch s {
	case "working", "thinking", "tool_use", "running":
		return fmt.Sprintf("%s◉%s %s%s%s", cYellow, reset, cYellow, s, reset)
	case "idle":
		return fmt.Sprintf("%s●%s %sidle%s", cGreen, reset, cMuted, reset)
	default:
		return fmt.Sprintf("%s○ %s%s", cMuted, s, reset)
	}
}

func formatMode(cycleMode any) string {
	m := strings.ToLower(strings.TrimSpace(asString(cycleMode)))
	switch m {
	case "auto":
		return cTeal + bold + "» auto" + reset
	case "accept-edits":
		return cPeach + bold + "✎ accept-edits" + reset
	case "plan":
		return cSky + bold + "◈ plan" + reset
	case "", "default":
		return cSubtext + "◦ default" + reset
	default:
		return cSubtext + m + reset
	}
}

func hasGitDir(cwd string) bool {
	cur, err := filepath.Abs(cwd)
	if err != nil {
		return false
	}
	for {
		if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return false
		}
		cur = parent
	}
}

func getGitStatus(cwd string) (string, bool) {
	if cwd == "" {
		return "", false
	}
	info, err := os.Stat(cwd)
	if err != nil || !info.IsDir() || !hasGitDir(cwd) {
		return "", false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "--no-optional-locks", "-C", cwd, "status", "--porcelain=v2", "--branch")
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}

	var branch, oid string
	dirty := false
	for line := range strings.SplitSeq(string(out), "\n") {
		if v, ok := strings.CutPrefix(line, "# branch.head "); ok {
			branch = strings.TrimSpace(v)
		} else if v, ok := strings.CutPrefix(line, "# branch.oid "); ok {
			oid = strings.TrimSpace(v)
		} else if !strings.HasPrefix(line, "#") && strings.TrimSpace(line) != "" {
			dirty = true
		}
	}

	if branch == "(detached)" {
		if oid != "" && oid != "(initial)" {
			branch = "@" + oid[:min(7, len(oid))]
		} else {
			branch = "detached"
		}
	}
	return branch, dirty
}

func joinLeftRight(leftStr, rightStr string, targetWidth int, sep string) string {
	switch {
	case leftStr != "" && rightStr != "":
		gap := targetWidth - visibleWidth(leftStr) - visibleWidth(rightStr)
		if gap >= 2 {
			return leftStr + strings.Repeat(" ", gap) + rightStr
		}
		return leftStr + sep + rightStr
	case leftStr != "":
		return leftStr
	case rightStr != "":
		pad := max(0, targetWidth-visibleWidth(rightStr))
		return strings.Repeat(" ", pad) + rightStr
	default:
		return ""
	}
}

func render(data map[string]any) string {
	width := 120
	if w, ok := asFloat(data["terminal_width"]); ok && w > 0 {
		width = int(w)
	}
	compact := width < 80
	targetWidth := max(40, width-4)

	var l1Left, l1Right, l2Left, l2Right []string

	model := asMap(data["model"])
	modelID := asString(model["id"])
	modelName := cmp.Or(asString(model["display_name"]), modelID)
	effort := asString(model["effort"])

	if modelName != "" {
		modelLabel := modelName
		if effort != "" && !strings.Contains(strings.ToLower(modelName), strings.ToLower(effort)) {
			modelLabel = fmt.Sprintf("%s %s(%s)%s", modelName, cMuted, effort, reset)
		}
		l1Left = append(l1Left, fmt.Sprintf("%s\U000f0674 %s%s%s", cLavender, bold, modelLabel, reset))
	}

	cw := asMap(data["context_window"])
	if ctxStr := formatContext(cw, compact); ctxStr != "" {
		l1Left = append(l1Left, ctxStr)
	}
	if cacheStr := formatCache(cw, compact); cacheStr != "" {
		l1Left = append(l1Left, cacheStr)
	}

	if !compact {
		if stateStr := formatState(data["agent_state"]); stateStr != "" {
			l1Right = append(l1Right, stateStr)
		}
	}
	if modeStr := formatMode(data["cycle_mode"]); modeStr != "" {
		l1Right = append(l1Right, modeStr)
	}

	if !compact {
		if planTier := asString(data["plan_tier"]); planTier != "" {
			l2Left = append(l2Left, fmt.Sprintf("%s\U000f01a5%s %s%s%s", cYellow, reset, cSubtext, planTier, reset))
		}
	}

	ws := asMap(data["workspace"])
	cwd := cmp.Or(asString(ws["current_dir"]), asString(ws["current_directory"]), asString(data["cwd"]))
	dirName := baseDirName(cwd)

	vcs := asMap(data["vcs"])
	branch := cmp.Or(asString(vcs["branch"]), asString(ws["git_branch"]))
	var dirty any
	if v, ok := vcs["dirty"]; ok {
		dirty = v
	} else {
		dirty = ws["is_dirty"]
	}

	vcsType := strings.ToLower(asString(vcs["type"]))
	if branch == "" && cwd != "" && (vcsType == "" || vcsType == "git") {
		if gitBranch, gitDirty := getGitStatus(cwd); gitBranch != "" {
			branch = gitBranch
			if dirty == nil {
				dirty = gitDirty
			}
		}
	}

	var locParts []string
	if dirName != "" {
		locParts = append(locParts, fmt.Sprintf("%s\U000f0770 %s%s", cBlue, dirName, reset))
	}
	if branch != "" {
		dirtyMark := ""
		if asBool(dirty) {
			dirtyMark = cPeach + "*" + reset
		}
		locParts = append(locParts, fmt.Sprintf("%s\uf418 %s%s%s", cMauve, branch, reset, dirtyMark))
	}
	if len(locParts) > 0 {
		l2Left = append(l2Left, strings.Join(locParts, "  "))
	}

	var extras []string
	switch sa := data["subagents"].(type) {
	case []any:
		if len(sa) > 0 {
			extras = append(extras, fmt.Sprintf("%sagents:%d%s", cSky, len(sa), reset))
		}
	default:
		if n, ok := asFloat(sa); ok && int(n) > 0 {
			extras = append(extras, fmt.Sprintf("%sagents:%d%s", cSky, int(n), reset))
		}
	}

	if n, ok := asFloat(data["task_count"]); ok && int(n) > 0 {
		extras = append(extras, fmt.Sprintf("%stasks:%d%s", cYellow, int(n), reset))
	}
	if n, ok := asFloat(data["artifact_count"]); ok && int(n) > 0 {
		extras = append(extras, fmt.Sprintf("%sartifacts:%d%s", cMuted, int(n), reset))
	}
	if sb := asMap(data["sandbox"]); sb != nil && asBool(sb["enabled"]) {
		extras = append(extras, cTeal+"sandbox"+reset)
	}
	if len(extras) > 0 {
		l2Left = append(l2Left, strings.Join(extras, " "))
	}

	if quotaStr := formatQuota(data, modelID+" "+modelName, compact); quotaStr != "" {
		l2Right = append(l2Right, quotaStr)
	}

	if !compact && len(l1Left) > 0 && len(l2Left) > 0 {
		w1 := visibleWidth(l1Left[0])
		w2 := visibleWidth(l2Left[0])
		if w1 > w2 {
			l2Left[0] += strings.Repeat(" ", w1-w2)
		} else if w2 > w1 {
			l1Left[0] += strings.Repeat(" ", w2-w1)
		}
	}

	sep := " " + cSep + "│" + reset + " "
	row1 := joinLeftRight(strings.Join(l1Left, sep), strings.Join(l1Right, "  "), targetWidth, sep)
	row2 := joinLeftRight(strings.Join(l2Left, sep), strings.Join(l2Right, sep), targetWidth, sep)

	var lines []string
	if row1 != "" {
		lines = append(lines, row1)
	}
	if row2 != "" {
		lines = append(lines, row2)
	}
	return strings.Join(lines, "\n")
}

func baseDirName(cwd string) string {
	trimmed := strings.TrimRight(cwd, `/\`)
	if trimmed == "" {
		return ""
	}
	if i := strings.LastIndexAny(trimmed, `/\`); i >= 0 {
		return trimmed[i+1:]
	}
	return trimmed
}

func debugFilePath() string {
	return filepath.Join(os.TempDir(), "agy-statusline-debug.json")
}

func run(r io.Reader, w io.Writer) error {
	raw, err := io.ReadAll(r)
	if err != nil || len(strings.TrimSpace(string(raw))) == 0 {
		return err
	}

	if os.Getenv("AGY_STATUS_DEBUG") == "1" {
		_ = os.WriteFile(debugFilePath(), raw, 0o600)
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	if data == nil {
		return errors.New("status payload must be a JSON object")
	}

	if out := render(data); out != "" {
		_, err = fmt.Fprintln(w, out)
		return err
	}
	return nil
}

func runCLI(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		if len(args) == 1 {
			switch args[0] {
			case "-h", "--help":
				_, _ = fmt.Fprint(stdout, usageText)
				return exitOK
			case "-v", "--version":
				_, _ = fmt.Fprintln(stdout, "agy-statusline", currentBuild())
				return exitOK
			}
		}
		_, _ = fmt.Fprintf(stderr, "agy-statusline: unknown argument: %s\nRun 'agy-statusline --help' for usage.\n", strings.Join(args, " "))
		return exitUsage
	}

	if err := run(stdin, stdout); err != nil {
		_, _ = fmt.Fprintf(stdout, "%s✦ Antigravity%s\n", cLavender, reset)
	}
	return exitOK
}

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
