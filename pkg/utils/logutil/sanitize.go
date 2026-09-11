package logutil

import (
	"regexp"
	"strings"
)

// SanitizeLogString removes control characters, newlines, and ANSI escape sequences
// from strings before logging to prevent log injection attacks.
func SanitizeLogString(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	s = re.ReplaceAllString(s, "")
	return s
}
