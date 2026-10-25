package transform

func transformV1MCPServers(value any, direction Direction) (bool, []Decision) {
	return transformMCPServers(value, direction, false)
}

func transformV2MCPServers(value any, direction Direction) (bool, []Decision) {
	return transformMCPServers(value, direction, true)
}

func transformMCPServers(value any, direction Direction, requiresStdioType bool) (bool, []Decision) {
	servers, ok := value.([]any)
	if !ok {
		return false, nil
	}
	changed := false
	var decisions []Decision
	for _, value := range servers {
		server, ok := value.(map[string]any)
		if !ok || (requiresStdioType && server["type"] != "stdio") || (!requiresStdioType && !isV1MCPStdioServer(server)) {
			continue
		}
		itemChanged, itemDecisions := transformPaths(server, direction, "command")
		changed = changed || itemChanged
		decisions = append(decisions, itemDecisions...)
	}
	return changed, decisions
}

func isV1MCPStdioServer(server map[string]any) bool {
	_, hasName := server["name"].(string)
	_, hasCommand := server["command"].(string)
	_, hasArgs := server["args"].([]any)
	_, hasEnv := server["env"].([]any)
	return hasName && hasCommand && hasArgs && hasEnv
}
