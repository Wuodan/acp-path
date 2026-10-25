# Kickoff: `acp-path`

Create a new, standalone Go project named `acp-path`.

`acp-path` is a stdio proxy for the Agent Client Protocol (ACP). Its purpose is to make ACP path values usable
between a Windows IDE client and an agent running in Linux on WSL, including an agent started in a Docker container.

## Required behavior

- Read ACP JSON-RPC traffic from standard input and forward it to a downstream ACP command. The command and its
  arguments follow `--`, so the bridge can run either on the Windows host or in a container without changing the
  transformation core.
- Forward the downstream command's ACP output to standard output. Standard output is protocol-only: never write
  diagnostics, logging, or human-facing output there.
- Translate standard Windows drive paths, such as `C:\\work\\project`, to WSL paths, such as
  `/mnt/c/work/project`, on traffic travelling from the Windows client to the Linux agent.
- Translate `/mnt/<drive>/...` paths back to Windows drive paths on traffic travelling from the Linux agent to the
  Windows client.
- Translate ordinary relative-path separators in both directions: `src\\file.go` becomes `src/file.go` toward
  Linux, and `src/file.go` becomes `src\\file.go` toward Windows.
- Support ACP v1 and v2 unless initial protocol research establishes that one version is dead. Inventory every
  standardized structured field that represents a filesystem path or file URI in both versions, including relative
  paths. Start end-to-end work with one observed version, but make the routing explicitly version-aware so adding
  the other is a finite, tested mapping task rather than a redesign.
- Parse only known JSON-RPC methods and schema-defined fields. Never scan, interpret, or rewrite arbitrary strings,
  extension fields, prompts, chat content, or natural-language text.
- Forward unknown, malformed, unmappable, and extension data unchanged. Do not let a parser error terminate the
  proxy or alter the original line.
- Classify Windows drive-relative paths such as `C:folder\\file` and current-drive-rooted paths such as
  `\\folder\\file`, but pass them through unchanged. Their meaning depends on Windows current-drive or per-drive
  current-directory state that the proxy does not have; log their classification when diagnostics are enabled.
- Treat UNC paths, custom WSL mount roots, and paths embedded in text as deferred extensions. Do not add them to the
  first implementation.

## Development logging

This proxy needs strong observability while it is being developed. Write complete directional ACP traffic to an
append-only development log: raw inbound lines, transformed outbound lines, transformation decisions, subprocess
lifecycle events, and errors. Default the log location to an OS-appropriate user cache directory and provide a
`--log-file` override.

Keep standard output ACP-only. Mirroring diagnostics to standard error is acceptable as a temporary convenience,
but it is not a reliability mechanism: remove or disable it immediately if any IDE, WSL, Docker, or agent layer
hides, merges, or disrupts stderr. File logging is the primary diagnostic channel.

For a later product mode, design for logging to be disabled by default, with explicit opt-in minimal diagnostics and
full traffic tracing. Do not optimize the initial development build at the expense of the logs needed to understand
real ACP traffic.

## Verification

- Add focused unit tests for every inventoried v1/v2 structured path and file-URI mapping in both directions.
- Test ordinary relative-path conversion in both directions. Test drive-relative and current-drive-rooted paths for
  exact passthrough and diagnostic classification.
- Test unknown fields, extension fields, malformed input, and unmappable paths for exact passthrough.
- Test development log-file creation and the guarantee that standard output contains only downstream ACP data.
- Add an end-to-end test using a fake ACP agent subprocess to verify bidirectional forwarding, transformation, and
  subprocess lifecycle behavior.

## Boundaries

The path-mapping direction is always Windows ACP client to Linux agent, with reverse conversion for paths returned
to the Windows client. Assume standard WSL drive mounts at `/mnt/<drive>`.
