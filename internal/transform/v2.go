package transform

// transformV2Params maps only fields in the v2 inventory. The v2 schema does
// not define the v1 filesystem or terminal/create methods.
func transformV2Params(method string, params map[string]any, direction Direction) (bool, []Decision) {
	switch method {
	case "session/new", "session/resume":
		return transformPaths(params, direction, "cwd", "additionalDirectories")
	case "session/list":
		return transformPaths(params, direction, "cwd")
	case "session/prompt":
		return transformContentList(params["prompt"], direction)
	case "session/update":
		return transformUpdate(params["update"], direction)
	case "session/request_permission":
		subject, ok := params["subject"].(map[string]any)
		if !ok || subject["kind"] != "command" {
			return false, nil
		}
		return transformPaths(subject, direction, "cwd")
	}
	return false, nil
}
