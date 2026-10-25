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

func TestHandleAgentMessageSelectsAllowOncePermission(t *testing.T) {
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

	if err := client.handleAgentMessage(message); err != nil {
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

func TestIsMarkerContents(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		want     bool
	}{
		{name: "exact", contents: marker, want: true},
		{name: "unix line ending", contents: marker + "\n", want: true},
		{name: "windows line ending", contents: marker + "\r\n", want: true},
		{name: "extra text", contents: marker + " extra", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isMarkerContents([]byte(test.contents), marker); got != test.want {
				t.Errorf("isMarkerContents(%q) = %t, want %t", test.contents, got, test.want)
			}
		})
	}
}
