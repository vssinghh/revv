package adk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	apiKey string
	model  string
}

func NewClient(apiKey, model string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is not set")
	}
	return &Client{
		apiKey: apiKey,
		model:  model,
	}, nil
}

type GeminiSchema struct {
	Type                 string                  `json:"type"`
	Properties           map[string]GeminiSchema `json:"properties,omitempty"`
	Required             []string                `json:"required,omitempty"`
	Items                *GeminiSchema           `json:"items,omitempty"`
	Enum                 []string                `json:"enum,omitempty"`
	Description          string                  `json:"description,omitempty"`
}

type GenerationConfig struct {
	ResponseMimeType string        `json:"responseMimeType,omitempty"`
	ResponseSchema   *GeminiSchema `json:"responseSchema,omitempty"`
}

type Part struct {
	Text string `json:"text"`
}

type Content struct {
	Parts []Part `json:"parts"`
}

type RequestBody struct {
	Contents         []Content         `json:"contents"`
	GenerationConfig *GenerationConfig `json:"generationConfig,omitempty"`
}

type ResponseBody struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.model, c.apiKey)

	actionField := GeminiSchema{
		Type: "STRING",
		Enum: []string{"add", "update", "keep", "delete"},
		Description: "Action to perform: add (new item), update (modify existing), keep (no change), delete (remove)",
	}

	schema := &GeminiSchema{
		Type: "OBJECT",
		Properties: map[string]GeminiSchema{
			"dockerfile": {Type: "STRING"},
			"helpers": {
				Type: "ARRAY",
				Items: &GeminiSchema{
					Type: "OBJECT",
					Properties: map[string]GeminiSchema{
						"filename": {Type: "STRING"},
						"content":  {Type: "STRING"},
						"action":   actionField,
					},
					Required: []string{"filename", "content", "action"},
				},
			},
			"tests": {
				Type: "ARRAY",
				Items: &GeminiSchema{
					Type: "OBJECT",
					Properties: map[string]GeminiSchema{
						"category": {Type: "STRING"},
						"name":     {Type: "STRING"},
						"test_md":  {Type: "STRING"},
						"action":   actionField,
						"helpers": {
							Type: "ARRAY",
							Items: &GeminiSchema{
								Type: "OBJECT",
								Properties: map[string]GeminiSchema{
									"filename": {Type: "STRING"},
									"content":  {Type: "STRING"},
									"action":   actionField,
								},
								Required: []string{"filename", "content", "action"},
							},
						},
					},
					Required: []string{"category", "name", "test_md", "action"},
				},
			},
		},
		Required: []string{"dockerfile", "helpers", "tests"},
	}

	reqBody := RequestBody{
		Contents: []Content{
			{
				Parts: []Part{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: &GenerationConfig{
			ResponseMimeType: "application/json",
			ResponseSchema:   schema,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned non-200 status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp ResponseBody
	if err := json.Unmarshal(bodyBytes, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response candidate from Gemini")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func ConstructPrompt(repoContext map[string]string, existingConfig map[string]string) string {
	var sb strings.Builder
	sb.WriteString("You are an expert software developer and system integrator. We are configuring a testing tool called `revv` for this repository.\n")
	sb.WriteString("Here is the context of the repository gathered from files:\n\n")
	for name, content := range repoContext {
		sb.WriteString(fmt.Sprintf("--- File: %s ---\n", name))
		sb.WriteString(content)
		sb.WriteString("\n\n")
	}

	if len(existingConfig) > 0 {
		sb.WriteString("\n--- EXISTING .revv/ CONFIGURATION ---\n")
		sb.WriteString("This repository already has a .revv/ configuration. Review it against the current repo state and produce an incremental update.\n")
		sb.WriteString("For each item, set the 'action' field to one of: add, update, keep, delete.\n")
		sb.WriteString("- 'keep': item is unchanged, no modifications needed\n")
		sb.WriteString("- 'update': item exists but needs changes (provide updated content)\n")
		sb.WriteString("- 'add': new item that doesn't exist yet\n")
		sb.WriteString("- 'delete': item is obsolete and should be removed (content can be empty)\n\n")
		for path, content := range existingConfig {
			sb.WriteString(fmt.Sprintf("--- .revv/%s ---\n", path))
			sb.WriteString(content)
			sb.WriteString("\n\n")
		}
	} else {
		sb.WriteString("No existing .revv/ configuration found. This is a fresh initialization.\n")
		sb.WriteString("Set the 'action' field to 'add' for all items.\n\n")
	}

	sb.WriteString("Based on the repository structure, language, and files provided, please generate:\n")
	sb.WriteString("1. A Dockerfile (dockerfile) that sets up the correct build and test dependencies for this project.\n")
	sb.WriteString("2. Global helpers (helpers) as an array of objects, each with 'filename', 'content', and 'action' fields.\n")
	sb.WriteString("3. A list of tests (tests), where each test has a category, name, test_md, action, and helpers (array of {filename, content, action} objects).\n")
	sb.WriteString("Ensure a 'manual' category is always generated in the tests array.\n")
	sb.WriteString("The output must be a JSON object conforming to the response schema, containing:\n")
	sb.WriteString(" - dockerfile (string)\n")
	sb.WriteString(" - helpers (array of {filename, content, action} objects for global shared helpers)\n")
	sb.WriteString(" - tests (array of objects with category, name, test_md, action, and helpers array)\n\n")
	sb.WriteString("Each test's test_md MUST be formatted as markdown containing these section headers:\n")
	sb.WriteString("## Description\n## Priority\n## Commands\n## Expected Output\n")
	return sb.String()
}
