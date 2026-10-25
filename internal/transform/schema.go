package transform

import (
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed acpschema/v1.json
var schemaV1 []byte

//go:embed acpschema/v2.json
var schemaV2 []byte

type schemaCatalog struct {
	root      map[string]any
	defs      map[string]any
	params    map[string][]map[string]any
	results   map[string][]map[string]any
	validator *jsonschema.Schema
}

var catalogs = map[int]*schemaCatalog{
	1: newSchemaCatalog(schemaV1),
	2: newSchemaCatalog(schemaV2),
}

func newSchemaCatalog(source []byte) *schemaCatalog {
	var root map[string]any
	if err := json.Unmarshal(source, &root); err != nil {
		panic(err)
	}
	defs, _ := root["$defs"].(map[string]any)
	catalog := &schemaCatalog{
		root:    root,
		defs:    defs,
		params:  make(map[string][]map[string]any),
		results: make(map[string][]map[string]any),
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", root); err != nil {
		panic(err)
	}
	validator, err := compiler.Compile("schema.json")
	if err != nil {
		panic(err)
	}
	catalog.validator = validator
	for name, value := range defs {
		schema, ok := value.(map[string]any)
		if !ok {
			continue
		}
		method, _ := schema["x-method"].(string)
		if method == "" {
			continue
		}
		switch {
		case strings.HasSuffix(name, "Request") || strings.HasSuffix(name, "Notification"):
			catalog.params[method] = append(catalog.params[method], schema)
		case strings.HasSuffix(name, "Response"):
			catalog.results[method] = append(catalog.results[method], schema)
		}
	}
	return catalog
}

func (catalog *schemaCatalog) validates(message map[string]any) bool {
	return catalog.validator.Validate(message) == nil
}

func (catalog *schemaCatalog) resolve(schema map[string]any) []map[string]any {
	if ref, _ := schema["$ref"].(string); strings.HasPrefix(ref, "#/$defs/") {
		name := strings.TrimPrefix(ref, "#/$defs/")
		if resolved, ok := catalog.defs[name].(map[string]any); ok {
			return catalog.resolve(resolved)
		}
	}
	resolved := []map[string]any{schema}
	for _, keyword := range []string{"allOf", "anyOf", "oneOf"} {
		items, _ := schema[keyword].([]any)
		for _, item := range items {
			if child, ok := item.(map[string]any); ok {
				resolved = append(resolved, catalog.resolve(child)...)
			}
		}
	}
	return resolved
}
