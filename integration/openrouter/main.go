package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const marker = "acp-path-openrouter-smoke"

const smokeTimeout = 100 * time.Minute
const shutdownTimeout = 30 * time.Second

type acpClient struct {
	input  io.WriteCloser
	output *bufio.Scanner
	nextID int
}

func main() {
	flag.Parse()

	if err := runSmoke(flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, "openrouter smoke:", err)
		os.Exit(1)
	}
}

func runSmoke(agentCommand []string) error {
	if len(agentCommand) == 0 {
		return fmt.Errorf("agent command is required after --")
	}
	workspace := os.Getenv("GITHUB_WORKSPACE")
	if workspace == "" {
		return fmt.Errorf("GITHUB_WORKSPACE is not set")
	}

	target := filepath.Join(workspace, marker+".txt")
	if err := removePreviousMarker(target); err != nil {
		return err
	}

	smokeContext, cancel := context.WithTimeout(context.Background(), smokeTimeout)
	defer cancel()

	client, command, err := startAgent(smokeContext, agentCommand)
	if err != nil {
		return err
	}
	defer stopAgent(command, client.input)

	if err := initialize(client); err != nil {
		return err
	}
	sessionID, err := createSession(client, workspace)
	if err != nil {
		return err
	}
	if err := askForMarkerFile(client, sessionID); err != nil {
		return err
	}
	return checkMarkerFile(target)
}

func startAgent(smokeContext context.Context, arguments []string) (*acpClient, *exec.Cmd, error) {
	command := exec.CommandContext(smokeContext, arguments[0], arguments[1:]...)
	command.Stderr = os.Stderr
	input, err := command.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("open agent stdin: %w", err)
	}
	output, err := command.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("open agent stdout: %w", err)
	}
	if err := command.Start(); err != nil {
		return nil, nil, fmt.Errorf("start agent: %w", err)
	}
	scanner := bufio.NewScanner(output)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	return &acpClient{input: input, output: scanner, nextID: 1}, command, nil
}

func stopAgent(command *exec.Cmd, input io.Closer) {
	_ = input.Close()
	done := make(chan error, 1)
	go func() {
		done <- command.Wait()
	}()
	select {
	case <-done:
	case <-time.After(shutdownTimeout):
		_ = command.Process.Kill()
		<-done
	}
}

func initialize(client *acpClient) error {
	_, err := client.request("initialize", map[string]any{
		"protocolVersion":    1,
		"clientCapabilities": map[string]any{},
		"clientInfo":         map[string]string{"name": "acp-path smoke", "version": "1"},
	})
	return err
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

func askForMarkerFile(client *acpClient, sessionID string) error {
	prompt := "Create " + marker + ".txt in the current workspace containing exactly " + marker + "."
	_, err := client.request("session/prompt", map[string]any{"sessionId": sessionID, "prompt": []any{map[string]string{"type": "text", "text": prompt}}})
	return err
}

func removePreviousMarker(target string) error {
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove prior marker: %w", err)
	}
	return nil
}
func checkMarkerFile(target string) error {
	contents, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("read marker file: %w", err)
	}
	if string(contents) != marker {
		return fmt.Errorf("marker contents = %q, want %q", contents, marker)
	}
	return nil
}

func (client *acpClient) request(method string, params map[string]any) (map[string]any, error) {
	id := client.nextID
	client.nextID++
	if err := client.send(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
		return nil, err
	}
	for client.output.Scan() {
		var message map[string]any
		if err := json.Unmarshal(client.output.Bytes(), &message); err != nil {
			return nil, fmt.Errorf("decode ACP message: %w", err)
		}
		if message["id"] == float64(id) {
			return responseResult(method, message)
		}
		if err := client.handleAgentRequest(message); err != nil {
			return nil, err
		}
	}
	if err := client.output.Err(); err != nil {
		return nil, fmt.Errorf("read ACP output: %w", err)
	}
	return nil, fmt.Errorf("agent closed ACP stdout")
}

func responseResult(method string, message map[string]any) (map[string]any, error) {
	if failure, ok := message["error"]; ok {
		return nil, fmt.Errorf("%s failed: %v", method, failure)
	}
	result, ok := message["result"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s returned a non-object result", method)
	}
	return result, nil
}
func (client *acpClient) handleAgentRequest(message map[string]any) error {
	id, hasID := message["id"]
	method, hasMethod := message["method"].(string)
	if !hasID || !hasMethod {
		return nil
	}
	if method == "session/request_permission" {
		return client.grantPermission(id, message["params"])
	}
	return client.send(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": -32601, "message": "unsupported by smoke test"}})
}

func (client *acpClient) grantPermission(id any, value any) error {
	params, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("permission request has non-object params")
	}
	options, ok := params["options"].([]any)
	if !ok {
		return fmt.Errorf("permission request has no options")
	}
	for _, value := range options {
		option, ok := value.(map[string]any)
		if !ok || option["kind"] != "allow_once" {
			continue
		}
		optionID, ok := option["optionId"].(string)
		if !ok {
			return fmt.Errorf("allow_once permission option has no optionId")
		}
		return client.send(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"outcome": map[string]string{"outcome": "selected", "optionId": optionID}}})
	}
	return fmt.Errorf("permission request has no allow_once option")
}

func (client *acpClient) send(message map[string]any) error {
	encoded, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = client.input.Write(append(encoded, '\n'))
	return err
}
