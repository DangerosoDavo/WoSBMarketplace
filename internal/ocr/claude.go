package ocr

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type ClaudeClient struct {
	claudeCodePath string
}

func NewClaudeClient(claudeCodePath string) *ClaudeClient {
	// claudeCodePath should be the path to the claude executable
	// Default to "claude" if empty (assumes it's in PATH)
	if claudeCodePath == "" {
		claudeCodePath = "claude"
	}
	return &ClaudeClient{
		claudeCodePath: claudeCodePath,
	}
}

// AnalyzeScreenshot uses Claude Code CLI to analyze a market screenshot
func (c *ClaudeClient) AnalyzeScreenshot(ctx context.Context, imagePath string) (*MarketData, error) {
	// Construct the prompt for Claude Code
	prompt := fmt.Sprintf(`Please analyze the image at "%s" and extract market data.

This is a World of Sea Battle market screenshot. Extract the following information in JSON format:

{
  "port": "Port Name",
  "order_type": "buy" or "sell",
  "items": [
    {
      "name": "Item Name",
      "price": 123,
      "quantity": 456
    }
  ]
}

Instructions:
1. Identify the port name (usually shown at the top of the market interface)
2. Determine if this shows BUY orders or SELL orders (check which button is highlighted/active)
3. Extract each item row with:
   - Item name (exact spelling)
   - Price per unit (integer)
   - Quantity available (integer)
4. Return ONLY valid JSON in your response, no markdown code blocks or explanation
5. If you cannot determine the port or order type, set them to "unknown"
6. Ensure all item names are trimmed and properly capitalized

Please respond with ONLY the JSON object, nothing else.`, imagePath)

	// Execute Claude Code CLI
	// Use --dangerously-skip-console-check to run in non-interactive mode
	cmd := exec.CommandContext(ctx, c.claudeCodePath, "--dangerously-skip-console-check")
	cmd.Stdin = strings.NewReader(prompt)

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("claude code execution failed: %w (output: %s)", err, string(output))
	}

	// Parse the output
	outputStr := string(output)

	// Claude Code may include additional text, so we need to extract the JSON
	// Look for the JSON structure in the output
	jsonStart := strings.Index(outputStr, "{")
	jsonEnd := strings.LastIndex(outputStr, "}")

	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("no JSON found in claude code output: %s", outputStr)
	}

	jsonStr := outputStr[jsonStart : jsonEnd+1]

	// Clean up any markdown code blocks if present
	jsonStr = strings.TrimSpace(jsonStr)
	jsonStr = strings.TrimPrefix(jsonStr, "```json")
	jsonStr = strings.TrimPrefix(jsonStr, "```")
	jsonStr = strings.TrimSuffix(jsonStr, "```")
	jsonStr = strings.TrimSpace(jsonStr)

	// Parse the JSON response
	var marketData MarketData
	if err := json.Unmarshal([]byte(jsonStr), &marketData); err != nil {
		return nil, fmt.Errorf("failed to parse market data: %w (json: %s)", err, jsonStr)
	}

	// Validate
	if marketData.Port == "" || marketData.Port == "unknown" {
		return nil, fmt.Errorf("could not determine port from screenshot")
	}

	if marketData.OrderType != "buy" && marketData.OrderType != "sell" {
		return nil, fmt.Errorf("could not determine order type (buy/sell) from screenshot")
	}

	if len(marketData.Items) == 0 {
		return nil, fmt.Errorf("no items found in screenshot")
	}

	return &marketData, nil
}

// MarketData represents parsed market data from screenshot
type MarketData struct {
	Port      string       `json:"port"`
	OrderType string       `json:"order_type"`
	Items     []MarketItem `json:"items"`
}

// MarketItem represents a single market entry
type MarketItem struct {
	Name     string `json:"name"`
	Price    int    `json:"price"`
	Quantity int    `json:"quantity"`
}
