// Package transform rewrites documented ACP path fields between Windows
// clients and Linux agents, with optional conversion for fields outside ACP schemas.
package transform

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"
)

// Direction identifies which side of the bridge receives a transformed message.
type Direction uint8

const (
	// ToLinux transforms Windows client paths for the Linux agent.
	ToLinux Direction = iota
	// ToWindows transforms Linux agent paths for the Windows client.
	ToWindows
)

// Decision records a path transformation or an unsupported path classification.
type Decision struct{ Field, Kind, Input, Output string }

// Transformer retains only JSON-RPC routing state and the negotiated version.
type Transformer struct {
	mu            sync.Mutex
	version       int
	pending       map[string]string
	unknownFields bool
}

// New creates a Transformer for ACP protocol version 1.
// unknownFields enables path conversion in fields outside the ACP schema.
func New(unknownFields bool) *Transformer {
	return &Transformer{version: 1, pending: make(map[string]string), unknownFields: unknownFields}
}

// Transform rewrites path fields in one message.
func (t *Transformer) Transform(line []byte, direction Direction) ([]byte, []Decision) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(bytes.TrimSpace(line)) == 0 {
		return line, nil
	}
	var message any
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.UseNumber()
	if decoder.Decode(&message) != nil {
		return line, []Decision{{Kind: "malformed"}}
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return line, []Decision{{Kind: "malformed"}}
	}
	changed, decisions := t.transformValue(message, direction)
	if !changed {
		return line, decisions
	}
	encoded, err := json.Marshal(message)
	if err != nil {
		return line, append(decisions, Decision{Kind: "marshal-error"})
	}
	if bytes.HasSuffix(line, []byte("\r\n")) {
		encoded = append(encoded, '\r', '\n')
	} else if bytes.HasSuffix(line, []byte("\n")) {
		encoded = append(encoded, '\n')
	}
	return encoded, decisions
}

func (t *Transformer) transformValue(value any, direction Direction) (bool, []Decision) {
	switch value := value.(type) {
	case []any:
		changed := false
		var decisions []Decision
		for _, item := range value {
			itemChanged, itemDecisions := t.transformValue(item, direction)
			changed = changed || itemChanged
			decisions = append(decisions, itemDecisions...)
		}
		return changed, decisions
	case map[string]any:
		catalog := t.catalogFor(value)
		schemaValid := catalog.validates(value)
		if t.unknownFields && !schemaValid {
			fallbackChanged, fallbackDecisions := transformUnknownFields(value, t.pending, catalog, direction)
			changed, decisions := t.transformMessage(value, direction)
			return changed || fallbackChanged, append(fallbackDecisions, decisions...)
		}
		changed, decisions := t.transformMessage(value, direction)
		return changed, decisions
	default:
		return false, nil
	}
}

func (t *Transformer) catalogFor(message map[string]any) *schemaCatalog {
	if method, _ := message["method"].(string); method == "initialize" {
		if params, ok := message["params"].(map[string]any); ok {
			if number, ok := params["protocolVersion"].(json.Number); ok {
				if version, err := number.Int64(); err == nil && (version == 1 || version == 2) {
					return catalogs[int(version)]
				}
			}
		}
	}
	return catalogs[t.version]
}

func (t *Transformer) transformMessage(message map[string]any, direction Direction) (bool, []Decision) {
	method, _ := message["method"].(string)
	id := requestID(message["id"])
	if method != "" {
		if id != "" {
			t.pending[id] = method
		}
		if method == "initialize" {
			t.setVersion(message["params"])
		}
		params, ok := message["params"].(map[string]any)
		if !ok {
			return false, nil
		}
		return transformKnownFields(t.version, method, "params", params, direction)
	}
	if id == "" {
		return false, nil
	}
	method = t.pending[id]
	if method == "" {
		return false, nil
	}
	delete(t.pending, id)
	if method == "initialize" {
		t.setVersion(message["result"])
	}
	result, ok := message["result"].(map[string]any)
	if !ok {
		return false, nil
	}
	return transformKnownFields(t.version, method, "result", result, direction)
}

func (t *Transformer) setVersion(value any) {
	params, ok := value.(map[string]any)
	if !ok {
		return
	}
	if number, ok := params["protocolVersion"].(json.Number); ok {
		if version, err := number.Int64(); err == nil && (version == 1 || version == 2) {
			t.version = int(version)
		}
	}
}
func requestID(value any) string {
	switch id := value.(type) {
	case string:
		return "s:" + id
	case json.Number:
		return "n:" + id.String()
	}
	return ""
}
