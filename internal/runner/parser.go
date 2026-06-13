package runner

import (
	"strings"
)

// ParsedTest represents a parsed test.md file.
type ParsedTest struct {
	Description string
	Priority    string // "blocking" or "warning"
	Type        string // "automated", "browser", or "manual"
	Commands    string // shell commands to execute
	Expected    string // expected output description
	NoCommands  bool   // true if test has no executable commands
}

// ParseTestMD parses a test.md file into its structured sections.
func ParseTestMD(content string) (*ParsedTest, error) {
	sections := parseSections(content)

	pt := &ParsedTest{}

	if desc, ok := sections["description"]; ok {
		pt.Description = strings.TrimSpace(desc)
	}

	if pri, ok := sections["priority"]; ok {
		pt.Priority = normalizePriority(strings.TrimSpace(strings.ToLower(pri)))
	}
	if pt.Priority == "" {
		pt.Priority = "warning" // default
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

	if typ, ok := sections["type"]; ok {
		pt.Type = normalizeType(strings.TrimSpace(strings.ToLower(typ)))
	}
	if pt.Type == "" {
		// Infer type from content: if it has Commands, it's automated
		if pt.Commands != "" {
			pt.Type = "automated"
		} else {
			pt.Type = "manual"
		}
	}

	// Binary only runs automated tests
	pt.NoCommands = pt.Commands == "" || pt.Type != "automated"

	return pt, nil
}

// normalizePriority maps common priority synonyms to "blocking" or "warning".
func normalizePriority(raw string) string {
	switch raw {
	case "blocking", "high", "critical", "p0", "must", "required":
		return "blocking"
	case "warning", "low", "medium", "minor", "p1", "p2", "info", "nice-to-have":
		return "warning"
	default:
		return "warning"
	}
}

// normalizeType maps test type values to "automated", "browser", or "manual".
func normalizeType(raw string) string {
	switch raw {
	case "automated", "auto", "docker", "command", "commands":
		return "automated"
	case "browser", "ui", "e2e", "visual":
		return "browser"
	case "manual", "human", "steps":
		return "manual"
	default:
		return "automated"
	}
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
