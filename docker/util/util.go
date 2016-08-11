package util

import (
	"bufio"
	"io"
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

func NetworkNameToId(input io.Reader) (map[string]string, error) {
	output := make(map[string]string)
	scanner := bufio.NewScanner(input)
	scanner.Split(bufio.ScanWords)
	var i int
	for scanner.Scan() {
		if i > 3 && (i-1)%3 == 0 {
			containerId := scanner.Text()
			scanner.Scan()
			name := scanner.Text()
			output[name] = containerId
			i += 2
		} else {
			i++
		}
	}

	err := scanner.Err()
	if err != nil {
		return nil, err
	}

	return output, nil
}
