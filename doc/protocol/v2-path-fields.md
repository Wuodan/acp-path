# ACP v2 filesystem field inventory

This inventory is derived from the ACP v2 schema in the
`agentclientprotocol/agent-client-protocol` repository. The v2 router is
separate from v1 because v2 changed tool-call diffs and terminal reporting.

| Context | Field |
| --- | --- |
| `session/new`, `session/resume` | `params.cwd`, `params.additionalDirectories[]` |
| `session/list` request | `params.cwd` |
| `session/list` response | `result.sessions[].cwd`, `result.sessions[].additionalDirectories[]` |
| `session/prompt` | typed resource/image file URIs |
| `session/request_permission` | `params.subject.cwd` for a `command` subject |
| `session/update` terminal update | `update.cwd` |
| `session/update` tool-call update | `locations[].path`; diff `changes[].path` and `changes[].oldPath`; typed content file URIs |

`file:` URIs in typed resource, image, and resource-link icon fields are
converted. HTTP URLs, elicitation URLs, MCP URLs, extension fields, raw tool
input/output, and all unknown tagged-union values are forwarded unchanged.
