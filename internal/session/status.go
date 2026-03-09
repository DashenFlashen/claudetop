package session

import (
	"regexp"
	"strings"
	"time"
)

const (
	startupGracePeriod    = 10 * time.Second
	activeOutputWindow    = 3 * time.Second
	stuckSilenceThreshold = 2 * time.Minute
)

var (
	needsInputPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\?\s*$`),
		regexp.MustCompile(`(?i)would you like`),
		regexp.MustCompile(`(?i)should i`),
		regexp.MustCompile(`(?i)do you want`),
		regexp.MustCompile(`(?i)shall i`),
	}

	permissionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)do you want me to`),
		regexp.MustCompile(`(?i)allow.*to run`),
		regexp.MustCompile(`(?i)may i`),
		regexp.MustCompile(`\(y/N\)`),
		regexp.MustCompile(`\(Y/n\)`),
	}

	errorPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?m)^Error:`),
		regexp.MustCompile(`(?m)^error:`),
	}
)

// lastNLines returns the last n lines of text.
func lastNLines(text string, n int) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) <= n {
		return text
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

// Detect derives session status from pane output and timing.
// current is the previously detected status (needed for stuck detection).
func Detect(paneOutput string, lastOutputAt time.Time, createdAt time.Time, current Status) Status {
	age := time.Since(createdAt)
	if age < startupGracePeriod {
		return StatusStarting
	}

	tail := lastNLines(paneOutput, 15)

	for _, p := range errorPatterns {
		if p.MatchString(tail) {
			return StatusError
		}
	}

	if time.Since(lastOutputAt) < activeOutputWindow {
		return StatusWorking
	}

	for _, p := range permissionPatterns {
		if p.MatchString(tail) {
			return StatusPermission
		}
	}

	for _, p := range needsInputPatterns {
		if p.MatchString(tail) {
			return StatusNeedsInput
		}
	}

	if current == StatusWorking && time.Since(lastOutputAt) > stuckSilenceThreshold {
		return StatusStuck
	}

	return StatusDone
}
