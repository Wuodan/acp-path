# ACP v1 filesystem field inventory

This inventory is derived from the ACP v1 schema in the `agentclientprotocol/agent-client-protocol` repository.

It defines the schema-specific mappings. Other payload values are inspected by the default
[unknown fields](path-field-inventory.md#unknown-fields), which can be disabled.

## Client to agent

<!-- markdownlint-disable line-length -->
| Method                                          | Field                                                                             |
|-------------------------------------------------|-----------------------------------------------------------------------------------|
| `session/new`                                   | `params.cwd`, `params.additionalDirectories[]`                                    |
| `session/load`                                  | `params.cwd`, `params.additionalDirectories[]`                                    |
| `session/resume`                                | `params.cwd`, `params.additionalDirectories[]`                                    |
| `session/new`, `session/load`, `session/resume` | `params.mcpServers[]` stdio-server `command`                                      |
| `session/list`                                  | `params.cwd`                                                                      |
| `session/prompt`                                | `params.prompt[]` resource and image `uri` fields when they use the `file` scheme |
<!-- markdownlint-enable line-length -->

## Agent to client

| Method                       | Field                                                            |
|------------------------------|------------------------------------------------------------------|
| `fs/read_text_file`          | `params.path`                                                    |
| `fs/write_text_file`         | `params.path`                                                    |
| `terminal/create`            | `params.cwd`                                                     |
| `session/update`             | tool-call locations, v1 diff `path`, and typed content file URIs |
| `session/request_permission` | tool-call locations, v1 diff `path`, and typed content file URIs |

## Responses

| Response to    | Field                                                                |
|----------------|----------------------------------------------------------------------|
| `session/list` | `result.sessions[].cwd`, `result.sessions[].additionalDirectories[]` |

Windows drive-relative paths, current-drive-rooted paths, and UNC paths are classified but not rewritten.

The required drive/current-directory context is not available to the bridge.

Non-`file` URIs are not filesystem paths and are therefore left unchanged.
