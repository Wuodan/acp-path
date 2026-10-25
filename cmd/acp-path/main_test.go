package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRunForwardsOnlyProtocolDataAndWritesLog(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "traffic.log")
	t.Setenv("ACP_PATH_HELPER", "1")
	input := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"C:\\work"}}` + "\n")
	var output, stderr bytes.Buffer
	err := run([]string{"--log-file", logFile, "--", os.Args[0], "-test.run=TestHelperProcess", "--"}, input, &output, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"jsonrpc":"2.0","id":9,"method":"fs/read_text_file","params":{"path":"C:\\work\\result.go"}}` + "\n"
	if !sameJSON(output.Bytes(), []byte(want)) {
		t.Fatalf("stdout = %q, want %q", output.String(), want)
	}
	contents, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(contents, []byte("client->agent")) || !bytes.Contains(contents, []byte("subprocess exited successfully")) {
		t.Fatalf("log missing required events: %s", contents)
	}
}

func TestCompiledBinaryForwardsClientMessageToAgent(t *testing.T) {
	captureFile := filepath.Join(t.TempDir(), "agent-input.json")
	logFile := filepath.Join(t.TempDir(), "traffic.log")
	t.Setenv("ACP_PATH_TEST_AGENT", "capture")
	t.Setenv("ACP_PATH_TEST_CAPTURE", captureFile)

	command := exec.Command(
		buildBinary(t),
		"--log-file", logFile,
		"--",
		os.Args[0],
	)
	command.Stdin = bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"C:\\work\\project"}}` + "\n")
	var stdout bytes.Buffer
	command.Stdout = &stdout
	if err := command.Run(); err != nil {
		t.Fatalf("run compiled bridge: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want no agent response", stdout.String())
	}
	received, err := os.ReadFile(captureFile)
	if err != nil {
		t.Fatalf("read fake-agent input: %v", err)
	}
	want := []byte(`{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"/mnt/c/work/project"}}` + "\n")
	if !sameJSON(received, want) {
		t.Fatalf("agent input = %s, want %s", received, want)
	}
}

func TestCompiledBinaryForwardsAgentMessageToClient(t *testing.T) {
	t.Setenv("ACP_PATH_TEST_AGENT", "emit")
	command := exec.Command(
		buildBinary(t),
		"--log-file", filepath.Join(t.TempDir(), "traffic.log"),
		"--",
		os.Args[0],
	)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("run compiled bridge: %v", err)
	}
	want := []byte(`{"jsonrpc":"2.0","id":2,"method":"fs/read_text_file","params":{"path":"C:\\work\\project\\result.go"}}` + "\n")
	if !sameJSON(output, want) {
		t.Fatalf("client output = %s, want %s", output, want)
	}
}

func buildBinary(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "acp-path")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build acp-path: %v\n%s", err, output)
	}
	return binary
}

func sameJSON(first, second []byte) bool {
	var one, two any
	return json.Unmarshal(first, &one) == nil && json.Unmarshal(second, &two) == nil && reflect.DeepEqual(one, two)
}

func TestHelperProcess(_ *testing.T) {
	if os.Getenv("ACP_PATH_HELPER") != "1" {
		return
	}
	line, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(2)
	}
	if !bytes.Contains(line, []byte(`/mnt/c/work`)) {
		fmt.Fprintln(os.Stderr, "input was not transformed")
		os.Exit(3)
	}
	fmt.Fprintln(os.Stdout, `{"jsonrpc":"2.0","id":9,"method":"fs/read_text_file","params":{"path":"/mnt/c/work/result.go"}}`)
	os.Exit(0)
}

func TestMain(m *testing.M) {
	if os.Getenv("ACP_PATH_TEST_AGENT") != "" {
		os.Exit(runFakeAgent())
	}
	os.Exit(m.Run())
}

// runFakeAgent is a one-message downstream ACP peer for compiled-binary
// tests. It either captures one request or emits one notification and exits.
func runFakeAgent() int {
	switch os.Getenv("ACP_PATH_TEST_AGENT") {
	case "capture":
		line, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "read bridge input:", err)
			return 1
		}
		if err := os.WriteFile(os.Getenv("ACP_PATH_TEST_CAPTURE"), line, 0o600); err != nil {
			fmt.Fprintln(os.Stderr, "capture bridge input:", err)
			return 1
		}
	case "emit":
		fmt.Fprintln(os.Stdout, `{"jsonrpc":"2.0","id":2,"method":"fs/read_text_file","params":{"path":"/mnt/c/work/project/result.go"}}`)
	}
	return 0
}
