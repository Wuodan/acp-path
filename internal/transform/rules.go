package transform

import (
	_ "embed"
	"encoding/json"
	"strconv"
	"strings"
)

//go:embed pathrules/v1.json
var knownFieldsV1Source []byte

//go:embed pathrules/v2.json
var knownFieldsV2Source []byte

type knownFieldRules struct {
	Rules []knownFieldRule `json:"rules"`
}

type knownFieldRule struct {
	Method    string   `json:"method"`
	Payload   string   `json:"payload"`
	Selectors []string `json:"selectors"`
}

type selectorSegment struct {
	name, predicate, expected string
	array                     bool
}

var knownRules = map[int]knownFieldRules{
	1: loadKnownFieldRules(knownFieldsV1Source),
	2: loadKnownFieldRules(knownFieldsV2Source),
}

func loadKnownFieldRules(source []byte) knownFieldRules {
	var rules knownFieldRules
	if err := json.Unmarshal(source, &rules); err != nil {
		panic(err)
	}
	for _, rule := range rules.Rules {
		for _, selector := range rule.Selectors {
			if _, ok := parseSelector(selector); !ok {
				panic("invalid known field selector: " + selector)
			}
		}
	}
	return rules
}

func transformKnownFields(version int, method, payload string, value map[string]any, direction Direction) (bool, []Decision) {
	changed := false
	var decisions []Decision
	for _, rule := range knownRules[version].Rules {
		if rule.Method != method || rule.Payload != payload {
			continue
		}
		for _, selector := range rule.Selectors {
			segments, _ := parseSelector(selector)
			selectorChanged, selectorDecisions := transformSelectedNode(value, segments, direction, payload)
			changed = changed || selectorChanged
			decisions = append(decisions, selectorDecisions...)
		}
	}
	return changed, decisions
}

func parseSelector(selector string) ([]selectorSegment, bool) {
	parts := strings.Split(selector, ".")
	segments := make([]selectorSegment, 0, len(parts))
	for _, part := range parts {
		segment := selectorSegment{name: part}
		if open := strings.IndexByte(part, '['); open >= 0 {
			segment.name = part[:open]
			rest := part[open:]
			if strings.HasPrefix(rest, "[]") {
				segment.array, rest = true, rest[2:]
			}
			if rest == "" {
				segments = append(segments, segment)
				continue
			}
			if !strings.HasPrefix(rest, "[") || !strings.HasSuffix(rest, "]") {
				return nil, false
			}
			key, expected, found := strings.Cut(strings.TrimSuffix(rest[1:], "]"), "=")
			if !found || key == "" || expected == "" {
				return nil, false
			}
			segment.predicate, segment.expected = key, expected
		}
		if segment.name == "" {
			return nil, false
		}
		segments = append(segments, segment)
	}
	return segments, true
}

func transformSelectedNode(value any, segments []selectorSegment, direction Direction, field string) (bool, []Decision) {
	object, ok := value.(map[string]any)
	if !ok || len(segments) == 0 {
		return false, nil
	}
	segment := segments[0]
	child, ok := object[segment.name]
	if !ok {
		return false, nil
	}
	childField := field + "." + segment.name
	if segment.array || segment.predicate != "" {
		if items, isArray := child.([]any); isArray {
			changed := false
			var decisions []Decision
			for index, item := range items {
				if segment.predicate != "" && !matchesPredicate(item, segment) {
					continue
				}
				if len(segments) == 1 {
					text, ok := item.(string)
					if !ok {
						continue
					}
					converted, decision := convertLocation(text, direction)
					if converted == text {
						continue
					}
					decision.Field = childField + "[" + strconv.Itoa(index) + "]"
					items[index] = converted
					changed = true
					decisions = append(decisions, decision)
					continue
				}
				itemChanged, itemDecisions := transformSelectedNode(item, segments[1:], direction, childField+"["+strconv.Itoa(index)+"]")
				changed = changed || itemChanged
				decisions = append(decisions, itemDecisions...)
			}
			return changed, decisions
		}
		if segment.array || !matchesPredicate(child, segment) {
			return false, nil
		}
	}
	if len(segments) == 1 {
		text, ok := child.(string)
		if !ok {
			return false, nil
		}
		converted, decision := convertLocation(text, direction)
		if converted == text {
			return false, nil
		}
		decision.Field = childField
		object[segment.name] = converted
		return true, []Decision{decision}
	}
	return transformSelectedNode(child, segments[1:], direction, childField)
}

func matchesPredicate(value any, segment selectorSegment) bool {
	object, ok := value.(map[string]any)
	if !ok {
		return false
	}
	actual, exists := object[segment.predicate]
	if !exists || segment.expected == "*" {
		return exists
	}
	for _, expected := range strings.Split(segment.expected, "|") {
		if actual == expected {
			return true
		}
	}
	return false
}
