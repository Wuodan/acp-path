package transform

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestFallbackAbsolutePaths(t *testing.T) {
	tests := []struct {
		name, input, want string
		direction         Direction
	}{
		{
			"extra skill roots",
			`{"id":4,"method":"skills/extraRoots/set","params":{"extraRoots":["C:\\Users\\stefa\\skills"]}}`,
			`{"id":4,"method":"skills/extraRoots/set","params":{"extraRoots":["/mnt/c/Users/stefa/skills"]}}`,
			ToLinux,
		},
		{
			"whole string text value",
			`{"method":"_custom","params":{"text":"C:\\work"}}`,
			`{"method":"_custom","params":{"text":"/mnt/c/work"}}`,
			ToLinux,
		},
		{
			"Windows path with trailing separator",
			`{"method":"_custom","params":{"root":"C:\\work\\"}}`,
			`{"method":"_custom","params":{"root":"/mnt/c/work/"}}`,
			ToLinux,
		},
		{
			"WSL path with trailing separator",
			`{"method":"_custom","params":{"root":"/mnt/c/work/"}}`,
			`{"method":"_custom","params":{"root":"C:\\work\\"}}`,
			ToWindows,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, decisions := New(true).Transform([]byte(test.input), test.direction)
			if !sameJSON(got, []byte(test.want)) {
				t.Fatalf("got %s, want %s", got, test.want)
			}
			if len(decisions) == 0 {
				t.Fatal("missing conversion decisions")
			}
		})
	}
}

