package main

import (
	"fmt"
	"os"
)

const marker = "acp-path-openrouter-smoke"
const absoluteMarker = "acp-path-openrouter-absolute-smoke"

func initialize(client *acpClient) error {
	result, err := client.request("initialize", map[string]any{
		"protocolVersion":    1,
		"clientCapabilities": map[string]any{},
		"clientInfo":         map[string]string{"name": "acp-path smoke", "version": "1"},
	})
	if err != nil {
		return err
	}
	negotiated, ok := result["protocolVersion"].(float64)
	if !ok || int(negotiated) != 1 {
		return fmt.Errorf("agent negotiated ACP protocol version %v, want 1", result["protocolVersion"])
	}
	return nil
}

func createSession(client *acpClient, workspace string) (string, error) {
	result, err := client.request("session/new", map[string]any{"cwd": workspace, "mcpServers": []any{}})
	if err != nil {
		return "", err
	}
	sessionID, ok := result["sessionId"].(string)
	if !ok {
		return "", fmt.Errorf("session/new returned no sessionId")
	}
	return sessionID, nil
}

func askForMarkerFiles(client *acpClient, sessionID, absoluteMarkerPath string) error {
	prompt := "Create " + marker + ".txt directly in the current workspace containing exactly " + marker + ". Then create the file at " + absoluteMarkerPath + " containing exactly " + absoluteMarker + "."
	_, err := client.request("session/prompt", map[string]any{"sessionId": sessionID, "prompt": []any{map[string]string{"type": "text", "text": prompt}}})
	return err
}

func removePreviousMarker(target string) error {
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove prior marker: %w", err)
	}
	return nil
}

func checkMarkerFile(target, marker string) error {
	contents, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("read marker file: %w", err)
	}
	if !isMarkerContents(contents, marker) {
		return fmt.Errorf("marker contents = %q, want %q", contents, marker)
	}
	return nil
}

func isMarkerContents(contents []byte, marker string) bool {
	text := string(contents)
	return text == marker || text == marker+"\n" || text == marker+"\r\n"
}
