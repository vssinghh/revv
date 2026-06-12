package llm

import (
	"context"
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
}

func GenerateConfig(ctx context.Context, modelName string, repoContext map[string]string) (*ConfigOutput, error) {
	output := &ConfigOutput{
		Dockerfile: "FROM golang:1.26.4-alpine\nRUN apk add --no-cache git make\nWORKDIR /workspace\n",
		Helpers: map[string]string{
			"check.sh": "#!/bin/sh\necho \"Running repository checks...\"\n",
		},
		Tests: []TestInfo{
			{
				Category: "build",
				Name:     "build_test",
				TestMD:   "# Build Verification\nVerify that the binary builds successfully.\n",
				Helpers: map[string]string{
					"build_check.sh": "#!/bin/sh\nmake build\n",
				},
			},
		},
	}
	return output, nil
}
