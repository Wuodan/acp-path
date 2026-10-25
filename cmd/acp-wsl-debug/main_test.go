package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestRunWritesRawTrafficLog(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "traffic.log")
	t.Setenv("ACP_PATH_DEBUG_HELPER", "1")
	input := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"C:\\work"}}` + "\n")
	var output, stderr bytes.Buffer
	err := run([]string{"--log-file", logFile, "--", os.Args[0], "-test.run=TestHelperProcess", "--"}, input, &output, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(contents, []byte("client->agent")) || !bytes.Contains(contents, []byte("subprocess exited successfully")) {
		t.Fatalf("log missing required events: %s", contents)
	}
}

func TestRunWithoutLogFileForwardsTraffic(t *testing.T) {
	t.Setenv("ACP_PATH_DEBUG_HELPER", "1")
	input := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"C:\\work"}}` + "\n")
	var output, stderr bytes.Buffer
	if err := run([]string{"--", os.Args[0], "-test.run=TestHelperProcess", "--"}, input, &output, &stderr); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output.Bytes(), []byte(`C:\\work\\result.go`)) {
		t.Fatalf("output = %q, want mapped agent response", output.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunCommandLineBehavior(t *testing.T) {
	t.Run("help", func(t *testing.T) {
		var output, stderr bytes.Buffer
		if err := run([]string{"--help"}, nil, &output, &stderr); err != nil {
			t.Fatal(err)
		}
		if output.Len() != 0 || !bytes.Contains(stderr.Bytes(), []byte("Usage of acp-wsl-debug:")) {
			t.Fatalf("stdout=%q stderr=%q, want help on stderr", output.String(), stderr.String())
		}
	})

	t.Run("version", func(t *testing.T) {
		var output, stderr bytes.Buffer
		if err := run([]string{"--version"}, nil, &output, &stderr); err != nil {
			t.Fatal(err)
		}
		if got, want := output.String(), "acp-wsl-debug "+version+"\n"; got != want || stderr.Len() != 0 {
			t.Fatalf("stdout=%q stderr=%q, want %q and empty stderr", got, stderr.String(), want)
		}
	})

	for _, args := range [][]string{{"--unknown"}, nil, {"--"}} {
		t.Run("invalid command line", func(t *testing.T) {
			if err := run(args, nil, io.Discard, io.Discard); err == nil {
				t.Fatalf("run(%q) returned nil error", args)
			}
		})
	}
}

func TestOpenLogCreatesDirectoriesAndReportsFailures(t *testing.T) {
	t.Run("creates parent directories", func(t *testing.T) {
		name := filepath.Join(t.TempDir(), "logs", "traffic.log")
		file, err := openLog(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(name); err != nil {
			t.Fatalf("stat created log: %v", err)
		}
	})

	for _, name := range []string{
		filepath.Join(t.TempDir(), "\x00", "traffic.log"),
		t.TempDir(),
	} {
		if file, err := openLog(name); err == nil {
			file.Close()
			t.Fatalf("openLog(%q) returned nil error", name)
		}
	}
}

func TestHelperProcess(_ *testing.T) {
	if os.Getenv("ACP_PATH_DEBUG_HELPER") == "fallback" {
		line, err := io.ReadAll(os.Stdin)
		if err != nil {
			os.Exit(2)
		}
		if err := os.WriteFile(os.Getenv("ACP_PATH_DEBUG_CAPTURE"), line, 0o600); err != nil {
			os.Exit(3)
		}
		fmt.Fprintln(os.Stdout, `{"id":4,"result":{"root":"/mnt/c/work"}}`)
		os.Exit(0)
	}
	if os.Getenv("ACP_PATH_DEBUG_HELPER") != "1" {
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

func TestRunUnknownPathFallbackFlag(t *testing.T) {
	for _, test := range []struct {
		name                  string
		flags                 []string
		wantInput, wantOutput string
	}{
		{"default", nil, "/mnt/c/work", `C:\work`},
		{"disabled", []string{"--unknown-fields=false"}, `C:\work`, "/mnt/c/work"},
	} {
		t.Run(test.name, func(t *testing.T) {
			captureFile := filepath.Join(t.TempDir(), "input.json")
			logFile := filepath.Join(t.TempDir(), "traffic.log")
			t.Setenv("ACP_PATH_DEBUG_HELPER", "fallback")
			t.Setenv("ACP_PATH_DEBUG_CAPTURE", captureFile)
			args := append([]string{}, test.flags...)
			args = append(args, "--log-file", logFile, "--", os.Args[0], "-test.run=TestHelperProcess", "--")
			input := bytes.NewBufferString(`{"id":4,"method":"skills/extraRoots/set","params":{"extraRoots":["C:\\work"]}}` + "\n")
			var stdout, stderr bytes.Buffer
			if err := run(args, input, &stdout, &stderr); err != nil {
				t.Fatal(err)
			}
			received, err := os.ReadFile(captureFile)
			if err != nil {
				t.Fatal(err)
			}
			var request struct{ Params struct{ ExtraRoots []string } }
			if err := json.Unmarshal(received, &request); err != nil {
				t.Fatal(err)
			}
			var response struct{ Result struct{ Root string } }
			if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if len(request.Params.ExtraRoots) != 1 || request.Params.ExtraRoots[0] != test.wantInput || response.Result.Root != test.wantOutput {
				t.Fatalf("unexpected traffic: input=%s output=%s", received, stdout.Bytes())
			}
			contents, err := os.ReadFile(logFile)
			if err != nil {
				t.Fatal(err)
			}
			if test.name == "default" && (!bytes.Contains(contents, []byte("unknown-field-windows-drive")) || !bytes.Contains(contents, []byte("unknown-field-wsl-mount"))) {
				t.Fatalf("fallback decisions missing from log: %s", contents)
			}
		})
	}
}
