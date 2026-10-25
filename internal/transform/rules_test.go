package transform

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed pathrules/known_fields.schema.json
var knownFieldRulesSchemaSource []byte

func TestKnownFieldRulesMatchSchema(t *testing.T) {
	schema := compileKnownFieldRulesSchema(t)
	for version, source := range map[int][]byte{1: knownFieldsV1Source, 2: knownFieldsV2Source} {
		var rules any
		if err := json.Unmarshal(source, &rules); err != nil {
			t.Fatal(err)
		}
		if err := schema.Validate(rules); err != nil {
			t.Errorf("v%d rules: %v", version, err)
		}
	}
}

func TestKnownFieldRulesSchemaRejectsInvalidRule(t *testing.T) {
	schema := compileKnownFieldRulesSchema(t)
	invalid := map[string]any{
		"rules": []any{map[string]any{
			"method": "session/new", "payload": "body", "selectors": []any{"cwd"}, "extra": true,
		}},
	}
	if err := schema.Validate(invalid); err == nil {
		t.Fatal("invalid rule matched schema")
	}
}

func compileKnownFieldRulesSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	var source any
	if err := json.Unmarshal(knownFieldRulesSchemaSource, &source); err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("known_fields.schema.json", source); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile("known_fields.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	return schema
}

func TestKnownFieldSelectorsAreWellFormed(t *testing.T) {
	for version, rules := range knownRules {
		for _, rule := range rules.Rules {
			for _, selector := range rule.Selectors {
				if _, ok := parseSelector(selector); !ok {
					t.Errorf("v%d invalid selector %q", version, selector)
				}
			}
		}
	}
}

func TestKnownLocationConversionKeepsOtherURIsOpaque(t *testing.T) {
	converted, _ := convertLocation("https://example.test/path", ToWindows)
	if converted != "https://example.test/path" {
		t.Fatalf("HTTP URI = %q", converted)
	}
	converted, _ = convertLocation("file:///mnt/c/work/file.go", ToWindows)
	if converted != `file:///C:/work/file.go` {
		t.Fatalf("file URI = %q", converted)
	}
}

func TestKnownFieldRuleLoadingRejectsInvalidSource(t *testing.T) {
	for _, source := range [][]byte{
		[]byte(`{"rules":`),
		[]byte(`{"rules":[{"selectors":["items["]}]}`),
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid rules did not panic")
				}
			}()
			loadKnownFieldRules(source)
		}()
	}
}

func TestParseSelectorRejectsMalformedSelectors(t *testing.T) {
	for _, selector := range []string{"", ".path", "items[", "items[]junk", "items[=x]", "items[type=]"} {
		if _, ok := parseSelector(selector); ok {
			t.Errorf("parseSelector(%q) succeeded", selector)
		}
	}
}

func TestTransformSelectedNodeSkipsUnmatchedValues(t *testing.T) {
	segments, ok := parseSelector("items[type=file].path")
	if !ok {
		t.Fatal("parse selector")
	}
	value := map[string]any{
		"items": []any{
			nil,
			map[string]any{"type": "text", "path": `C:\leave`},
			map[string]any{"type": "file", "path": 42},
			map[string]any{"type": "file", "path": "unchanged"},
			map[string]any{"type": "file", "path": `C:\work`},
		},
	}
	changed, decisions := transformSelectedNode(value, segments, ToLinux, "params")
	if !changed || len(decisions) != 1 || decisions[0].Field != "params.items[4].path" {
		t.Fatalf("changed=%v decisions=%+v", changed, decisions)
	}
	items := value["items"].([]any)
	if items[1].(map[string]any)["path"] != `C:\leave` || items[4].(map[string]any)["path"] != "/mnt/c/work" {
		t.Fatalf("items = %#v", items)
	}

	for _, candidate := range []any{nil, map[string]any{}, map[string]any{"items": "not an array"}} {
		if changed, _ := transformSelectedNode(candidate, segments, ToLinux, "params"); changed {
			t.Fatalf("unexpected conversion for %#v", candidate)
		}
	}
}

func TestTransformSelectedNodeHandlesArrayLeavesAndObjectPredicates(t *testing.T) {
	arraySelector, ok := parseSelector("roots[]")
	if !ok {
		t.Fatal("parse array selector")
	}
	value := map[string]any{"roots": []any{42, "relative", `C:\work`}}
	changed, _ := transformSelectedNode(value, arraySelector, ToLinux, "params")
	if !changed || value["roots"].([]any)[2] != "/mnt/c/work" {
		t.Fatalf("roots = %#v", value["roots"])
	}

	predicateSelector, ok := parseSelector("plan[type=file].uri")
	if !ok {
		t.Fatal("parse predicate selector")
	}
	for _, value := range []map[string]any{
		{"plan": map[string]any{"type": "text", "uri": "file:///C:/leave"}},
		{"plan": "not an object"},
	} {
		if changed, _ := transformSelectedNode(value, predicateSelector, ToLinux, "params"); changed {
			t.Fatalf("unexpected conversion for %#v", value)
		}
	}
}
