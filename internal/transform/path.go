package transform

import (
	"net/url"
	"regexp"
	"strings"
)

func transformPaths(object map[string]any, direction Direction, fields ...string) (bool, []Decision) {
	changed := false
	var decisions []Decision
	for _, field := range fields {
		switch value := object[field].(type) {
		case string:
			converted, decision := convertPath(value, direction)
			if decision.Kind != "" {
				decision.Field = field
				decisions = append(decisions, decision)
			}
			if converted != value {
				object[field] = converted
				changed = true
			}
		case []any:
			for index, entry := range value {
				text, ok := entry.(string)
				if !ok {
					continue
				}
				converted, decision := convertPath(text, direction)
				if decision.Kind != "" {
					decision.Field = field
					decisions = append(decisions, decision)
				}
				if converted != text {
					value[index] = converted
					changed = true
				}
			}
		}
	}
	return changed, decisions
}

func transformURIs(object map[string]any, direction Direction, fields ...string) (bool, []Decision) {
	changed := false
	var decisions []Decision
	for _, field := range fields {
		raw, ok := object[field].(string)
		if !ok {
			continue
		}
		converted, decision := convertURI(raw, direction)
		if decision.Kind != "" {
			decision.Field = field
			decisions = append(decisions, decision)
		}
		if converted != raw {
			object[field] = converted
			changed = true
		}
	}
	return changed, decisions
}

var driveAbsolute = regexp.MustCompile(`^[A-Za-z]:[\\/]`)
var driveRelative = regexp.MustCompile(`^[A-Za-z]:[^\\/]`)
var mountPath = regexp.MustCompile(`^/mnt/([A-Za-z])(?:/(.*))?$`)

func convertPath(input string, direction Direction) (string, Decision) {
	if direction == ToLinux {
		if driveAbsolute.MatchString(input) {
			output := "/mnt/" + strings.ToLower(input[:1]) + "/" + strings.ReplaceAll(input[3:], "\\", "/")
			return output, Decision{Kind: "windows-drive", Input: input, Output: output}
		}
		if driveRelative.MatchString(input) {
			// Its meaning depends on the current directory for this drive.
			return input, Decision{Kind: "windows-drive-relative", Input: input}
		}
		if strings.HasPrefix(input, `\\`) {
			// UNC conversion is deferred because the bridge has no mount mapping.
			return input, Decision{Kind: "unc", Input: input}
		}
		if strings.HasPrefix(input, `\`) {
			// Its meaning depends on the current Windows drive.
			return input, Decision{Kind: "windows-current-drive-rooted", Input: input}
		}
		if !strings.HasPrefix(input, "/") && strings.Contains(input, `\`) {
			output := strings.ReplaceAll(input, `\`, "/")
			return output, Decision{Kind: "relative", Input: input, Output: output}
		}
		return input, Decision{}
	}
	if match := mountPath.FindStringSubmatch(input); match != nil {
		output := strings.ToUpper(match[1]) + `:\`
		if match[2] != "" {
			output += strings.ReplaceAll(match[2], "/", `\`)
		}
		return output, Decision{Kind: "wsl-mount", Input: input, Output: output}
	}
	if !strings.HasPrefix(input, "/") && strings.Contains(input, "/") {
		output := strings.ReplaceAll(input, "/", `\`)
		return output, Decision{Kind: "relative", Input: input, Output: output}
	}
	return input, Decision{}
}

func convertURI(input string, direction Direction) (string, Decision) {
	// A non-file URI is an identifier, not a filesystem path for this bridge.
	uri, err := url.Parse(input)
	if err != nil || uri.Scheme != "file" || uri.Host != "" {
		return input, Decision{}
	}
	path := uri.Path
	if direction == ToLinux {
		path = strings.TrimPrefix(path, "/")
	}
	converted, decision := convertPath(path, direction)
	if decision.Kind == "" {
		return input, decision
	}
	if direction == ToLinux {
		uri.Path = converted
	} else {
		uri.Path = "/" + strings.ReplaceAll(converted, `\`, "/")
	}
	output := uri.String()
	decision.Kind = "file-uri"
	decision.Input = input
	decision.Output = output
	return output, decision
}
