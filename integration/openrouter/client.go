// Package main runs the OpenRouter integration smoke test.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

const shutdownTimeout = 30 * time.Second

type acpClient struct {
	input  io.WriteCloser
	output *bufio.Scanner
	nextID int
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

func (client *acpClient) request(method string, params map[string]any) (map[string]any, error) {
	id := client.nextID
	client.nextID++
	if err := client.send(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
		return nil, err
	}
	for {
		message, err := client.nextMessage()
		if err != nil {
			return nil, err
		}
		if message["id"] == float64(id) {
			return responseResult(method, message)
		}
		if err := client.handleAgentMessage(message); err != nil {
			return nil, err
		}
	}
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

func (client *acpClient) nextMessage() (map[string]any, error) {
	if !client.output.Scan() {
		if err := client.output.Err(); err != nil {
			return nil, fmt.Errorf("read ACP output: %w", err)
		}
		return nil, fmt.Errorf("agent closed ACP stdout")
	}
	var message map[string]any
	if err := json.Unmarshal(client.output.Bytes(), &message); err != nil {
		return nil, fmt.Errorf("decode ACP message: %w", err)
	}
	return message, nil
}

func (client *acpClient) handleAgentMessage(message map[string]any) error {
	id, hasID := message["id"]
	method, hasMethod := message["method"].(string)
	if !hasMethod {
		return nil
	}
	if !hasID {
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
