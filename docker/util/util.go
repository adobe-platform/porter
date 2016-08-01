package util

import (
	"regexp"
	"strings"
)

var validKvp = regexp.MustCompile(`^[a-zA-Z0-9_]+=.*$`)

// CleanEnvFile scrubs comments and removes invalid key-value pairs
func CleanEnvFile(contents string) string {
	lines := strings.Split(contents, "\n")
	cleanedLines := []string{}
	for _, line := range lines {
		if validKvp.MatchString(line) {
			cleanedLines = append(cleanedLines, line)
		}
	}
	return strings.Join(cleanedLines, "\n")
}
