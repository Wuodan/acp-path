package transform

func (t *Transformer) transformParams(method string, params map[string]any, direction Direction) (bool, []Decision) {
	if t.version == 1 {
		return transformV1Params(method, params, direction)
	}
	return transformV2Params(method, params, direction)
}

func transformSessionList(result map[string]any, direction Direction) (bool, []Decision) {
	changed := false
	var decisions []Decision
	items, _ := result["sessions"].([]any)
	for _, item := range items {
		if session, ok := item.(map[string]any); ok {
			itemChanged, itemDecisions := transformPaths(session, direction, "cwd", "additionalDirectories")
			changed = changed || itemChanged
			decisions = append(decisions, itemDecisions...)
		}
	}
	return changed, decisions
}
