package transform

import (
	"net/url"
	"regexp"
	"strings"
)

func convertLocation(input string, direction Direction) (string, Decision) {
	if driveAbsolute.MatchString(input) || mountPath.MatchString(input) {
		return convertPath(input, direction)
	}
	uri, err := url.Parse(input)
	if err == nil && uri.Scheme != "" {
		return convertURI(input, direction)
	}
	return convertPath(input, direction)
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
