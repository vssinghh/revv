package runner

import (
	"os"
	"regexp"
	"sort"
	"strings"
)

// shellBuiltins are environment variables provided by the shell/OS that should
// not be forwarded to the sandbox.
var shellBuiltins = map[string]bool{
	"HOME": true, "PWD": true, "USER": true, "PATH": true,
	"SHELL": true, "TERM": true, "TMPDIR": true, "LANG": true,
	"HOSTNAME": true, "OLDPWD": true, "SHLVL": true, "_": true,
	"LOGNAME": true, "DISPLAY": true, "EDITOR": true, "VISUAL": true,
	"PAGER": true, "LESS": true, "MANPATH": true, "INFOPATH": true,
	"LC_ALL": true, "LC_CTYPE": true, "COLORTERM": true,
	"TERM_PROGRAM": true, "TERM_SESSION_ID": true,
	"XPC_FLAGS": true, "XPC_SERVICE_NAME": true,
	"DOCKER_HOST": true, // managed by sandbox itself
}

// envVarPattern matches $VAR_NAME and ${VAR_NAME} in shell commands.
var envVarPattern = regexp.MustCompile(`\$\{?([A-Z][A-Z0-9_]*)\}?`)

// DetectEnvVars scans test.md contents for environment variable references,
// checks which ones are set on the host, and returns them as KEY=VALUE pairs.
// Also loads variables from .env files if they exist.
func DetectEnvVars(testContents []string, envFiles []string) ([]string, []EnvVarStatus) {
	// Collect all referenced variables
	seen := make(map[string]bool)
	for _, content := range testContents {
		matches := envVarPattern.FindAllStringSubmatch(content, -1)
		for _, m := range matches {
			name := m[1]
			if !shellBuiltins[name] {
				seen[name] = true
			}
		}
	}

	// Load .env files
	envFileVars := make(map[string]string)
	for _, path := range envFiles {
		loadEnvFile(path, envFileVars)
	}

	// Build results
	var names []string
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)

	var envPairs []string
	var statuses []EnvVarStatus

	for _, name := range names {
		status := EnvVarStatus{Name: name}

		// Check host env first, then .env file
		if val, ok := os.LookupEnv(name); ok {
			status.Set = true
			status.Source = "host"
			envPairs = append(envPairs, name+"="+val)
		} else if val, ok := envFileVars[name]; ok {
			status.Set = true
			status.Source = ".env"
			envPairs = append(envPairs, name+"="+val)
		} else {
			status.Set = false
		}

		statuses = append(statuses, status)
	}

	return envPairs, statuses
}

// EnvVarStatus describes a detected environment variable and its availability.
type EnvVarStatus struct {
	Name   string
	Set    bool
	Source string // "host" or ".env"
}

// loadEnvFile parses a simple .env file (KEY=VALUE, one per line, # comments).
func loadEnvFile(path string, vars map[string]string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			// Strip surrounding quotes
			val = strings.Trim(val, `"'`)
			vars[key] = val
		}
	}
}
