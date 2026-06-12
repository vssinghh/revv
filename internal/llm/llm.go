package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/vipinsingh/revv/internal/adk"
)

type ConfigOutput struct {
	Dockerfile string
	Helpers    map[string]string
	Tests      []TestInfo
}

type TestInfo struct {
	Category string
	Name     string
	TestMD   string
	Helpers  map[string]string
	Action   string // add, update, keep, delete
}

type HelperInfo struct {
	Filename string
	Content  string
	Action   string
}

type helperEntry struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
	Action   string `json:"action"`
}

type adkResponse struct {
	Dockerfile string        `json:"dockerfile"`
	Helpers    []helperEntry `json:"helpers"`
	Tests      []struct {
		Category string        `json:"category"`
		Name     string        `json:"name"`
		TestMD   string        `json:"test_md"`
		Action   string        `json:"action"`
		Helpers  []helperEntry `json:"helpers"`
	} `json:"tests"`
}

func GenerateConfig(ctx context.Context, modelName string, repoContext map[string]string, existingConfig map[string]string) (*ConfigOutput, error) {
	// If mock mode is requested, use the mock implementation
	if os.Getenv("REVV_MOCK_LLM") == "true" {
		dockerfileContent := "FROM golang:1.26.4-alpine\nRUN apk add --no-cache git make\nWORKDIR /workspace\n"

		// Support simulating large Dockerfile via environment variable
		if os.Getenv("REVV_MOCK_LARGE_DOCKERFILE") == "true" {
			buf := make([]byte, 1024*1024)
			for i := range buf {
				buf[i] = 'A'
			}
			dockerfileContent = string(buf)
		}

		helpers := map[string]string{
			"helpers/check.sh": "#!/bin/sh\necho \"Running repository checks...\"\n",
		}

		var tests []TestInfo

		// Check if empty config is requested
		if modelName == "empty-config" || os.Getenv("REVV_MOCK_EMPTY") == "true" {
			tests = []TestInfo{}
		} else if modelName == "special-chars" || os.Getenv("REVV_MOCK_SPECIAL_CHARS") == "true" {
			tests = []TestInfo{
				{
					Category: "special@category",
					Name:     "special#test",
					TestMD:   "# Special Characters Test\n## Description\nThis test checks special characters handling.\n## Priority\nMedium\n## Commands\necho 'special'\n## Expected Output\nspecial\n",
					Helpers: map[string]string{
						"special_helper.sh": "#!/bin/sh\necho special\n",
					},
				},
			}
		} else {
			// Standard rich configuration containing multiple categories
			tests = []TestInfo{
				{
					Category: "unit",
					Name:     "unit_test",
					TestMD:   "# Unit Test Case\n## Description\nVerify unit tests pass.\n## Priority\nHigh\n## Commands\ngo test -v ./...\n## Expected Output\nPASS\n",
					Helpers: map[string]string{
						"unit_runner.sh": "#!/bin/sh\ngo test ./...\n",
					},
				},
				{
					Category: "integration",
					Name:     "integration_test",
					TestMD:   "# Integration Test Case\n## Description\nVerify integration tests pass.\n## Priority\nMedium\n## Commands\ngo test -tags=integration ./...\n## Expected Output\nPASS\n",
					Helpers: map[string]string{
						"integration_runner.sh": "#!/bin/sh\necho integration\n",
					},
				},
				{
					Category: "lint",
					Name:     "lint_test",
					TestMD:   "# Lint Test Case\n## Description\nVerify code linting.\n## Priority\nLow\n## Commands\ngolangci-lint run\n## Expected Output\nclean\n",
					Helpers: map[string]string{
						"lint_runner.sh": "#!/bin/sh\necho lint\n",
					},
				},
				{
					Category: "manual",
					Name:     "manual_test",
					TestMD:   "# Manual Verification\n## Description\nPerform manual sanity check.\n## Priority\nLow\n## Commands\n./bin/revv --help\n## Expected Output\nUsage of revv:\n",
					Helpers: map[string]string{
						"manual_helper.sh": "#!/bin/sh\necho manual\n",
					},
				},
				{
					Category: "build",
					Name:     "build_test",
					TestMD:   "# Build Verification\n## Description\nVerify build compiles.\n## Priority\nHigh\n## Commands\nmake build\n## Expected Output\nsuccess\n",
					Helpers: map[string]string{
						"build_check.sh": "#!/bin/sh\nmake build\n",
					},
				},
			}
		}

		output := &ConfigOutput{
			Dockerfile: dockerfileContent,
			Helpers:    helpers,
			Tests:      tests,
		}
		return output, nil
	}

	// Real ADK/Gemini API Client integration
	apiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is not set")
	}

	client, err := adk.NewClient(apiKey, modelName)
	if err != nil {
		return nil, err
	}

	prompt := adk.ConstructPrompt(repoContext, existingConfig)
	responseJSON, err := client.Generate(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var parsed adkResponse
	if err := json.Unmarshal([]byte(responseJSON), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini response JSON: %w (raw response: %s)", err, responseJSON)
	}

	globalHelpers := make(map[string]string)
	for _, h := range parsed.Helpers {
		globalHelpers[h.Filename] = h.Content
	}

	var tests []TestInfo
	for _, t := range parsed.Tests {
		testHelpers := make(map[string]string)
		for _, h := range t.Helpers {
			testHelpers[h.Filename] = h.Content
		}
		action := t.Action
		if action == "" {
			action = "add"
		}
		tests = append(tests, TestInfo{
			Category: t.Category,
			Name:     t.Name,
			TestMD:   t.TestMD,
			Helpers:  testHelpers,
			Action:   action,
		})
	}

	return &ConfigOutput{
		Dockerfile: parsed.Dockerfile,
		Helpers:    globalHelpers,
		Tests:      tests,
	}, nil
}
