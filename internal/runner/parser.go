package runner

import (
	"fmt"
	"strings"
)

// ParsedTest represents a parsed test.md file.
type ParsedTest struct {
	Description string
	Priority    string // "blocking" or "warning"
	Commands    string // shell commands to execute
	Expected    string // expected output description
	IsManual    bool   // true if test has no executable commands
}

// ParseTestMD parses a test.md file into its structured sections.
func ParseTestMD(content string) (*ParsedTest, error) {
	sections := parseSections(content)

	pt := &ParsedTest{}

	if desc, ok := sections["description"]; ok {
		pt.Description = strings.TrimSpace(desc)
	}

	if pri, ok := sections["priority"]; ok {
		pt.Priority = strings.TrimSpace(strings.ToLower(pri))
	}
	if pt.Priority == "" {
		pt.Priority = "warning" // default
	}
	if pt.Priority != "blocking" && pt.Priority != "warning" {
		return nil, fmt.Errorf("invalid priority %q: must be 'blocking' or 'warning'", pt.Priority)
	}

	if cmds, ok := sections["commands"]; ok {
		pt.Commands = extractCodeBlock(cmds)
		if pt.Commands == "" {
			// No fenced code block — use the raw content
			pt.Commands = strings.TrimSpace(cmds)
		}
	}

	if exp, ok := sections["expected output"]; ok {
		pt.Expected = strings.TrimSpace(exp)
	}

	pt.IsManual = pt.Commands == ""

	return pt, nil
}

// parseSections splits markdown content by ## headings into a map.
func parseSections(content string) map[string]string {
	sections := make(map[string]string)
	lines := strings.Split(content, "\n")

	var currentSection string
	var currentContent []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			// Save previous section
			if currentSection != "" {
				sections[currentSection] = strings.Join(currentContent, "\n")
			}
			currentSection = strings.ToLower(strings.TrimPrefix(trimmed, "## "))
			currentContent = nil
		} else {
			currentContent = append(currentContent, line)
		}
	}

	// Save last section
	if currentSection != "" {
		sections[currentSection] = strings.Join(currentContent, "\n")
	}

	return sections
}

// extractCodeBlock extracts the content of the first fenced code block (```...```).
func extractCodeBlock(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if inBlock {
				break // end of block
			}
			inBlock = true
			continue
		}
		if inBlock {
			result = append(result, line)
		}
	}

	if len(result) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}
