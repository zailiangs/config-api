package sshconfig

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	portLineRE = regexp.MustCompile(`(?i)^(\s*)port\s+\d+(\s*(?:#.*)?)$`)
	matchRE    = regexp.MustCompile(`(?i)^\s*match(?:\s|$)`)
)

// SetGlobalPort replaces the global Port directive and removes duplicate
// global Port directives. Match blocks are left untouched.
func SetGlobalPort(input []byte, port int) ([]byte, bool, error) {
	if port < 1 || port > 65535 {
		return nil, false, fmt.Errorf("port must be between 1 and 65535")
	}

	hasFinalNewline := bytes.HasSuffix(input, []byte("\n"))
	lines := strings.Split(strings.TrimSuffix(string(input), "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = nil
	}

	firstMatch := len(lines)
	for i, line := range lines {
		if matchRE.MatchString(line) {
			firstMatch = i
			break
		}
	}

	portText := strconv.Itoa(port)
	found := false
	out := make([]string, 0, len(lines)+1)
	for i, line := range lines {
		if i < firstMatch {
			if matches := portLineRE.FindStringSubmatch(line); matches != nil {
				if found {
					continue
				}
				line = matches[1] + "Port " + portText + matches[2]
				found = true
			}
		}
		out = append(out, line)
	}

	if !found {
		insertAt := len(out)
		for i, line := range out {
			if matchRE.MatchString(line) {
				insertAt = i
				break
			}
		}
		out = append(out, "")
		copy(out[insertAt+1:], out[insertAt:])
		out[insertAt] = "Port " + portText
	}

	result := []byte(strings.Join(out, "\n"))
	if hasFinalNewline || len(result) > 0 {
		result = append(result, '\n')
	}
	return result, !bytes.Equal(input, result), nil
}
