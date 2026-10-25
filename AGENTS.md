# Engineering guidelines for `acp-wsl`

## Product content

- Repository files contain product-relevant source, documentation, configuration, and durable engineering
  instructions only.
- Do not add conversational feedback, implementation-status commentary, user-directed instructions, or language
  intended to demonstrate compliance with a chat request to repository files.
- Keep agent instructions factual and reusable. Do not embed session-specific context or reassurance.

## Design and package boundaries

- Prefer small, cohesive packages and files with one clear responsibility. Split code when a file becomes difficult
  to navigate rather than accumulating unrelated helpers.
- Keep process orchestration, ACP message routing, versioned schema mappings, path conversion, and logging behind
  narrow package boundaries.
- Avoid cyclic dependencies, broad utility packages, and packages that become catch-alls for unrelated behavior.
- Do not introduce abstractions, configuration, or public APIs before a concrete caller requires them.
- Make path classification explicit and side-effect-free. Pass unsupported path forms through unchanged and diagnose
  them when logging is enabled; never guess or silently resolve their meaning.

## Go visibility and code quality

- Follow Go visibility idioms strictly: identifiers are lowercase and package-private by default. Export an
  uppercase identifier only when another package genuinely needs it.
- Do not export convenience APIs, types, constants, configuration, or helpers merely for possible future use.
- Use clear, descriptive names and methods with one coherent purpose. Keep methods short enough to understand
  without reconstructing distant control flow.
- Use explicit types where they clarify a protocol or process boundary. Handle errors directly and add context that
  identifies the failed operation.
- Avoid speculative defensive code for impossible states. Prefer the smallest direct implementation that satisfies
  the supported protocol behavior.

## Tests

- Keep test package and directory structure recognizable relative to the production package being tested.
- Name tests after the production behavior they verify. Keep version-specific protocol fixtures beside their related
  versioned mappings, rather than in an unrelated shared fixture dump.
- Use table-driven unit tests for path and message mappings where cases differ only by input and expectation.
- Include explicit end-to-end tests for stream forwarding and subprocess lifecycle behavior.
- Preserve exact passthrough behavior in tests for unsupported, unknown, malformed, and invalid traffic.

## Protocol streams and logging

- Treat standard output as ACP protocol-only. No logs, diagnostics, progress messages, or human-facing text may
  enter that stream.
- During development, log complete directional raw traffic to the configured file so transformation behavior can be
  inspected and reproduced.
- Standard error is optional convenience output only. If it is swallowed, merged, or interferes with a surrounding
  stdio chain, disable it without reducing file-log observability.
- Do not change protocol support, path-mapping rules, command-line behavior, or logging defaults without matching
  tests and a documented reason.

## Git workflow

- Leave worktree changes uncommitted for review by default.
- Never commit, amend, push, rebase, reset, or otherwise alter Git history unless explicitly instructed to perform
  that specific action in the current request.
- A request to implement a change does not authorize creating or modifying commits.
