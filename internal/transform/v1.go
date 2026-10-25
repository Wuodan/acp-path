package transform

// transformV1Params maps only fields in the v1 inventory. Unlisted methods
// and extension data remain opaque to the bridge.
func transformV1Params(method string, params map[string]any, direction Direction) (bool, []Decision) {
	switch method {
	case "session/new", "session/load", "session/resume":
		return transformPaths(params, direction, "cwd", "additionalDirectories")
	case "session/list":
		return transformPaths(params, direction, "cwd")
	case "session/prompt":
		return transformContentList(params["prompt"], direction)
	case "fs/read_text_file", "fs/write_text_file":
		return transformPaths(params, direction, "path")
	case "terminal/create":
		return transformPaths(params, direction, "cwd")
	case "session/update":
		return transformUpdate(params["update"], direction)
	case "session/request_permission":
		return transformToolCall(params["toolCall"], direction)
	}
	return false, nil
}
