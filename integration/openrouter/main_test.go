package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

type recordingWriteCloser struct {
	bytes.Buffer
}

func (recordingWriteCloser) Close() error {
	return nil
}

func TestHandleAgentRequestSelectsAllowOncePermission(t *testing.T) {
	output := &recordingWriteCloser{}
	client := acpClient{input: output}
	message := map[string]any{
		"jsonrpc": "2.0",
		"id":      7,
		"method":  "session/request_permission",
		"params": map[string]any{"options": []any{
			map[string]any{"optionId": "deny", "kind": "reject_once"},
			map[string]any{"optionId": "approve", "kind": "allow_once"},
		}},
	}

	if err := client.handleAgentRequest(message); err != nil {
		t.Fatal(err)
	}
	var response map[string]any
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"jsonrpc": "2.0", "id": float64(7), "result": map[string]any{"outcome": map[string]any{"outcome": "selected", "optionId": "approve"}}}
	if !reflect.DeepEqual(response, want) {
		t.Errorf("response = %#v, want %#v", response, want)
	}
}