func TestUnknownFieldSkipsLargeStrings(t *testing.T) {
	path := `C:\` + strings.Repeat("a", maximumUnknownFieldPathBytes)
	input, err := json.Marshal(map[string]any{
		"method": "_custom",
		"params": map[string]any{"path": path},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, decisions := New(true).Transform(input, ToLinux)
	if !bytes.Equal(got, input) {
		t.Fatalf("got %q, want unchanged %q", got, input)
	}
	if len(decisions) != 0 {
		t.Fatalf("unexpected decisions: %+v", decisions)
	}
}

func TestUnknownFieldSkipsLargeNonPathText(t *testing.T) {
	text := strings.Repeat("x", maximumUnknownFieldPathBytes)
	input, err := json.Marshal(map[string]any{
		"method": "_custom",
		"params": map[string]any{"text": text},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, decisions := New(true).Transform(input, ToLinux)
	if !bytes.Equal(got, input) {
		t.Fatalf("got %q, want unchanged %q", got, input)
	}
	if len(decisions) != 0 {
		t.Fatalf("unexpected decisions: %+v", decisions)
	}
}

func TestKnownSchemaFieldsAreNotFallbackCandidates(t *testing.T) {
	mapper := New(true)
	_, _ = mapper.Transform([]byte(`{"id":1,"method":"initialize","params":{"protocolVersion":2}}`), ToLinux)
	input := []byte(`{"id":2,"method":"session/new","params":{"cwd":"C:\\work","systemPrompt":"C:\\do-not-rewrite"}}`)
	got, _ := mapper.Transform(input, ToLinux)
	want := []byte(`{"id":2,"method":"session/new","params":{"cwd":"/mnt/c/work","systemPrompt":"C:\\do-not-rewrite"}}`)
	if !sameJSON(got, want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestFallbackPreservesUnsupportedValuesAndRouting(t *testing.T) {
	for _, direction := range []Direction{ToLinux, ToWindows} {
		for _, input := range []string{
			` { "id":"C:\\request", "method":"/mnt/c/method", "params":{"relative":"src/file.go","windowsRelative":"src\\file.go","driveRelative":"C:foo","unc":"\\\\server\\share","rooted":"\\foo","linux":"/home/user","url":"https://host/path","uri":"file:///C:/file","text":"See C:\\work for details","multiline":"C:\\work\nmore text","number":42,"empty":null}} `,
			`{"path":"C:\\work"}`,
			`not json`,
			`{"method":"_custom","params":{"root":"C:\\work"}} {}`,
		} {
			got, _ := New(true).Transform([]byte(input), direction)
			if string(got) != input {
				t.Fatalf("got %s, want exact passthrough %s", got, input)
			}
		}
	}
}

func TestFallbackPreservesLargeNumbers(t *testing.T) {
	input := []byte(`{"id":9007199254740993,"method":"_custom","params":{"root":"C:\\work","number":9007199254740995}}`)
	got, _ := New(true).Transform(input, ToLinux)
	if !bytes.Contains(got, []byte(`"id":9007199254740993`)) ||
		!bytes.Contains(got, []byte(`"number":9007199254740995`)) ||
		!bytes.Contains(got, []byte(`"root":"/mnt/c/work"`)) {
		t.Fatalf("unexpected payload: %s", got)
	}
}

func TestFallbackCanBeDisabled(t *testing.T) {
	input := []byte(" {\"method\":\"skills/extraRoots/set\",\"params\":{\"extraRoots\":[\"C:\\\\skills\"]}}\r\n")
	got, _ := New(false).Transform(input, ToLinux)
	if !bytes.Equal(got, input) {
		t.Fatalf("got %q, want unchanged %q", got, input)
	}
	known := []byte(`{"method":"session/new","params":{"cwd":"C:\\work"}}`)
	got, _ = New(false).Transform(known, ToLinux)
	if !sameJSON(got, []byte(`{"method":"session/new","params":{"cwd":"/mnt/c/work"}}`)) {
		t.Fatalf("standard mapping disabled: %s", got)
	}
}

func TestFallbackBatchAndDecisionField(t *testing.T) {
	input := []byte("[{\"method\":\"_custom\",\"params\":{\"roots\":[\"C:\\\\work\"]}}]\r\n")
	got, decisions := New(true).Transform(input, ToLinux)
	if !sameJSON(got, []byte(`[{"method":"_custom","params":{"roots":["/mnt/c/work"]}}]`)) || !bytes.HasSuffix(got, []byte("\r\n")) {
		t.Fatalf("unexpected batch: %q", got)
	}
	if len(decisions) != 1 || decisions[0].Field != "params.roots[0]" || decisions[0].Kind != "unknown-field-windows-drive" {
		t.Fatalf("unexpected decisions: %+v", decisions)
	}
}

func TestUnknownFieldTraversalHonorsSchemaBoundaries(t *testing.T) {
	catalog := &schemaCatalog{}
	t.Run("rewrites unknown field in constrained object", func(t *testing.T) {
		value := map[string]any{
			"known": `C:\leave-alone`,
			"extra": `C:\work`,
		}
		schema := map[string]any{
			"properties": map[string]any{
				"known": map[string]any{"type": "string"},
			},
			"additionalProperties": false,
		}
		converted, decisions := transformUnknownValue(value, []map[string]any{schema}, catalog, ToLinux, "params")
		got := converted.(map[string]any)
		if got["known"] != `C:\leave-alone` || got["extra"] != "/mnt/c/work" {
			t.Fatalf("converted = %#v", got)
		}
		if len(decisions) != 1 || decisions[0].Field != "params.extra" {
			t.Fatalf("decisions = %+v", decisions)
		}
	})

	t.Run("rewrites unknown fields in constrained array items", func(t *testing.T) {
		value := map[string]any{
			"roots": []any{map[string]any{"extra": `C:\work`}},
		}
		schema := map[string]any{
			"properties": map[string]any{
				"roots": map[string]any{
					"items": map[string]any{"additionalProperties": false},
				},
			},
		}
		converted, decisions := transformUnknownValue(value, []map[string]any{schema}, catalog, ToLinux, "params")
		roots := converted.(map[string]any)["roots"].([]any)
		if got := roots[0].(map[string]any)["extra"]; got != "/mnt/c/work" {
			t.Fatalf("extra = %q, want /mnt/c/work", got)
		}
		if len(decisions) != 1 || decisions[0].Field != "params.roots[0].extra" {
			t.Fatalf("decisions = %+v", decisions)
		}
	})

	t.Run("preserves permitted additional properties", func(t *testing.T) {
		value := map[string]any{"extra": `C:\work`}
		schema := map[string]any{"additionalProperties": true}
		converted, decisions := transformUnknownValue(value, []map[string]any{schema}, catalog, ToLinux, "params")
		if got := converted.(map[string]any)["extra"]; got != `C:\work` {
			t.Fatalf("extra = %q, want unchanged path", got)
		}
		if len(decisions) != 0 {
			t.Fatalf("decisions = %+v, want none", decisions)
		}
	})
}
