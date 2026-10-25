package transform

// transformV2Params maps only fields in the v2 inventory. The v2 schema does
// not define the v1 filesystem or terminal/create methods.
func transformV2Params(method string, params map[string]any, direction Direction) (bool, []Decision) {
	switch method {
	case "session/new", "session/resume", "session/fork":
		return transformV2SessionPaths(params, direction)
	case "session/list":
		return transformPaths(params, direction, "cwd")
	case "session/prompt":
		return transformContentList(params["prompt"], direction)
	case "session/update":
		return transformUpdate(params["update"], direction)
	case "session/request_permission":
		subject, ok := params["subject"].(map[string]any)
		if !ok || subject["type"] != "command" {
			return false, nil
		}
		return transformPaths(subject, direction, "cwd")
	}
	return false, nil
}

func transformV2SessionPaths(params map[string]any, direction Direction) (bool, []Decision) {
	changed, decisions := transformPaths(params, direction, "cwd", "additionalDirectories")
	mcpChanged, mcpDecisions := transformV2MCPServers(params["mcpServers"], direction)
	return changed || mcpChanged, append(decisions, mcpDecisions...)
}
