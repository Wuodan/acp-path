package transform

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestPathsInKnownV1MethodFields(t *testing.T) {
	tests := []struct {
		name, input, want string
		direction         Direction
	}{
		{"new session", `{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"C:\\work\\project","additionalDirectories":["D:\\lib","src\\local.go"]}}`, `{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"/mnt/c/work/project","additionalDirectories":["/mnt/d/lib","src/local.go"]}}`, ToLinux},
		{"read text", `{"jsonrpc":"2.0","id":1,"method":"fs/read_text_file","params":{"path":"/mnt/c/work/file.go"}}`, `{"jsonrpc":"2.0","id":1,"method":"fs/read_text_file","params":{"path":"C:\\work\\file.go"}}`, ToWindows},
		{"terminal", `{"jsonrpc":"2.0","id":1,"method":"terminal/create","params":{"cwd":"/mnt/d/work"}}`, `{"jsonrpc":"2.0","id":1,"method":"terminal/create","params":{"cwd":"D:\\work"}}`, ToWindows},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, _ := New().Transform([]byte(test.input), test.direction)
			if !sameJSON(got, []byte(test.want)) {
				t.Fatalf("got %s, want %s", got, test.want)
			}
		})
	}
}

func TestContentFileURIsAndV2Diffs(t *testing.T) {
	mapper := New()
	_, _ = mapper.Transform([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":2}}`), ToLinux)
	input := []byte(`{"jsonrpc":"2.0","method":"session/update","params":{"update":{"sessionUpdate":"tool_call_update","locations":[{"path":"/mnt/c/a.go"}],"content":[{"type":"diff","changes":[{"operation":"move","oldPath":"/mnt/c/old.go","path":"/mnt/d/new.go"}]},{"type":"content","content":{"type":"resource_link","name":"x","uri":"file:///mnt/c/readme.md"}}]}}}`)
	want := `{"jsonrpc":"2.0","method":"session/update","params":{"update":{"sessionUpdate":"tool_call_update","locations":[{"path":"C:\\a.go"}],"content":[{"type":"diff","changes":[{"operation":"move","oldPath":"C:\\old.go","path":"D:\\new.go"}]},{"type":"content","content":{"type":"resource_link","name":"x","uri":"file:///C:/readme.md"}}]}}}`
	got, _ := mapper.Transform(input, ToWindows)
	if !sameJSON(got, []byte(want)) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestUnsupportedMalformedAndAmbiguousValuesPassThrough(t *testing.T) {
	mapper := New()
	for _, input := range [][]byte{
		[]byte(`{"jsonrpc":"2.0","method":"_extension","params":{"path":"C:\\must-not-change"}}`),
		[]byte(`{"jsonrpc":"2.0","method":"session/new","params":{"cwd":"C:relative\\file"}}`),
		[]byte(`{"jsonrpc":"2.0","method":"session/new","params":{"cwd":"\\rooted"}}`),
		[]byte(`{"jsonrpc":"2.0","method":"session/new","params":{"cwd":"\\\\server\\share"}}`),
		[]byte(`not json`),
	} {
		got, _ := mapper.Transform(input, ToLinux)
		if !bytes.Equal(got, input) {
			t.Fatalf("got %q, want unchanged %q", got, input)
		}
	}
}

func TestSessionListResponseUsesRequestCorrelation(t *testing.T) {
	mapper := New()
	_, _ = mapper.Transform([]byte(`{"jsonrpc":"2.0","id":"list","method":"session/list","params":{}}`), ToLinux)
	got, _ := mapper.Transform([]byte(`{"jsonrpc":"2.0","id":"list","result":{"sessions":[{"cwd":"/mnt/c/work","additionalDirectories":["/mnt/d/lib"]}]}}`), ToWindows)
	want := `{"jsonrpc":"2.0","id":"list","result":{"sessions":[{"cwd":"C:\\work","additionalDirectories":["D:\\lib"]}]}}`
	if !sameJSON(got, []byte(want)) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestV1PermissionToolCallPaths(t *testing.T) {
	input := []byte(`{"jsonrpc":"2.0","id":4,"method":"session/request_permission","params":{"toolCall":{"locations":[{"path":"/mnt/c/file.go"}],"content":[{"type":"diff","path":"/mnt/c/file.go","newText":"x"}]}}}`)
	got, _ := New().Transform(input, ToWindows)
	want := []byte(`{"jsonrpc":"2.0","id":4,"method":"session/request_permission","params":{"toolCall":{"locations":[{"path":"C:\\file.go"}],"content":[{"type":"diff","path":"C:\\file.go","newText":"x"}]}}}`)
	if !sameJSON(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestV1MCPStdioCommandPath(t *testing.T) {
	input := []byte(`{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"C:\\work","mcpServers":[{"name":"local","command":"C:\\tools\\server.exe","args":[],"env":[]}]}}`)
	validateACPFixture(t, 1, input)
	want := []byte(`{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"/mnt/c/work","mcpServers":[{"name":"local","command":"/mnt/c/tools/server.exe","args":[],"env":[]}]}}`)
	got, _ := New().Transform(input, ToLinux)
	if !sameJSON(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestV1MCPHTTPCommandExtensionPassesThrough(t *testing.T) {
	input := []byte(`{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"C:\\work","mcpServers":[{"type":"http","name":"remote","url":"https://example.test/mcp","headers":[],"command":"C:\\must-not-change.exe"}]}}`)
	validateACPFixture(t, 1, input)
	want := []byte(`{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"/mnt/c/work","mcpServers":[{"type":"http","name":"remote","url":"https://example.test/mcp","headers":[],"command":"C:\\must-not-change.exe"}]}}`)
	got, _ := New().Transform(input, ToLinux)
	if !sameJSON(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestV2CommandPermissionCWD(t *testing.T) {
	mapper := New()
	mapper.Transform([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":2}}`), ToLinux)
	input := []byte(`{"jsonrpc":"2.0","id":2,"method":"session/request_permission","params":{"sessionId":"session","title":"Run command","options":[{"optionId":"allow","name":"Allow once","kind":"allow_once"}],"subject":{"type":"command","command":"pwd","cwd":"C:\\work"}}}`)
	validateACPFixture(t, 2, input)
	want := []byte(`{"jsonrpc":"2.0","id":2,"method":"session/request_permission","params":{"sessionId":"session","title":"Run command","options":[{"optionId":"allow","name":"Allow once","kind":"allow_once"}],"subject":{"type":"command","command":"pwd","cwd":"/mnt/c/work"}}}`)
	got, _ := mapper.Transform(input, ToLinux)
	if !sameJSON(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestV2ForkMCPStdioCommandPath(t *testing.T) {
	mapper := New()
	mapper.Transform([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":2}}`), ToLinux)
	input := []byte(`{"jsonrpc":"2.0","id":2,"method":"session/fork","params":{"sessionId":"source","cwd":"C:\\work","mcpServers":[{"type":"stdio","name":"local","command":"C:\\tools\\server.exe","args":[]}]}}`)
	validateACPFixture(t, 2, input)
	want := []byte(`{"jsonrpc":"2.0","id":2,"method":"session/fork","params":{"sessionId":"source","cwd":"/mnt/c/work","mcpServers":[{"type":"stdio","name":"local","command":"/mnt/c/tools/server.exe","args":[]}]}}`)
	got, _ := mapper.Transform(input, ToLinux)
	if !sameJSON(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestV2PlanFileURI(t *testing.T) {
	mapper := New()
	mapper.Transform([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":2}}`), ToLinux)
	input := []byte(`{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"session","update":{"sessionUpdate":"plan_update","plan":{"type":"file","planId":"plan","uri":"file:///C:/work/plan.md"}}}}`)
	validateACPFixture(t, 2, input)
	want := []byte(`{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"session","update":{"sessionUpdate":"plan_update","plan":{"type":"file","planId":"plan","uri":"file:///mnt/c/work/plan.md"}}}}`)
	got, _ := mapper.Transform(input, ToLinux)
	if !sameJSON(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func sameJSON(first, second []byte) bool {
	var one, two any
	return json.Unmarshal(first, &one) == nil && json.Unmarshal(second, &two) == nil && reflect.DeepEqual(one, two)
}
