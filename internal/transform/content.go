package transform

func transformUpdate(value any, direction Direction) (bool, []Decision) {
	update, ok := value.(map[string]any)
	if !ok {
		return false, nil
	}
	// ACP permits future update discriminators. Only the schema-defined
	// variants below are safe to inspect; unknown updates remain opaque.
	switch update["sessionUpdate"] {
	case "user_message_chunk", "agent_message_chunk", "agent_thought_chunk", "compaction_summary_chunk", "tool_call_content_chunk":
		return transformContent(update["content"], direction)
	case "user_message", "agent_message", "agent_thought":
		return transformContentList(update["content"], direction)
	case "terminal_update":
		return transformPaths(update, direction, "cwd")
	case "plan_update":
		return transformPlan(update["plan"], direction)
	case "tool_call", "tool_call_update":
		return transformToolCall(update, direction)
	}
	return false, nil
}

func transformPlan(value any, direction Direction) (bool, []Decision) {
	plan, ok := value.(map[string]any)
	if !ok || plan["type"] != "file" {
		return false, nil
	}
	return transformURIs(plan, direction, "uri")
}

func transformToolCall(value any, direction Direction) (bool, []Decision) {
	call, ok := value.(map[string]any)
	if !ok {
		return false, nil
	}
	changed, decisions := transformLocations(call["locations"], direction)
	contentChanged, contentDecisions := transformToolContentList(call["content"], direction)
	return changed || contentChanged, append(decisions, contentDecisions...)
}

func transformLocations(value any, direction Direction) (bool, []Decision) {
	locations, ok := value.([]any)
	if !ok {
		return false, nil
	}
	changed := false
	var decisions []Decision
	for _, value := range locations {
		location, ok := value.(map[string]any)
		if !ok {
			continue
		}
		itemChanged, itemDecisions := transformPaths(location, direction, "path")
		changed = changed || itemChanged
		decisions = append(decisions, itemDecisions...)
	}
	return changed, decisions
}

func transformToolContentList(value any, direction Direction) (bool, []Decision) {
	if content, ok := value.(map[string]any); ok {
		return transformToolContent(content, direction)
	}
	items, ok := value.([]any)
	if !ok {
		return false, nil
	}
	changed := false
	var decisions []Decision
	for _, item := range items {
		itemChanged, itemDecisions := transformToolContent(item, direction)
		changed = changed || itemChanged
		decisions = append(decisions, itemDecisions...)
	}
	return changed, decisions
}

func transformToolContent(value any, direction Direction) (bool, []Decision) {
	content, ok := value.(map[string]any)
	if !ok {
		return false, nil
	}
	switch content["type"] {
	case "content":
		return transformContent(content["content"], direction)
	case "diff":
		// v1 stores one path directly on the diff. v2 stores one or more
		// structured changes. Patch text is deliberately never parsed.
		changed, decisions := transformPaths(content, direction, "path")
		changes, _ := content["changes"].([]any)
		for _, value := range changes {
			change, ok := value.(map[string]any)
			if !ok {
				continue
			}
			itemChanged, itemDecisions := transformPaths(change, direction, "path", "oldPath")
			changed = changed || itemChanged
			decisions = append(decisions, itemDecisions...)
		}
		return changed, decisions
	default:
		return transformContent(value, direction)
	}
}

func transformContentList(value any, direction Direction) (bool, []Decision) {
	items, ok := value.([]any)
	if !ok {
		return false, nil
	}
	changed := false
	var decisions []Decision
	for _, item := range items {
		itemChanged, itemDecisions := transformContent(item, direction)
		changed = changed || itemChanged
		decisions = append(decisions, itemDecisions...)
	}
	return changed, decisions
}

func transformContent(value any, direction Direction) (bool, []Decision) {
	content, ok := value.(map[string]any)
	if !ok {
		return false, nil
	}
	// Content values can carry user text. Only these typed resource fields may
	// be file URIs; text and unknown content are never interpreted.
	switch content["type"] {
	case "image":
		return transformURIs(content, direction, "uri")
	case "resource_link":
		changed, decisions := transformURIs(content, direction, "uri")
		icons, _ := content["icons"].([]any)
		for _, value := range icons {
			icon, ok := value.(map[string]any)
			if !ok {
				continue
			}
			itemChanged, itemDecisions := transformURIs(icon, direction, "src")
			changed = changed || itemChanged
			decisions = append(decisions, itemDecisions...)
		}
		return changed, decisions
	case "resource":
		resource, ok := content["resource"].(map[string]any)
		if !ok {
			return false, nil
		}
		return transformURIs(resource, direction, "uri")
	}
	return false, nil
}
