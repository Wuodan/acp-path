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
	"runtime"
	"testing"
)

func TestRunHelp(t *testing.T) {
	for _, argument := range []string{"-h", "--help"} {
		t.Run(argument, func(t *testing.T) {
			var output, stderr bytes.Buffer
			if err := run([]string{argument}, nil, &output, &stderr); err != nil {
				t.Fatal(err)
			}
			if output.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", output.String())
			}
			if !bytes.Contains(stderr.Bytes(), []byte("Usage of acp-wsl:")) {
				t.Fatalf("stderr = %q, want usage", stderr.String())
			}
		})
	}
}

func TestRunVersion(t *testing.T) {
	var output, stderr bytes.Buffer
	if err := run([]string{"--version"}, nil, &output, &stderr); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "acp-wsl "+version+"\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunDoesNotAcceptLogFile(t *testing.T) {
	err := run([]string{"--log-file", "traffic.log", "--", "agent"}, nil, io.Discard, io.Discard)
	if err == nil || err.Error() != "flag provided but not defined: -log-file" {
		t.Fatalf("run error = %v, want undefined log-file flag", err)
	}
}

func TestCompiledBinaryForwardsClientMessageToAgent(t *testing.T) {
	captureFile := filepath.Join(t.TempDir(), "agent-input.json")
	t.Setenv("ACP_PATH_TEST_AGENT", "capture")
	t.Setenv("ACP_PATH_TEST_CAPTURE", captureFile)

	command := exec.Command(
		buildBinary(t),
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
	name := "acp-wsl"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(t.TempDir(), name)
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build acp-wsl: %v\n%s", err, output)
	}
	return binary
}

func sameJSON(first, second []byte) bool {
	var one, two any
	return json.Unmarshal(first, &one) == nil && json.Unmarshal(second, &two) == nil && reflect.DeepEqual(one, two)
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

func TestRunUnknownPathFallbackFlag(t *testing.T) {
	for _, test := range []struct {
		name  string
		flags []string
		want  string
	}{
		{"default", nil, "/mnt/c/work"},
		{"enabled", []string{"--unknown-fields=true"}, "/mnt/c/work"},
		{"disabled", []string{"--unknown-fields=false"}, `C:\work`},
	} {
		t.Run(test.name, func(t *testing.T) {
			captureFile := filepath.Join(t.TempDir(), "input.json")
			t.Setenv("ACP_PATH_TEST_AGENT", "capture")
			t.Setenv("ACP_PATH_TEST_CAPTURE", captureFile)
			args := append([]string{}, test.flags...)
			args = append(args, "--", os.Args[0])
			input := bytes.NewBufferString(`{"id":4,"method":"skills/extraRoots/set","params":{"extraRoots":["C:\\work"]}}` + "\n")
			var stdout, stderr bytes.Buffer
			if err := run(args, input, &stdout, &stderr); err != nil {
				t.Fatal(err)
			}
			received, err := os.ReadFile(captureFile)
			if err != nil {
				t.Fatal(err)
			}
			var message struct{ Params struct{ ExtraRoots []string } }
			if err := json.Unmarshal(received, &message); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(message.Params.ExtraRoots, []string{test.want}) {
				t.Fatalf("received %s, want path %q", received, test.want)
			}
			if stdout.Len() != 0 || stderr.Len() != 0 {
				t.Fatalf("unexpected output: stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
		})
	}
}
